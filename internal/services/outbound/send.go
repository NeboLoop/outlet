package outbound

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/db"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

// outMessage is one email as it goes on the wire.
type outMessage struct {
	FromName, From, To string
	Subject, Body      string
	MessageID          string // without angle brackets
	InReplyTo          string // without angle brackets
	References         []string
	Date               time.Time
}

func headerSafe(s string) string {
	return strings.NewReplacer("\r", " ", "\n", " ").Replace(s)
}

func angle(id string) string { return "<" + strings.Trim(id, "<> ") + ">" }

// build writes a plain-text email: one text part, no tracking, threading
// headers when it is a follow-up, and a List-Unsubscribe that replies to the
// sender (read by the reply reader).
func (m outMessage) build() []byte {
	var b bytes.Buffer
	h := func(k, v string) { fmt.Fprintf(&b, "%s: %s\r\n", k, headerSafe(v)) }
	h("From", (&mail.Address{Name: m.FromName, Address: m.From}).String())
	h("To", m.To)
	h("Subject", mime.QEncoding.Encode("utf-8", m.Subject))
	h("Date", m.Date.Format(time.RFC1123Z))
	h("Message-ID", angle(m.MessageID))
	if m.InReplyTo != "" {
		h("In-Reply-To", angle(m.InReplyTo))
		refs := make([]string, 0, len(m.References))
		for _, r := range m.References {
			refs = append(refs, angle(r))
		}
		h("References", strings.Join(refs, " "))
	}
	h("List-Unsubscribe", "<mailto:"+m.From+"?subject=unsubscribe>")
	h("MIME-Version", "1.0")
	h("Content-Type", `text/plain; charset="utf-8"`)
	h("Content-Transfer-Encoding", "quoted-printable")
	b.WriteString("\r\n")
	qp := quotedprintable.NewWriter(&b)
	_, _ = qp.Write([]byte(strings.ReplaceAll(strings.ReplaceAll(m.Body, "\r\n", "\n"), "\n", "\r\n")))
	_ = qp.Close()
	return b.Bytes()
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// dialSMTP connects and logs in: implicit TLS on 465, STARTTLS otherwise.
// Plain text is allowed only to a loopback host (a local test server).
func dialSMTP(mb db.OutboundMailbox, password string) (*smtp.Client, error) {
	addr := net.JoinHostPort(mb.SmtpHost, fmt.Sprint(mb.SmtpPort))
	tlsCfg := &tls.Config{ServerName: mb.SmtpHost}
	var c *smtp.Client
	var err error
	switch {
	case mb.SmtpPort == 465:
		c, err = smtp.DialTLS(addr, tlsCfg)
	case isLoopback(mb.SmtpHost):
		c, err = smtp.Dial(addr)
	default:
		c, err = smtp.DialStartTLS(addr, tlsCfg)
	}
	if err != nil {
		return nil, err
	}
	if err := c.Hello(helloName(mb.Email)); err != nil {
		c.Close()
		return nil, err
	}
	if err := c.Auth(sasl.NewPlainClient("", mb.SmtpUsername, password)); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}

func helloName(email string) string {
	if d := domainOf(email); d != "" {
		return d
	}
	return "localhost"
}

func testSMTP(_ context.Context, mb db.OutboundMailbox, password string) error {
	c, err := dialSMTP(mb, password)
	if err != nil {
		return err
	}
	return c.Quit()
}

// sendSMTP sends one message. A 4xx reply is transient; anything else that
// fails after the server answered is permanent for this message.
func sendSMTP(mb db.OutboundMailbox, password string, m outMessage) (transient bool, err error) {
	c, err := dialSMTP(mb, password)
	if err != nil {
		return true, err
	}
	defer c.Close()
	err = c.SendMail(mb.Email, []string{m.To}, bytes.NewReader(m.build()))
	if err != nil {
		if se, ok := err.(*smtp.SMTPError); ok {
			return se.Code >= 400 && se.Code < 500, err
		}
		return true, err
	}
	_ = c.Quit()
	return false, nil
}
