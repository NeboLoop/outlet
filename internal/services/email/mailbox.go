package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"

	"github.com/outlet-sh/outlet/internal/db"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

// Outreach (cold email) leaves only through the brand's own connected
// mailboxes, logged in over SMTP. It never reaches SES or any other email
// platform: their terms forbid it, and a suspension would stop every brand's
// transactional mail.

// DeliveryError is a mailbox delivery failure. Temporary ones (a 4xx reply, a
// dropped connection) are tried again later from the same mailbox.
type DeliveryError struct {
	Temporary bool
	Err       error
}

func (e *DeliveryError) Error() string { return e.Err.Error() }
func (e *DeliveryError) Unwrap() error { return e.Err }

// IsTemporary reports whether a send failed in a way worth trying again.
func IsTemporary(err error) bool {
	var d *DeliveryError
	return errors.As(err, &d) && d.Temporary
}

// MailboxLogin is what a mailbox needs to send.
type MailboxLogin struct {
	Email, Host, Username, Password string
	Port                            int
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// dialMailbox connects and logs in: implicit TLS on 465, STARTTLS otherwise.
// Plain text is allowed only to a loopback host (a local test server).
func dialMailbox(l MailboxLogin) (*smtp.Client, error) {
	addr := net.JoinHostPort(l.Host, fmt.Sprint(l.Port))
	tlsCfg := &tls.Config{ServerName: l.Host}
	var c *smtp.Client
	var err error
	switch {
	case l.Port == 465:
		c, err = smtp.DialTLS(addr, tlsCfg)
	case isLoopback(l.Host):
		c, err = smtp.Dial(addr)
	default:
		c, err = smtp.DialStartTLS(addr, tlsCfg)
	}
	if err != nil {
		return nil, err
	}
	hello := "localhost"
	if _, d, ok := cutAt(l.Email); ok {
		hello = d
	}
	if err := c.Hello(hello); err != nil {
		c.Close()
		return nil, err
	}
	if err := c.Auth(sasl.NewPlainClient("", l.Username, l.Password)); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}

func cutAt(addr string) (string, string, bool) {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == '@' {
			return addr[:i], addr[i+1:], true
		}
	}
	return "", "", false
}

// CheckMailboxLogin logs in to a mailbox's SMTP server and out again.
func CheckMailboxLogin(l MailboxLogin) error {
	c, err := dialMailbox(l)
	if err != nil {
		return err
	}
	return c.Quit()
}

// sendThroughMailbox delivers an outreach email from its mailbox. The From is
// the mailbox itself; nothing else may send as it.
func (s *Service) sendThroughMailbox(ctx context.Context, m Message) error {
	if m.MailboxID == "" || m.OrgID == "" {
		return invalid("outreach needs its brand and mailbox")
	}
	mb, err := s.db.GetMailbox(ctx, db.GetMailboxParams{ID: m.MailboxID, OrgID: m.OrgID})
	if err != nil {
		return invalid("mailbox %s is not one of this brand's", m.MailboxID)
	}
	if s.cryptoSvc == nil {
		return fmt.Errorf("ENCRYPTION_KEY is not set, so mailbox passwords can't be read")
	}
	password, err := s.cryptoSvc.DecryptString(mb.SmtpPasswordEnc)
	if err != nil {
		return err
	}
	m.FromEmail, m.FromName = mb.Email, mb.FromName
	from, err := parseSingleAddress("from", m.FromEmail)
	if err != nil {
		return err
	}
	from.Name = m.FromName
	to, err := parseSingleAddress("to", m.To)
	if err != nil {
		return err
	}
	env := envelope{from: from, to: to, msg: m}
	if blocked, err := s.db.IsSendBlocked(ctx, db.IsSendBlockedParams{Email: to.Address, OrgID: m.OrgID, Domain: env.recipientDomain()}); err != nil {
		return fmt.Errorf("check suppression: %w", err)
	} else if blocked == 1 {
		return ErrSuppressed
	}
	raw, err := env.build()
	if err != nil {
		return err
	}
	c, err := dialMailbox(MailboxLogin{Email: mb.Email, Host: mb.SmtpHost, Port: int(mb.SmtpPort), Username: mb.SmtpUsername, Password: password})
	if err != nil {
		return &DeliveryError{Temporary: true, Err: err}
	}
	defer c.Close()
	if err := c.SendMail(mb.Email, []string{to.Address}, bytes.NewReader(raw)); err != nil {
		var se *smtp.SMTPError
		if errors.As(err, &se) {
			return &DeliveryError{Temporary: se.Code >= 400 && se.Code < 500, Err: err}
		}
		return &DeliveryError{Temporary: true, Err: err}
	}
	_ = c.Quit()
	return nil
}
