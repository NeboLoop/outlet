// Package validation runs email validation for an org: the network check (emailval,
// cached instance-wide), overlaid with deliverability signals from every org on this
// instance and from the Outlet Signal Network, recorded as usage and on the org's contact.
package validation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/config"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/emailval"
	"github.com/outlet-sh/outlet/internal/services/signalnet"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

// TimeLayout is how SQLite datetime('now') renders timestamps; stored times use it too.
const TimeLayout = "2006-01-02 15:04:05"

// Service validates addresses. Safe for concurrent use.
type Service struct {
	store   *db.Store
	cfg     config.ValidationConfig
	network *signalnet.Client // nil = Signal Network disabled
}

// New creates the validation service. network may be nil.
func New(store *db.Store, cfg config.ValidationConfig, network *signalnet.Client) *Service {
	return &Service{store: store, cfg: cfg, network: network}
}

// Validate checks one address for an org and records it: an email_validations row
// (every check is metered, cached or not) and the org's contact validation columns.
// source is one of api, admin, mcp.
func (s *Service) Validate(ctx context.Context, orgID, email, source string) (*types.ValidationResult, error) {
	res := s.Check(ctx, email)
	row := Row(res)
	row.ID = uuid.New().String()
	row.OrgID = orgID
	row.Email = strings.TrimSpace(email)
	row.Source = source
	if err := s.store.InsertEmailValidation(ctx, db.InsertEmailValidationParams{
		ID: row.ID, OrgID: orgID, Email: row.Email, Status: row.Status, SubStatus: row.SubStatus,
		FreeEmail: row.FreeEmail, DidYouMean: row.DidYouMean, Account: row.Account, Domain: row.Domain,
		DomainAgeDays: row.DomainAgeDays, SmtpProvider: row.SmtpProvider, MxFound: row.MxFound,
		MxRecord: row.MxRecord, Source: source,
	}); err != nil {
		return nil, err
	}
	s.UpdateContact(ctx, orgID, row.Email, res)
	row.ProcessedAt = sql.NullString{String: time.Now().UTC().Format(TimeLayout), Valid: true}
	out := ToType(row)
	return &out, nil
}

// Check returns the final verdict for an address without recording anything:
// cached or fresh network result, then instance and Signal Network overlays.
func (s *Service) Check(ctx context.Context, email string) *emailval.Result {
	res := s.probe(ctx, email)
	return Overlay(res, s.localSignals(ctx, res), s.networkSignal(ctx, res), time.Now())
}

// UpdateContact writes the verdict onto the org's contact with this address, if any.
func (s *Service) UpdateContact(ctx context.Context, orgID, email string, res *emailval.Result) {
	if err := s.store.UpdateContactValidation(ctx, db.UpdateContactValidationParams{
		Status:    sql.NullString{String: res.Status, Valid: true},
		SubStatus: sql.NullString{String: res.SubStatus, Valid: true},
		OrgID:     sql.NullString{String: orgID, Valid: true},
		Email:     email,
	}); err != nil {
		logx.WithContext(ctx).Errorf("validation: update contact %s: %v", email, err)
	}
}

// probe returns the network-derived result from the cache, or runs the engine and
// caches definitive answers (anything but unknown).
func (s *Service) probe(ctx context.Context, email string) *emailval.Result {
	key := strings.ToLower(strings.TrimSpace(email))
	if s.cfg.CacheDays > 0 {
		freshAfter := time.Now().UTC().AddDate(0, 0, -s.cfg.CacheDays).Format(TimeLayout)
		cached, err := s.store.GetValidationCache(ctx, db.GetValidationCacheParams{EmailLower: key, FreshAfter: freshAfter})
		if err == nil {
			var res emailval.Result
			if json.Unmarshal([]byte(cached), &res) == nil {
				return &res
			}
		} else if !errors.Is(err, sql.ErrNoRows) {
			logx.WithContext(ctx).Errorf("validation: cache lookup: %v", err)
		}
	}

	res, err := emailval.Validate(ctx, email, s.options())
	if err != nil {
		logx.WithContext(ctx).Errorf("validation: engine %s: %v", key, err)
	}
	if res == nil {
		res = &emailval.Result{Input: email, Normalized: key, Status: emailval.StatusUnknown, SubStatus: emailval.SubExceptionOccurred}
	}

	if s.cfg.CacheDays > 0 && res.Status != emailval.StatusUnknown && key != "" {
		if b, err := json.Marshal(res); err == nil {
			if err := s.store.UpsertValidationCache(ctx, db.UpsertValidationCacheParams{EmailLower: key, Result: string(b)}); err != nil {
				logx.WithContext(ctx).Errorf("validation: cache store: %v", err)
			}
		}
	}
	return res
}

func (s *Service) options() *emailval.Options {
	return &emailval.Options{
		Timeout:         20 * time.Second,
		CheckSMTP:       s.cfg.SMTPCheck,
		CheckDisposable: true,
		CheckDomainAge:  true,
		Proxies:         s.cfg.Proxies,
		SMTP: &emailval.SMTPConfig{
			HeloHostname: s.cfg.HeloHostname,
			FromDomain:   s.cfg.FromDomain,
			MaxRetries:   2,
		},
	}
}

// localSignals gathers what this instance (all orgs) knows about the address.
func (s *Service) localSignals(ctx context.Context, res *emailval.Result) LocalSignals {
	var sig LocalSignals
	email := res.Normalized
	if email == "" || !res.SyntaxOK {
		return sig
	}
	log := logx.WithContext(ctx)

	if n, err := s.store.HasEmailComplaint(ctx, email); err == nil {
		sig.Complaint = n > 0
	} else {
		log.Errorf("validation: complaint lookup: %v", err)
	}
	if n, err := s.store.IsEmailGloballySuppressed(ctx, email); err == nil {
		sig.Suppressed = n > 0
	} else {
		log.Errorf("validation: suppression lookup: %v", err)
	}
	if e, err := s.store.GetEmailEngagement(ctx, email); err == nil {
		sig.Sends = int(e.Sends)
		sig.LastEngagedAt = latest(parseTime(e.LastOpenedAt), parseTime(e.LastClickedAt))
	} else {
		log.Errorf("validation: engagement lookup: %v", err)
	}
	// Domain history is a full scan; only needed to promote catch-alls.
	if isCatchAll(res) && res.Domain != "" {
		if d, err := s.store.GetDomainDeliveryStats(ctx, res.Domain); err == nil {
			sig.DomainDelivered, sig.DomainBounced = int(d.Delivered), int(d.Bounced)
		} else {
			log.Errorf("validation: domain stats: %v", err)
		}
	}
	return sig
}

func (s *Service) networkSignal(ctx context.Context, res *emailval.Result) *signalnet.Signal {
	if s.network == nil || res.Normalized == "" || !res.SyntaxOK {
		return nil
	}
	sig, err := s.network.Lookup(ctx, res.Normalized)
	if err != nil {
		logx.WithContext(ctx).Errorf("validation: signal network lookup: %v", err)
		return nil
	}
	return sig
}

// Row maps a result to an email_validations row (identity, source and timestamps unset).
func Row(res *emailval.Result) db.EmailValidation {
	row := db.EmailValidation{
		EmailLower:   res.Normalized,
		Status:       res.Status,
		SubStatus:    res.SubStatus,
		FreeEmail:    boolInt(res.FreeEmail),
		DidYouMean:   res.DidYouMean,
		Account:      res.Account,
		Domain:       res.Domain,
		SmtpProvider: res.SMTPProvider,
		MxFound:      boolInt(res.HasMX),
		MxRecord:     res.MXRecord,
	}
	if res.DomainAgeDays > 0 {
		row.DomainAgeDays = sql.NullInt64{Int64: int64(res.DomainAgeDays), Valid: true}
	}
	return row
}

// ToType maps an email_validations row to the API type.
func ToType(row db.EmailValidation) types.ValidationResult {
	return types.ValidationResult{
		Address:       row.Email,
		Status:        row.Status,
		SubStatus:     row.SubStatus,
		FreeEmail:     row.FreeEmail == 1,
		DidYouMean:    row.DidYouMean,
		Account:       row.Account,
		Domain:        row.Domain,
		DomainAgeDays: int(row.DomainAgeDays.Int64),
		SmtpProvider:  row.SmtpProvider,
		MxFound:       row.MxFound == 1,
		MxRecord:      row.MxRecord,
		ProcessedAt:   row.ProcessedAt.String,
	}
}

func boolInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func parseTime(s string) time.Time {
	for _, layout := range []string{TimeLayout, time.RFC3339Nano, "2006-01-02T15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func latest(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}
