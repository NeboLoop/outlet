package email

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"time"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

// A brand's own SMTP logins: its connected mailboxes, and the relay of an
// smtp pathway. Both deliver through deliverSMTP.

// DeliveryError is a delivery failure Send can tell apart. Temporary ones (a
// 4xx reply, a dropped connection, a sender over its pathway's limit) are
// tried again later; RetryAt, when set, is the earliest worth trying.
type DeliveryError struct {
	Temporary bool
	RetryAt   time.Time
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

// deliverSMTP sends one built message through a brand's own SMTP login. A
// 4xx reply or a failed connection is temporary.
func deliverSMTP(l MailboxLogin, from, to string, raw []byte) error {
	c, err := dialMailbox(l)
	if err != nil {
		return &DeliveryError{Temporary: true, Err: err}
	}
	defer c.Close()
	if err := c.SendMail(from, []string{to}, bytes.NewReader(raw)); err != nil {
		return smtpFailure(err)
	}
	_ = c.Quit()
	return nil
}

// smtpFailure classifies an SMTP send's error: a 4xx reply is temporary, a
// 5xx reply permanent, and anything without a reply (a refused or dropped
// connection, a timeout) temporary. It reads the replies of both SMTP
// clients Outlet uses: go-smtp's and net/smtp's.
func smtpFailure(err error) error {
	if err == nil {
		return nil
	}
	var se *smtp.SMTPError
	if errors.As(err, &se) {
		return &DeliveryError{Temporary: se.Code >= 400 && se.Code < 500, Err: err}
	}
	var te *textproto.Error
	if errors.As(err, &te) {
		return &DeliveryError{Temporary: te.Code >= 400 && te.Code < 500, Err: err}
	}
	return &DeliveryError{Temporary: true, Err: err}
}
