package email

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"strings"
	"time"
)

// Message is one outbound email. Every email Outlet sends, transactional or
// marketing, goes through Service.Send with a Message.
type Message struct {
	// OrgID is the brand sending the email. Empty for platform mail such as
	// password resets.
	OrgID string

	// From and Reply-To. Empty fields fall back to the brand's settings, then
	// the platform's.
	FromName  string
	FromEmail string
	ReplyTo   string

	To      string
	Subject string
	HTML    string
	Text    string // plain-text alternative; always send one when available

	Attachments []Attachment

	// ListUnsubscribe adds RFC 2369 List-Unsubscribe and RFC 8058
	// List-Unsubscribe-Post headers. The URL must accept a POST with the body
	// "List-Unsubscribe=One-Click". Set it on marketing mail only.
	ListUnsubscribe string
}

// ErrInvalidMessage wraps every validation failure, so callers can tell a bad
// request from a delivery failure.
var ErrInvalidMessage = errors.New("invalid message")

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidMessage, fmt.Sprintf(format, args...))
}

// parseSingleAddress accepts exactly one address, with or without a display name.
func parseSingleAddress(field, value string) (*mail.Address, error) {
	if strings.ContainsAny(value, "\r\n") {
		return nil, invalid("%s contains a line break", field)
	}
	addr, err := mail.ParseAddress(value)
	if err != nil {
		return nil, invalid("%s %q is not a valid email address", field, value)
	}
	return addr, nil
}

// envelope is a Message with its addresses resolved and validated.
type envelope struct {
	from    *mail.Address
	replyTo *mail.Address // nil when none
	to      *mail.Address
	msg     Message
}

func (e envelope) domain() string {
	at := strings.LastIndex(e.from.Address, "@")
	return e.from.Address[at+1:]
}

func (e envelope) recipientDomain() string {
	at := strings.LastIndex(e.to.Address, "@")
	return strings.ToLower(e.to.Address[at+1:])
}

// build renders the message as RFC 5322 bytes. The body is HTML alone, or
// multipart/alternative (text + HTML) when there is a text part, wrapped in
// multipart/mixed when there are attachments.
func (e envelope) build() ([]byte, error) {
	m := e.msg
	if strings.ContainsAny(m.Subject, "\r\n") {
		return nil, invalid("subject contains a line break")
	}
	if strings.ContainsAny(m.ListUnsubscribe, "\r\n<>") {
		return nil, invalid("unsubscribe URL contains invalid characters")
	}

	var h bytes.Buffer
	header := func(name, value string) { fmt.Fprintf(&h, "%s: %s\r\n", name, value) }
	header("From", e.from.String())
	header("To", e.to.String())
	if e.replyTo != nil {
		header("Reply-To", e.replyTo.String())
	}
	header("Subject", mime.QEncoding.Encode("utf-8", m.Subject))
	header("Date", time.Now().Format(time.RFC1123Z))
	header("Message-ID", fmt.Sprintf("<%s@%s>", randomID(), e.domain()))
	if m.ListUnsubscribe != "" {
		header("List-Unsubscribe", "<"+m.ListUnsubscribe+">")
		header("List-Unsubscribe-Post", "List-Unsubscribe=One-Click")
	}
	header("MIME-Version", "1.0")

	var body bytes.Buffer
	contentType, err := writeBody(&body, m)
	if err != nil {
		return nil, err
	}
	header("Content-Type", contentType)
	if !strings.HasPrefix(contentType, "multipart/") {
		header("Content-Transfer-Encoding", "quoted-printable")
	}
	h.WriteString("\r\n")
	h.Write(body.Bytes())
	return h.Bytes(), nil
}

// writeBody writes the body and returns its Content-Type.
func writeBody(w *bytes.Buffer, m Message) (string, error) {
	if len(m.Attachments) == 0 {
		return writeAlternative(w, m)
	}

	mixed := multipart.NewWriter(w)
	var inner bytes.Buffer
	innerType, err := writeAlternative(&inner, m)
	if err != nil {
		return "", err
	}
	partHeader := textproto.MIMEHeader{"Content-Type": {innerType}}
	if !strings.HasPrefix(innerType, "multipart/") {
		partHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	}
	part, err := mixed.CreatePart(partHeader)
	if err != nil {
		return "", err
	}
	if _, err := part.Write(inner.Bytes()); err != nil {
		return "", err
	}

	for _, a := range m.Attachments {
		if strings.ContainsAny(a.Filename+a.ContentType, "\r\n\"") {
			return "", invalid("attachment %q has an invalid name or type", a.Filename)
		}
		ct := a.ContentType
		if ct == "" {
			ct = "application/octet-stream"
		}
		part, err := mixed.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {ct},
			"Content-Transfer-Encoding": {"base64"},
			"Content-Disposition":       {mime.FormatMediaType("attachment", map[string]string{"filename": a.Filename})},
		})
		if err != nil {
			return "", err
		}
		enc := base64.StdEncoding.EncodeToString(a.Content)
		// RFC 2045 caps base64 lines at 76 characters.
		for i := 0; i < len(enc); i += 76 {
			end := min(i+76, len(enc))
			if _, err := io.WriteString(part, enc[i:end]+"\r\n"); err != nil {
				return "", err
			}
		}
	}
	if err := mixed.Close(); err != nil {
		return "", err
	}
	return "multipart/mixed; boundary=" + mixed.Boundary(), nil
}

// writeAlternative writes HTML alone, or text and HTML as multipart/alternative.
// Parts are quoted-printable so long lines never break the 998-character limit.
func writeAlternative(w *bytes.Buffer, m Message) (string, error) {
	const htmlType = "text/html; charset=UTF-8"
	if m.Text == "" {
		return htmlType, writeQP(w, m.HTML)
	}

	alt := multipart.NewWriter(w)
	for _, p := range []struct{ contentType, content string }{
		{"text/plain; charset=UTF-8", m.Text},
		{htmlType, m.HTML},
	} {
		part, err := alt.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {p.contentType},
			"Content-Transfer-Encoding": {"quoted-printable"},
		})
		if err != nil {
			return "", err
		}
		qp := quotedprintable.NewWriter(part)
		if _, err := io.WriteString(qp, p.content); err != nil {
			return "", err
		}
		if err := qp.Close(); err != nil {
			return "", err
		}
	}
	if err := alt.Close(); err != nil {
		return "", err
	}
	return "multipart/alternative; boundary=" + alt.Boundary(), nil
}

func writeQP(w io.Writer, s string) error {
	qp := quotedprintable.NewWriter(w)
	if _, err := io.WriteString(qp, s); err != nil {
		return err
	}
	return qp.Close()
}

func randomID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
