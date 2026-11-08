package email

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"

	"github.com/google/uuid"
)

// A sending pathway is one configured way a brand delivers mail. Each kind
// of mail (transactional, marketing, outreach) is carried by at most one of
// the brand's pathways, and Send uses exactly that one: a brand with no
// pathway for a kind can't send that kind, and nothing falls back to another
// pathway. The limits are per sender (a mailbox of a pool, or the pathway
// itself) and are enforced in Send.

// The providers a pathway can deliver through.
const (
	ProviderPlatform  = "platform"  // the server's email settings (SES or the platform SMTP)
	ProviderSES       = "ses"       // the brand's own AWS SES account
	ProviderSMTP      = "smtp"      // the brand's own SMTP relay
	ProviderMailboxes = "mailboxes" // the brand's pool of connected mailboxes
)

// OutreachWarning is shown whenever outreach rides on a provider other than
// mailboxes. It is never a refusal.
const OutreachWarning = "Amazon SES and most email platforms forbid cold email in their terms and can suspend the account."

// Name is the kind as people and the API say it.
func (k Kind) Name() string {
	if k == KindTransactional {
		return "transactional"
	}
	return string(k)
}

// ParseKind reads a kind's name.
func ParseKind(name string) (Kind, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "transactional":
		return KindTransactional, nil
	case "marketing":
		return KindMarketing, nil
	case "outreach":
		return KindOutreach, nil
	}
	return "", errorx.NewBadRequestError(fmt.Sprintf("%q is not a kind of mail: use transactional, marketing or outreach", name))
}

// Limits are a pathway's per-sender limits. 0 turns one off.
type Limits struct {
	DailyLimit      int64   `json:"daily_limit"`       // emails per sender per day
	WarmupStart     int64   `json:"warmup_start"`      // a new sender's first-day allowance
	WarmupStep      int64   `json:"warmup_step"`       // added each day until the daily limit
	MinGapSeconds   int64   `json:"min_gap_seconds"`   // between two sends from one sender
	MaxGapSeconds   int64   `json:"max_gap_seconds"`   //
	PauseBounceRate float64 `json:"pause_bounce_rate"` // a sender pauses above this bounce rate (0.03 is 3%), over 7 days
	PauseMinBounces int64   `json:"pause_min_bounces"` // ... once it has at least this many bounces
	PauseSpamBlocks int64   `json:"pause_spam_blocks"` // ... or this many spam blocks
	PauseComplaints int64   `json:"pause_complaints"`  // ... or this many complaints
}

// DefaultLimits are a new pathway's limits: a mailbox pool is paced and
// warmed up like a person sending; the other providers have no limits.
func DefaultLimits(provider string) Limits {
	l := Limits{PauseBounceRate: 0.03, PauseMinBounces: 2, PauseSpamBlocks: 2, PauseComplaints: 2}
	if provider == ProviderMailboxes {
		l.DailyLimit, l.WarmupStart, l.WarmupStep = 30, 5, 2
		l.MinGapSeconds, l.MaxGapSeconds = 240, 540
	}
	return l
}

// LimitsOf reads a pathway's limits.
func LimitsOf(p db.SendingPathway) Limits {
	return Limits{
		DailyLimit: p.DailyLimit, WarmupStart: p.WarmupStart, WarmupStep: p.WarmupStep,
		MinGapSeconds: p.MinGapSeconds, MaxGapSeconds: p.MaxGapSeconds,
		PauseBounceRate: p.PauseBounceRate, PauseMinBounces: p.PauseMinBounces,
		PauseSpamBlocks: p.PauseSpamBlocks, PauseComplaints: p.PauseComplaints,
	}
}

func (l Limits) validate() error {
	for name, v := range map[string]int64{
		"daily_limit": l.DailyLimit, "warmup_start": l.WarmupStart, "warmup_step": l.WarmupStep,
		"min_gap_seconds": l.MinGapSeconds, "max_gap_seconds": l.MaxGapSeconds,
		"pause_min_bounces": l.PauseMinBounces, "pause_spam_blocks": l.PauseSpamBlocks, "pause_complaints": l.PauseComplaints,
	} {
		if v < 0 {
			return errorx.NewBadRequestError(name + " can't be negative")
		}
	}
	if l.MaxGapSeconds < l.MinGapSeconds {
		return errorx.NewBadRequestError("max_gap_seconds must be at least min_gap_seconds")
	}
	if l.PauseBounceRate < 0 || l.PauseBounceRate > 1 {
		return errorx.NewBadRequestError("pause_bounce_rate is a fraction between 0 and 1 (0.03 is 3%)")
	}
	return nil
}

// Allowance is how many emails one sender may send on the day of now: the
// warm-up ramp from the sender's start, up to the daily limit (the
// mailbox's own when it has one). limited is false when nothing caps it.
func Allowance(p db.SendingPathway, start time.Time, override sql.NullInt64, now time.Time) (n int64, limited bool) {
	limit := p.DailyLimit
	if override.Valid {
		limit = override.Int64
	}
	if p.WarmupStart <= 0 {
		return limit, limit > 0
	}
	days := int64(now.Sub(start).Hours() / 24)
	if days < 0 {
		days = 0
	}
	ramp := p.WarmupStart + p.WarmupStep*days
	if limit > 0 {
		return min(limit, ramp), true
	}
	return ramp, true
}

// Health is one sender's last 7 days.
type Health struct {
	Sent, Bounces, SpamBlocks, Complaints int64
}

// HealthPause is why a sender with this health should pause under the
// pathway's thresholds, or "".
func HealthPause(p db.SendingPathway, h Health) string {
	switch {
	case p.PauseSpamBlocks > 0 && h.SpamBlocks >= p.PauseSpamBlocks:
		return fmt.Sprintf("%d emails rejected as spam in 7 days; limit %d", h.SpamBlocks, p.PauseSpamBlocks)
	case p.PauseBounceRate > 0 && h.Bounces >= max(p.PauseMinBounces, 1) && h.Sent > 0 && float64(h.Bounces)/float64(h.Sent) > p.PauseBounceRate:
		return fmt.Sprintf("bounce rate %.1f%% in 7 days (%d of %d); limit %.0f%%", 100*float64(h.Bounces)/float64(h.Sent), h.Bounces, h.Sent, 100*p.PauseBounceRate)
	case p.PauseComplaints > 0 && h.Complaints >= p.PauseComplaints:
		return fmt.Sprintf("%d complaints in 7 days; limit %d", h.Complaints, p.PauseComplaints)
	}
	return ""
}

// Day is the send-count day of t in a brand's time zone.
func Day(t time.Time, org db.Organization) string {
	return t.In(LoadLocation(org.OutboundTimezone)).Format(time.DateOnly)
}

// SentSince is how many emails one sender of a pathway sent from the day of
// since through today.
func (s *Service) SentSince(ctx context.Context, org db.Organization, pathwayID, senderID string, since time.Time) (int64, error) {
	return s.db.SumSendCountsSince(ctx, db.SumSendCountsSinceParams{PathwayID: pathwayID, SenderID: senderID, SinceDay: Day(since, org)})
}

// ---- the brand's pathways ---------------------------------------------------

// PathwayInput creates or changes a pathway. Credentials are write-only: an
// empty one keeps what is stored.
type PathwayInput struct {
	Name         string   `json:"name,omitempty"`
	Provider     string   `json:"provider,omitempty"` // on create only
	Kinds        []string `json:"kinds"`              // the whole set it carries
	AWSRegion    string   `json:"aws_region,omitempty"`
	AWSAccessKey string   `json:"aws_access_key,omitempty"`
	AWSSecretKey string   `json:"aws_secret_key,omitempty"`
	SMTPHost     string   `json:"smtp_host,omitempty"`
	SMTPPort     int64    `json:"smtp_port,omitempty"`
	SMTPUsername string   `json:"smtp_username,omitempty"`
	SMTPPassword string   `json:"smtp_password,omitempty"`
	Limits       *Limits  `json:"limits,omitempty"` // nil: the provider's defaults on create, unchanged on update
}

// Pathway is a pathway as the brand sees it: never its credentials, only
// whether they are set.
type Pathway struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Provider        string   `json:"provider"`
	Kinds           []string `json:"kinds"`
	AWSRegion       string   `json:"aws_region,omitempty"`
	AWSKeysSet      bool     `json:"aws_keys_set"`
	SMTPHost        string   `json:"smtp_host,omitempty"`
	SMTPPort        int64    `json:"smtp_port,omitempty"`
	SMTPUsername    string   `json:"smtp_username,omitempty"`
	SMTPPasswordSet bool     `json:"smtp_password_set"`
	Limits          Limits   `json:"limits"`
	Domains         []string `json:"domains"`
	Mailboxes       int      `json:"mailboxes"`
	Warning         string   `json:"warning,omitempty"`
	CreatedAt       string   `json:"created_at"`

	// A mailbox pool's mailbox service (manual, maildoso, cheapinboxes,
	// infraforge), and whether its API key is set.
	MailboxService       string `json:"mailbox_service,omitempty"`
	MailboxServiceKeySet bool   `json:"mailbox_service_key_set"`
}

// KindsOf lists the kinds a pathway carries.
func KindsOf(p db.SendingPathway) []string {
	kinds := []string{}
	if p.CarriesTransactional == 1 {
		kinds = append(kinds, KindTransactional.Name())
	}
	if p.CarriesMarketing == 1 {
		kinds = append(kinds, KindMarketing.Name())
	}
	if p.CarriesOutreach == 1 {
		kinds = append(kinds, KindOutreach.Name())
	}
	return kinds
}

func (s *Service) view(ctx context.Context, p db.SendingPathway) (Pathway, error) {
	out := Pathway{
		ID: p.ID, Name: p.Name, Provider: p.Provider, Kinds: KindsOf(p),
		AWSRegion: p.AwsRegion, AWSKeysSet: len(p.AwsAccessKeyEnc) > 0 && len(p.AwsSecretKeyEnc) > 0,
		SMTPHost: p.SmtpHost, SMTPPort: p.SmtpPort, SMTPUsername: p.SmtpUsername, SMTPPasswordSet: len(p.SmtpPasswordEnc) > 0,
		Limits: LimitsOf(p), Domains: []string{}, CreatedAt: p.CreatedAt,
	}
	if p.CarriesOutreach == 1 && p.Provider != ProviderMailboxes {
		out.Warning = OutreachWarning
	}
	domains, err := s.db.ListPathwayDomains(ctx, sql.NullString{String: p.ID, Valid: true})
	if err != nil {
		return out, err
	}
	for _, d := range domains {
		out.Domains = append(out.Domains, d.Domain)
	}
	if p.Provider == ProviderMailboxes {
		mbs, err := s.db.ListPoolMailboxes(ctx, sql.NullString{String: p.ID, Valid: true})
		if err != nil {
			return out, err
		}
		out.Mailboxes = len(mbs)
		out.MailboxService, out.MailboxServiceKeySet = p.MailboxService, len(p.MailboxServiceKeyEnc) > 0
	}
	return out, nil
}

// ListPathways is the brand's pathways, oldest first.
func (s *Service) ListPathways(ctx context.Context, orgID string) ([]Pathway, error) {
	ps, err := s.db.ListSendingPathways(ctx, orgID)
	if err != nil {
		return nil, err
	}
	out := make([]Pathway, 0, len(ps))
	for _, p := range ps {
		v, err := s.view(ctx, p)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// GetPathway is one of the brand's pathways.
func (s *Service) GetPathway(ctx context.Context, orgID, id string) (db.SendingPathway, error) {
	p, err := s.db.GetSendingPathway(ctx, db.GetSendingPathwayParams{ID: id, OrgID: orgID})
	if errors.Is(err, sql.ErrNoRows) {
		return p, errorx.NewNotFoundError("no such sending pathway")
	}
	return p, err
}

func kindFlags(kinds []string) (t, m, o int64, err error) {
	for _, name := range kinds {
		k, err := ParseKind(name)
		if err != nil {
			return 0, 0, 0, err
		}
		switch k {
		case KindTransactional:
			t = 1
		case KindMarketing:
			m = 1
		case KindOutreach:
			o = 1
		}
	}
	return t, m, o, nil
}

func (s *Service) encrypt(v string) ([]byte, error) {
	if s.cryptoSvc == nil {
		return nil, errorx.NewBadRequestError("ENCRYPTION_KEY is not set on this server, so it can't store credentials")
	}
	return s.cryptoSvc.EncryptString(v)
}

// CreatePathway adds a pathway to the brand. The kinds it carries leave the
// brand's other pathways.
func (s *Service) CreatePathway(ctx context.Context, orgID string, in PathwayInput) (Pathway, error) {
	provider := strings.ToLower(strings.TrimSpace(in.Provider))
	if !slices.Contains([]string{ProviderPlatform, ProviderSES, ProviderSMTP, ProviderMailboxes}, provider) {
		return Pathway{}, errorx.NewBadRequestError("provider must be platform, ses, smtp or mailboxes")
	}
	limits := DefaultLimits(provider)
	if in.Limits != nil {
		limits = *in.Limits
	}
	if err := limits.validate(); err != nil {
		return Pathway{}, err
	}
	t, m, o, err := kindFlags(in.Kinds)
	if err != nil {
		return Pathway{}, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = map[string]string{ProviderPlatform: "Platform email", ProviderSES: "Amazon SES", ProviderSMTP: "SMTP relay", ProviderMailboxes: "Mailboxes"}[provider]
	}
	params := db.CreateSendingPathwayParams{
		ID: uuid.NewString(), OrgID: orgID, Name: name, Provider: provider,
		DailyLimit: limits.DailyLimit, WarmupStart: limits.WarmupStart, WarmupStep: limits.WarmupStep,
		MinGapSeconds: limits.MinGapSeconds, MaxGapSeconds: limits.MaxGapSeconds,
		PauseBounceRate: limits.PauseBounceRate, PauseMinBounces: limits.PauseMinBounces,
		PauseSpamBlocks: limits.PauseSpamBlocks, PauseComplaints: limits.PauseComplaints,
	}
	switch provider {
	case ProviderSES:
		if in.AWSAccessKey == "" || in.AWSSecretKey == "" {
			return Pathway{}, errorx.NewBadRequestError("an ses pathway needs aws_access_key and aws_secret_key")
		}
		params.AwsRegion = firstNonEmpty(in.AWSRegion, "us-east-1")
		if params.AwsAccessKeyEnc, err = s.encrypt(in.AWSAccessKey); err != nil {
			return Pathway{}, err
		}
		if params.AwsSecretKeyEnc, err = s.encrypt(in.AWSSecretKey); err != nil {
			return Pathway{}, err
		}
	case ProviderSMTP:
		if strings.TrimSpace(in.SMTPHost) == "" || in.SMTPUsername == "" || in.SMTPPassword == "" {
			return Pathway{}, errorx.NewBadRequestError("an smtp pathway needs smtp_host, smtp_username and smtp_password")
		}
		params.SmtpHost, params.SmtpUsername = strings.TrimSpace(in.SMTPHost), in.SMTPUsername
		params.SmtpPort = in.SMTPPort
		if params.SmtpPort == 0 {
			params.SmtpPort = 587
		}
		if params.SmtpPasswordEnc, err = s.encrypt(in.SMTPPassword); err != nil {
			return Pathway{}, err
		}
	}
	err = s.db.ExecTx(ctx, func(q *db.Queries) error {
		if _, err := q.CreateSendingPathway(ctx, params); err != nil {
			return err
		}
		return setKinds(ctx, q, orgID, params.ID, t, m, o)
	})
	if err != nil {
		return Pathway{}, err
	}
	p, err := s.GetPathway(ctx, orgID, params.ID)
	if err != nil {
		return Pathway{}, err
	}
	return s.view(ctx, p)
}

func setKinds(ctx context.Context, q *db.Queries, orgID, id string, t, m, o int64) error {
	if err := q.ReleaseKindsFromOtherPathways(ctx, db.ReleaseKindsFromOtherPathwaysParams{Transactional: t, Marketing: m, Outreach: o, OrgID: orgID, ID: id}); err != nil {
		return err
	}
	return q.SetPathwayKinds(ctx, db.SetPathwayKindsParams{Transactional: t, Marketing: m, Outreach: o, ID: id, OrgID: orgID})
}

// UpdatePathway changes a pathway: its name, the kinds it carries (the
// whole set), credentials (empty keeps them) and limits (nil keeps them).
// Its provider never changes.
func (s *Service) UpdatePathway(ctx context.Context, orgID, id string, in PathwayInput) (Pathway, error) {
	p, err := s.GetPathway(ctx, orgID, id)
	if err != nil {
		return Pathway{}, err
	}
	if in.Provider != "" && !strings.EqualFold(in.Provider, p.Provider) {
		return Pathway{}, errorx.NewBadRequestError("a pathway's provider can't change; add a new pathway instead")
	}
	limits := LimitsOf(p)
	if in.Limits != nil {
		limits = *in.Limits
	}
	if err := limits.validate(); err != nil {
		return Pathway{}, err
	}
	t, m, o, err := kindFlags(in.Kinds)
	if err != nil {
		return Pathway{}, err
	}
	params := db.UpdateSendingPathwayParams{
		Name: firstNonEmpty(in.Name, p.Name), AwsRegion: p.AwsRegion,
		SmtpHost: p.SmtpHost, SmtpPort: p.SmtpPort, SmtpUsername: p.SmtpUsername,
		DailyLimit: limits.DailyLimit, WarmupStart: limits.WarmupStart, WarmupStep: limits.WarmupStep,
		MinGapSeconds: limits.MinGapSeconds, MaxGapSeconds: limits.MaxGapSeconds,
		PauseBounceRate: limits.PauseBounceRate, PauseMinBounces: limits.PauseMinBounces,
		PauseSpamBlocks: limits.PauseSpamBlocks, PauseComplaints: limits.PauseComplaints,
		ID: p.ID, OrgID: orgID,
	}
	var awsAccess, awsSecret, smtpPassword []byte
	switch p.Provider {
	case ProviderSES:
		params.AwsRegion = firstNonEmpty(in.AWSRegion, p.AwsRegion)
		if (in.AWSAccessKey == "") != (in.AWSSecretKey == "") {
			return Pathway{}, errorx.NewBadRequestError("change aws_access_key and aws_secret_key together")
		}
		if in.AWSAccessKey != "" {
			if awsAccess, err = s.encrypt(in.AWSAccessKey); err != nil {
				return Pathway{}, err
			}
			if awsSecret, err = s.encrypt(in.AWSSecretKey); err != nil {
				return Pathway{}, err
			}
		}
	case ProviderSMTP:
		params.SmtpHost = firstNonEmpty(in.SMTPHost, p.SmtpHost)
		params.SmtpUsername = firstNonEmpty(in.SMTPUsername, p.SmtpUsername)
		if in.SMTPPort != 0 {
			params.SmtpPort = in.SMTPPort
		}
		if in.SMTPPassword != "" {
			if smtpPassword, err = s.encrypt(in.SMTPPassword); err != nil {
				return Pathway{}, err
			}
		}
	}
	err = s.db.ExecTx(ctx, func(q *db.Queries) error {
		if err := q.UpdateSendingPathway(ctx, params); err != nil {
			return err
		}
		if awsAccess != nil {
			if err := q.SetSendingPathwayAWSKeys(ctx, db.SetSendingPathwayAWSKeysParams{AwsAccessKeyEnc: awsAccess, AwsSecretKeyEnc: awsSecret, ID: p.ID, OrgID: orgID}); err != nil {
				return err
			}
		}
		if smtpPassword != nil {
			if err := q.SetSendingPathwaySMTPPassword(ctx, db.SetSendingPathwaySMTPPasswordParams{SmtpPasswordEnc: smtpPassword, ID: p.ID, OrgID: orgID}); err != nil {
				return err
			}
		}
		return setKinds(ctx, q, orgID, p.ID, t, m, o)
	})
	if err != nil {
		return Pathway{}, err
	}
	if p, err = s.GetPathway(ctx, orgID, id); err != nil {
		return Pathway{}, err
	}
	return s.view(ctx, p)
}

// DeletePathway removes a pathway that carries no mail and holds no domains.
func (s *Service) DeletePathway(ctx context.Context, orgID, id string) error {
	p, err := s.GetPathway(ctx, orgID, id)
	if err != nil {
		return err
	}
	if kinds := KindsOf(p); len(kinds) > 0 {
		return errorx.NewBadRequestError(fmt.Sprintf("%s still carries %s mail; tick those kinds on another pathway first", p.Name, strings.Join(kinds, " and ")))
	}
	domains, err := s.db.ListPathwayDomains(ctx, sql.NullString{String: p.ID, Valid: true})
	if err != nil {
		return err
	}
	if len(domains) > 0 {
		return errorx.NewBadRequestError(fmt.Sprintf("%s still holds %d domains; remove them first", p.Name, len(domains)))
	}
	_, err = s.db.DeleteSendingPathway(ctx, db.DeleteSendingPathwayParams{ID: p.ID, OrgID: orgID})
	return err
}

// SetMailboxDailyLimit gives one mailbox its own daily limit, or (nil) puts
// it back on its pool's.
func (s *Service) SetMailboxDailyLimit(ctx context.Context, orgID, mailboxID string, limit *int64) error {
	v := sql.NullInt64{}
	if limit != nil {
		if *limit < 1 {
			return errorx.NewBadRequestError("a mailbox's daily limit is at least 1; clear it to use the pool's")
		}
		v = sql.NullInt64{Int64: *limit, Valid: true}
	}
	n, err := s.db.SetMailboxDailyLimit(ctx, db.SetMailboxDailyLimitParams{DailyLimit: v, ID: mailboxID, OrgID: orgID})
	if err != nil {
		return err
	}
	if n == 0 {
		return errorx.NewNotFoundError("no such mailbox")
	}
	return nil
}

// DomainPathway is the pathway a new domain of the brand joins. A sending
// domain joins the named pathway (never a mailbox pool), else the one
// carrying transactional mail, else marketing, else the brand's first
// non-mailbox pathway. An outreach domain joins the named mailbox pool, else
// the pool carrying outreach, else the brand's first pool, else a new pool
// (carrying outreach when nothing else does).
func (s *Service) DomainPathway(ctx context.Context, orgID, purpose, requestedID string) (db.SendingPathway, error) {
	outreach := purpose == "outreach"
	if requestedID != "" {
		p, err := s.GetPathway(ctx, orgID, requestedID)
		if err != nil {
			return p, err
		}
		if outreach != (p.Provider == ProviderMailboxes) {
			if outreach {
				return p, errorx.NewBadRequestError("outreach domains belong to a mailbox pool")
			}
			return p, errorx.NewBadRequestError("sending domains belong to a platform, ses or smtp pathway, not a mailbox pool")
		}
		return p, nil
	}
	all, err := s.db.ListSendingPathways(ctx, orgID)
	if err != nil {
		return db.SendingPathway{}, err
	}
	pick := func(carries func(db.SendingPathway) bool) (db.SendingPathway, bool) {
		for _, p := range all {
			if (p.Provider == ProviderMailboxes) == outreach && carries(p) {
				return p, true
			}
		}
		return db.SendingPathway{}, false
	}
	if outreach {
		if p, ok := pick(func(p db.SendingPathway) bool { return p.CarriesOutreach == 1 }); ok {
			return p, nil
		}
		if p, ok := pick(func(db.SendingPathway) bool { return true }); ok {
			return p, nil
		}
		in := PathwayInput{Provider: ProviderMailboxes}
		if !slices.ContainsFunc(all, func(p db.SendingPathway) bool { return p.CarriesOutreach == 1 }) {
			in.Kinds = []string{KindOutreach.Name()}
		}
		v, err := s.CreatePathway(ctx, orgID, in)
		if err != nil {
			return db.SendingPathway{}, err
		}
		return s.GetPathway(ctx, orgID, v.ID)
	}
	for _, carries := range []func(db.SendingPathway) bool{
		func(p db.SendingPathway) bool { return p.CarriesTransactional == 1 },
		func(p db.SendingPathway) bool { return p.CarriesMarketing == 1 },
		func(db.SendingPathway) bool { return true },
	} {
		if p, ok := pick(carries); ok {
			return p, nil
		}
	}
	return db.SendingPathway{}, errorx.NewBadRequestError("This brand has no sending pathway for its domains. Add one in Settings, Sending.")
}
