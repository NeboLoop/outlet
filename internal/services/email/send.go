package email

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
)

// ErrSuppressed means the recipient hard-bounced, complained, or is on the
// brand's suppression list or blocked domains, so the email was not sent.
var ErrSuppressed = errors.New("recipient is suppressed")

// Send delivers one email. It is the only way Outlet sends mail. A brand's
// mail goes through the one sending pathway carrying its kind (none means it
// is refused, never sent another way), within that pathway's limits; mail
// with no brand (password resets and the like) goes through the platform's
// email settings. Send refuses suppressed recipients, resolves From and
// Reply-To (message, then brand, then platform), and builds a safe MIME
// message. A sender over its limit gets a temporary DeliveryError wrapping
// ErrOverLimit, with RetryAt.
func (s *Service) Send(ctx context.Context, m Message) error {
	if m.OrgID == "" {
		if m.Kind == KindOutreach || m.MailboxID != "" {
			return invalid("outreach needs its brand and mailbox")
		}
		env, err := s.resolve(ctx, m, nil)
		if err != nil {
			return err
		}
		if err := s.checkSuppressed(ctx, m.OrgID, env); err != nil {
			return err
		}
		raw, err := env.build()
		if err != nil {
			return err
		}
		return s.deliverPlatform(ctx, env, raw)
	}

	org, err := s.db.GetOrganizationByID(ctx, m.OrgID)
	if err != nil {
		return invalid("brand %s not found", m.OrgID)
	}
	p, err := s.db.GetPathwayForKind(ctx, db.GetPathwayForKindParams{OrgID: org.ID, Kind: m.Kind.Name()})
	if errors.Is(err, sql.ErrNoRows) {
		return invalid("brand %q has no sending pathway for %s mail; add one in Settings, Sending", org.Name, m.Kind.Name())
	}
	if err != nil {
		return fmt.Errorf("find the sending pathway: %w", err)
	}
	if p.Provider == ProviderMailboxes {
		return s.sendFromPool(ctx, org, p, m)
	}
	return s.sendThroughProvider(ctx, org, p, m)
}

func (s *Service) checkSuppressed(ctx context.Context, orgID string, env envelope) error {
	blocked, err := s.db.IsSendBlocked(ctx, db.IsSendBlockedParams{Email: env.to.Address, OrgID: orgID, Domain: env.recipientDomain()})
	if err != nil {
		return fmt.Errorf("check suppression: %w", err)
	}
	if blocked == 1 {
		return ErrSuppressed
	}
	return nil
}

// mailboxEnvelope addresses a message from a mailbox: its own address and
// name, which nothing else may send as.
func mailboxEnvelope(m Message, mb db.Mailbox) (envelope, error) {
	from, err := parseSingleAddress("from", mb.Email)
	if err != nil {
		return envelope{}, err
	}
	from.Name = mb.FromName
	to, err := parseSingleAddress("to", m.To)
	if err != nil {
		return envelope{}, err
	}
	m.FromEmail, m.FromName = mb.Email, mb.FromName
	env := envelope{from: from, to: to, msg: m}
	if m.ReplyTo != "" {
		if env.replyTo, err = parseSingleAddress("reply-to", m.ReplyTo); err != nil {
			return envelope{}, err
		}
	}
	return env, nil
}

// brandMailbox is one of the brand's mailboxes, named by an outreach message.
func (s *Service) brandMailbox(ctx context.Context, m Message) (db.Mailbox, error) {
	if m.MailboxID == "" {
		return db.Mailbox{}, invalid("outreach needs its brand and mailbox")
	}
	mb, err := s.db.GetMailbox(ctx, db.GetMailboxParams{ID: m.MailboxID, OrgID: m.OrgID})
	if err != nil {
		return db.Mailbox{}, invalid("mailbox %s is not one of this brand's", m.MailboxID)
	}
	return mb, nil
}

// sendThroughProvider sends through a platform, ses or smtp pathway, which
// is itself the one sender its limits count. Outreach on such a pathway still
// comes from the person's mailbox address, so replies reach that mailbox.
func (s *Service) sendThroughProvider(ctx context.Context, org db.Organization, p db.SendingPathway, m Message) error {
	var env envelope
	var err error
	if m.Kind == KindOutreach {
		mb, err := s.brandMailbox(ctx, m)
		if err != nil {
			return err
		}
		if env, err = mailboxEnvelope(m, mb); err != nil {
			return err
		}
	} else {
		if m.MailboxID != "" {
			return invalid("only outreach names a mailbox")
		}
		if env, err = s.resolve(ctx, m, &p); err != nil {
			return err
		}
	}
	if err := s.checkSuppressed(ctx, org.ID, env); err != nil {
		return err
	}
	raw, err := env.build()
	if err != nil {
		return err
	}
	release, err := s.reserve(ctx, org, p, sender{id: p.ID, label: p.Name, start: parseStored(p.CreatedAt), next: p.NextSendAt.String})
	if err != nil {
		return err
	}
	if err := s.deliver(ctx, p, env, raw); err != nil {
		release()
		return err
	}
	return nil
}

// sendFromPool sends from a mailbox of a pool. Outreach comes from the
// person's own mailbox, so a thread stays in one mailbox; other mail from the
// least-used mailbox that is under its limit and past its spacing.
func (s *Service) sendFromPool(ctx context.Context, org db.Organization, p db.SendingPathway, m Message) error {
	pool, err := s.db.ListPoolMailboxes(ctx, sql.NullString{String: p.ID, Valid: true})
	if err != nil {
		return err
	}
	var candidates []db.Mailbox
	if m.Kind == KindOutreach {
		mb, err := s.brandMailbox(ctx, m)
		if err != nil {
			return err
		}
		if !slices.ContainsFunc(pool, func(x db.Mailbox) bool { return x.ID == mb.ID }) {
			return invalid("mailbox %s is not in the %q pool that carries outreach", mb.Email, p.Name)
		}
		candidates = []db.Mailbox{mb}
	} else {
		if m.MailboxID != "" {
			return invalid("only outreach names a mailbox; the pool picks one for other mail")
		}
		counts := map[string]int64{}
		rows, err := s.db.ListSendCountsForDay(ctx, db.ListSendCountsForDayParams{PathwayID: p.ID, Day: Day(s.now(), org)})
		if err != nil {
			return err
		}
		for _, r := range rows {
			counts[r.SenderID] = r.N
		}
		for _, mb := range pool {
			if mb.Status == "active" {
				candidates = append(candidates, mb)
			}
		}
		slices.SortStableFunc(candidates, func(a, b db.Mailbox) int { return int(counts[a.ID] - counts[b.ID]) })
		if len(candidates) == 0 {
			return &DeliveryError{Temporary: true, RetryAt: s.now().Add(time.Hour), Err: fmt.Errorf("no active mailbox in the %q pool", p.Name)}
		}
	}

	// The recipient is the same whichever mailbox sends.
	to, err := parseSingleAddress("to", m.To)
	if err != nil {
		return err
	}
	if err := s.checkSuppressed(ctx, org.ID, envelope{to: to}); err != nil {
		return err
	}
	var limitErr error
	for _, mb := range candidates {
		override, err := s.MailboxLimit(ctx, mb)
		if err != nil {
			return err
		}
		release, err := s.reserve(ctx, org, p, sender{id: mb.ID, label: mb.Email, start: parseStored(mb.WarmupStartedAt), override: override, next: mb.NextSendAt.String, mailbox: true})
		if errors.Is(err, ErrOverLimit) {
			if limitErr == nil || RetryAt(err, time.Time{}).Before(RetryAt(limitErr, time.Time{})) {
				limitErr = err
			}
			continue
		}
		if err != nil {
			return err
		}
		if err := s.sendAsMailbox(m, mb); err != nil {
			release()
			return err
		}
		return nil
	}
	return limitErr
}

// sendAsMailbox builds and delivers a message from one mailbox's own login.
func (s *Service) sendAsMailbox(m Message, mb db.Mailbox) error {
	env, err := mailboxEnvelope(m, mb)
	if err != nil {
		return err
	}
	raw, err := env.build()
	if err != nil {
		return err
	}
	if s.cryptoSvc == nil {
		return fmt.Errorf("ENCRYPTION_KEY is not set, so mailbox passwords can't be read")
	}
	password, err := s.cryptoSvc.DecryptString(mb.SmtpPasswordEnc)
	if err != nil {
		return err
	}
	return deliverSMTP(MailboxLogin{Email: mb.Email, Host: mb.SmtpHost, Port: int(mb.SmtpPort), Username: mb.SmtpUsername, Password: password}, mb.Email, env.to.Address, raw)
}

// deliver hands a built message to a platform, ses or smtp pathway.
func (s *Service) deliver(ctx context.Context, p db.SendingPathway, env envelope, raw []byte) error {
	switch p.Provider {
	case ProviderPlatform:
		return s.deliverPlatform(ctx, env, raw)
	case ProviderSES:
		region, accessKey, secretKey, err := pathwayAWSKeys(s.cryptoSvc, p)
		if err != nil {
			return err
		}
		return SendRawEmailViaSES(ctx, &SESConfig{Region: region, AccessKey: accessKey, SecretKey: secretKey}, env.from.Address, env.to.Address, raw)
	case ProviderSMTP:
		if s.cryptoSvc == nil {
			return fmt.Errorf("ENCRYPTION_KEY is not set, so the SMTP password can't be read")
		}
		password, err := s.cryptoSvc.DecryptString(p.SmtpPasswordEnc)
		if err != nil {
			return err
		}
		return deliverSMTP(MailboxLogin{Email: env.from.Address, Host: p.SmtpHost, Port: int(p.SmtpPort), Username: p.SmtpUsername, Password: password}, env.from.Address, env.to.Address, raw)
	}
	return fmt.Errorf("pathway %q has unknown provider %q", p.Name, p.Provider)
}

// deliverPlatform sends through the server's email settings: SES when it is
// configured, otherwise the platform SMTP server.
func (s *Service) deliverPlatform(ctx context.Context, env envelope, raw []byte) error {
	sesConfig, err := s.getSESConfig(ctx)
	if err == nil && s.hasSESConfig(sesConfig) {
		return SendRawEmailViaSES(ctx, sesConfig, env.from.Address, env.to.Address, raw)
	}
	smtpConfig, err := s.getGlobalSMTPConfig(ctx)
	if err != nil {
		return fmt.Errorf("load email settings: %w", err)
	}
	if smtpConfig.Host == "" || smtpConfig.User == "" || smtpConfig.Password == "" {
		return fmt.Errorf("email not configured - set AWS SES credentials or SMTP settings in platform settings")
	}
	if s.poolEnabled && s.pool != nil {
		return s.pool.SendWithPool(env.from.Address, []string{env.to.Address}, raw)
	}
	auth := smtp.PlainAuth("", smtpConfig.User, smtpConfig.Password, smtpConfig.Host)
	addr := net.JoinHostPort(smtpConfig.Host, strconv.Itoa(smtpConfig.Port))
	if err := smtp.SendMail(addr, auth, env.from.Address, []string{env.to.Address}, raw); err != nil {
		return fmt.Errorf("send via SMTP: %w", err)
	}
	return nil
}

// resolve validates the addresses and fills From and Reply-To from the brand
// and platform settings. A brand's From must be on a domain of the pathway
// carrying the mail (p); the platform's default sender is the one exception.
func (s *Service) resolve(ctx context.Context, m Message, p *db.SendingPathway) (envelope, error) {
	to, err := parseSingleAddress("to", m.To)
	if err != nil {
		return envelope{}, err
	}

	var orgFromName, orgFromEmail, orgReplyTo string
	if m.OrgID != "" {
		if org, err := s.db.GetOrgEmailSettings(ctx, m.OrgID); err == nil {
			orgFromName, orgFromEmail, orgReplyTo = org.FromName.String, org.FromEmail.String, org.ReplyTo.String
		}
	}
	platform, err := s.getGlobalSMTPConfig(ctx)
	if err != nil {
		return envelope{}, fmt.Errorf("load email settings: %w", err)
	}

	fromEmail := firstNonEmpty(m.FromEmail, orgFromEmail, platform.FromAddress)
	if fromEmail == "" {
		return envelope{}, invalid("no From address: set one on the brand or in platform settings")
	}
	from, err := parseSingleAddress("from", fromEmail)
	if err != nil {
		return envelope{}, err
	}
	if name := firstNonEmpty(m.FromName, orgFromName, platform.FromName); name != "" {
		if strings.ContainsAny(name, "\r\n") {
			return envelope{}, invalid("from name contains a line break")
		}
		from.Name = name
	}

	env := envelope{from: from, to: to, msg: m}
	if p != nil && firstNonEmpty(m.FromEmail, orgFromEmail) != "" {
		if err := checkPathwayDomain(ctx, s.db, *p, m.Kind, from.Address); err != nil {
			return envelope{}, fmt.Errorf("%w: %s", ErrInvalidMessage, err.Error())
		}
	}

	if replyTo := firstNonEmpty(m.ReplyTo, orgReplyTo, platform.ReplyTo); replyTo != "" {
		if env.replyTo, err = parseSingleAddress("reply-to", replyTo); err != nil {
			return envelope{}, err
		}
	}
	return env, nil
}

// parseStored reads a stored UTC time ('YYYY-MM-DD HH:MM:SS').
func parseStored(s string) time.Time {
	t, _ := time.ParseInLocation(TimeLayout, s, time.UTC)
	return t
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}
