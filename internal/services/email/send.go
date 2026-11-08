package email

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"

	"github.com/outlet-sh/outlet/internal/db"
)

// ErrSuppressed means the recipient hard-bounced, complained, or is on the
// brand's suppression list or blocked domains, so the email was not sent.
var ErrSuppressed = errors.New("recipient is suppressed")

// Send delivers one email. It is the only way Outlet sends mail: it refuses
// suppressed recipients, resolves From and Reply-To (message, then brand, then
// platform), builds a safe MIME message, and delivers through SES when it is
// configured, otherwise SMTP.
func (s *Service) Send(ctx context.Context, m Message) error {
	env, err := s.resolve(ctx, m)
	if err != nil {
		return err
	}

	blocked, err := s.db.IsSendBlocked(ctx, db.IsSendBlockedParams{
		Email:  env.to.Address,
		OrgID:  m.OrgID,
		Domain: env.recipientDomain(),
	})
	if err != nil {
		return fmt.Errorf("check suppression: %w", err)
	}
	if blocked == 1 {
		return ErrSuppressed
	}

	raw, err := env.build()
	if err != nil {
		return err
	}

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
// and platform settings.
func (s *Service) resolve(ctx context.Context, m Message) (envelope, error) {
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
	if m.OrgID != "" {
		claimed, err := s.db.IsDomainClaimedByOtherOrg(ctx, db.IsDomainClaimedByOtherOrgParams{
			Domain: env.domain(),
			OrgID:  m.OrgID,
		})
		if err != nil {
			return envelope{}, fmt.Errorf("check sender domain: %w", err)
		}
		if claimed == 1 {
			return envelope{}, invalid("from domain %s belongs to another brand", env.domain())
		}
	}

	if replyTo := firstNonEmpty(m.ReplyTo, orgReplyTo, platform.ReplyTo); replyTo != "" {
		if env.replyTo, err = parseSingleAddress("reply-to", replyTo); err != nil {
			return envelope{}, err
		}
	}
	return env, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}
