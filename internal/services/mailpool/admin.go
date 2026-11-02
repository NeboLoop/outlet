package mailpool

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/events"
)

// CreateDomain adds a pool-only sending domain.
func (s *Service) CreateDomain(ctx context.Context, domain string) (db.PoolDomain, error) {
	if err := s.on(); err != nil {
		return db.PoolDomain{}, err
	}
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" || strings.ContainsAny(domain, "@ /") || !strings.Contains(domain, ".") {
		return db.PoolDomain{}, errorx.NewBadRequestError("domain must be a domain name")
	}
	d, err := s.store.CreatePoolDomain(ctx, db.CreatePoolDomainParams{ID: newID(), Domain: domain, CreatedAt: fmtTime(s.now())})
	return d, conflict(err, "domain is already in the pool")
}

// conflict answers a unique-key violation as 409.
func conflict(err error, msg string) error {
	if err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return errorx.NewCodeError(http.StatusConflict, msg)
	}
	return err
}

// ListDomains lists every pool domain.
func (s *Service) ListDomains(ctx context.Context) ([]db.PoolDomain, error) {
	if err := s.on(); err != nil {
		return nil, err
	}
	return s.store.ListPoolDomains(ctx)
}

// MailboxInput is a mailbox an admin adds to the pool.
type MailboxInput struct {
	DomainID      string
	Address       string
	SMTPHost      string
	SMTPPort      int
	SMTPUsername  string
	SMTPPassword  string
	WarmStartedAt string // optional, RFC 3339 or YYYY-MM-DD; default now
}

// CreateMailbox adds a mailbox on a pool domain; its password is stored
// encrypted with the crypto service.
func (s *Service) CreateMailbox(ctx context.Context, in MailboxInput) (db.PoolMailbox, error) {
	if err := s.on(); err != nil {
		return db.PoolMailbox{}, err
	}
	if s.crypto == nil {
		return db.PoolMailbox{}, errorx.NewInternalError("ENCRYPTION_KEY is not set: mailbox passwords can't be stored")
	}
	if in.DomainID == "" || in.Address == "" || in.SMTPHost == "" || in.SMTPPort <= 0 || in.SMTPUsername == "" || in.SMTPPassword == "" {
		return db.PoolMailbox{}, errorx.NewBadRequestError("domain_id, address, smtp_host, smtp_port, smtp_username and smtp_password are required")
	}
	d, err := s.store.GetPoolDomain(ctx, in.DomainID)
	if errors.Is(err, sql.ErrNoRows) {
		return db.PoolMailbox{}, errorx.NewNotFoundError("pool domain not found")
	} else if err != nil {
		return db.PoolMailbox{}, err
	}
	address := strings.ToLower(strings.TrimSpace(in.Address))
	if domainOf(address) != d.Domain || strings.Count(address, "@") != 1 {
		return db.PoolMailbox{}, errorx.NewBadRequestError("address must be on the domain " + d.Domain)
	}
	now := s.now()
	warm := now
	if in.WarmStartedAt != "" {
		if warm, err = parseDate(in.WarmStartedAt); err != nil {
			return db.PoolMailbox{}, errorx.NewBadRequestError("warm_started_at must be RFC 3339 or YYYY-MM-DD")
		}
	}
	enc, err := s.crypto.EncryptString(in.SMTPPassword)
	if err != nil {
		return db.PoolMailbox{}, err
	}
	mb, err := s.store.CreatePoolMailbox(ctx, db.CreatePoolMailboxParams{
		ID: newID(), DomainID: d.ID, Address: address, SmtpHost: in.SMTPHost, SmtpPort: int64(in.SMTPPort),
		SmtpUsername: in.SMTPUsername, SmtpPasswordEnc: enc, WarmStartedAt: fmtTime(warm), CreatedAt: fmtTime(now),
	})
	return mb, conflict(err, "mailbox is already in the pool")
}

func parseDate(v string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", v)
}

// ListMailboxes lists every pool mailbox.
func (s *Service) ListMailboxes(ctx context.Context) ([]db.PoolMailbox, error) {
	if err := s.on(); err != nil {
		return nil, err
	}
	return s.store.ListPoolMailboxes(ctx)
}

// SetMailboxStatus pauses (status paused), resumes (active) or retires
// (retired) a mailbox. A resumed mailbox's health windows start fresh.
func (s *Service) SetMailboxStatus(ctx context.Context, id, status, reason string) (db.PoolMailbox, error) {
	if err := s.on(); err != nil {
		return db.PoolMailbox{}, err
	}
	mb, err := s.store.GetPoolMailbox(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return db.PoolMailbox{}, errorx.NewNotFoundError("pool mailbox not found")
	} else if err != nil {
		return db.PoolMailbox{}, err
	}
	now := s.now()
	switch status {
	case "paused":
		if reason == "" {
			reason = "paused by an admin"
		}
		err = s.pause(ctx, mb, "admin", reason, now)
	case "active":
		var n int64
		n, err = s.store.ResumePoolMailbox(ctx, db.ResumePoolMailboxParams{HealthSince: fmtTime(now), ID: id})
		if err == nil && n > 0 {
			s.closeSMTP(id) // the next send signs in afresh
			s.emit(events.TopicPoolMailboxResumed, events.PoolEvent{
				OrgID: mb.TenantOrgID.String, Tenant: mb.Tenant.String, MailboxID: mb.ID, Reason: "resumed by an admin",
			})
		}
	case "retired":
		_, err = s.store.RetirePoolMailbox(ctx, id)
		s.closeSMTP(id)
	default:
		return db.PoolMailbox{}, errorx.NewBadRequestError("status must be active, paused or retired")
	}
	if err != nil {
		return db.PoolMailbox{}, err
	}
	return s.store.GetPoolMailbox(ctx, id)
}

// SetDomainStatus pauses, resumes or retires a domain; the tenants of its
// active mailboxes hear that they stopped or send again.
func (s *Service) SetDomainStatus(ctx context.Context, id, status, reason string) (db.PoolDomain, error) {
	if err := s.on(); err != nil {
		return db.PoolDomain{}, err
	}
	if _, err := s.store.GetPoolDomain(ctx, id); errors.Is(err, sql.ErrNoRows) {
		return db.PoolDomain{}, errorx.NewNotFoundError("pool domain not found")
	} else if err != nil {
		return db.PoolDomain{}, err
	}
	var n int64
	var err error
	switch status {
	case "paused":
		if reason == "" {
			reason = "paused by an admin"
		}
		if n, err = s.store.PausePoolDomain(ctx, db.PausePoolDomainParams{PausedReason: nullString(reason), ID: id}); err == nil && n > 0 {
			err = s.announceDomain(ctx, id, events.TopicPoolMailboxPaused, "its domain is paused: "+reason)
		}
	case "active":
		if n, err = s.store.ResumePoolDomain(ctx, id); err == nil && n > 0 {
			err = s.announceDomain(ctx, id, events.TopicPoolMailboxResumed, "its domain sends again")
		}
	case "retired":
		_, err = s.store.RetirePoolDomain(ctx, id)
	default:
		return db.PoolDomain{}, errorx.NewBadRequestError("status must be active, paused or retired")
	}
	if err != nil {
		return db.PoolDomain{}, err
	}
	return s.store.GetPoolDomain(ctx, id)
}

// ClearTenantHold lifts a repeat-pause hold once the tenant's list is clean.
func (s *Service) ClearTenantHold(ctx context.Context, orgID, tenant string) (int, error) {
	if err := s.on(); err != nil {
		return 0, err
	}
	if orgID == "" || tenant == "" {
		return 0, errorx.NewBadRequestError("org_id and tenant are required")
	}
	n, err := s.store.ClearTenantPoolPauses(ctx, db.ClearTenantPoolPausesParams{
		ClearedAt: nullString(fmtTime(s.now())), OrgID: nullString(orgID), Tenant: nullString(tenant),
	})
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, errorx.NewNotFoundError("tenant has no pauses to clear")
	}
	return int(n), nil
}
