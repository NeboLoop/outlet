package email

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/textproto"
)

// Attachment is a single file attached to an outbound email.
type Attachment struct {
	Filename    string
	ContentType string
	Content     []byte
}

// sendOpts holds optional, backward-compatible send behavior.
type sendOpts struct {
	attachments     []Attachment
	text            string
	listUnsubscribe string
}

// SendOption configures an email send (e.g. attachments) without changing the
// base SendEmailFrom signature for existing callers.
type SendOption func(*sendOpts)

// WithAttachments attaches files to the email. When present, the message is sent
// as a multipart/mixed MIME (SES SendRawEmail / raw SMTP) instead of body-only.
func WithAttachments(atts []Attachment) SendOption {
	return func(o *sendOpts) { o.attachments = atts }
}

// WithText adds a plain-text alternative to the HTML body. HTML-only mail scores
// worse with spam filters, so transactional sends should always carry one.
func WithText(text string) SendOption {
	return func(o *sendOpts) { o.text = text }
}

// WithListUnsubscribe adds RFC 2369 List-Unsubscribe and RFC 8058
// List-Unsubscribe-Post headers pointing at url, so mailbox providers can show
// their own one-click unsubscribe button. Gmail and Yahoo require both headers
// on marketing mail from bulk senders. The url must accept a POST with the body
// "List-Unsubscribe=One-Click" and unsubscribe without further confirmation.
// Never set it on transactional mail.
func WithListUnsubscribe(url string) SendOption {
	return func(o *sendOpts) { o.listUnsubscribe = url }
}

// needsRawMessage reports whether the send needs a full MIME message rather than
// the body-only SES SendEmail call (which cannot carry attachments or headers).
func (o sendOpts) needsRawMessage() bool {
	return len(o.attachments) > 0 || o.listUnsubscribe != ""
}

func applyOpts(opts []SendOption) sendOpts {
	var o sendOpts
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// rawMessage is a complete outbound email, built into RFC 5322 bytes by build
// for SES SendRawEmail and for SMTP.
type rawMessage struct {
	From    string // display form, e.g. "Name <addr>"
	To      string
	ReplyTo string
	Subject string
	HTML    string
	opts    sendOpts
}

// build renders the message. With attachments it is multipart/mixed (HTML part
// plus base64 attachments); otherwise it is a single text/html body.
func (m rawMessage) build() ([]byte, error) {
	var msg bytes.Buffer
	fmt.Fprintf(&msg, "From: %s\r\n", m.From)
	fmt.Fprintf(&msg, "To: %s\r\n", m.To)
	if m.ReplyTo != "" {
		fmt.Fprintf(&msg, "Reply-To: %s\r\n", m.ReplyTo)
	}
	fmt.Fprintf(&msg, "Subject: %s\r\n", m.Subject)
	if m.opts.listUnsubscribe != "" {
		fmt.Fprintf(&msg, "List-Unsubscribe: <%s>\r\n", m.opts.listUnsubscribe)
		msg.WriteString("List-Unsubscribe-Post: List-Unsubscribe=One-Click\r\n")
	}
	msg.WriteString("MIME-Version: 1.0\r\n")

	if len(m.opts.attachments) == 0 {
		msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(m.HTML)
		return msg.Bytes(), nil
	}

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)

	htmlPart, err := mw.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {"text/html; charset=UTF-8"},
		"Content-Transfer-Encoding": {"7bit"},
	})
	if err != nil {
		return nil, err
	}
	if _, err := htmlPart.Write([]byte(m.HTML)); err != nil {
		return nil, err
	}

	for _, a := range m.opts.attachments {
		ct := a.ContentType
		if ct == "" {
			ct = "application/octet-stream"
		}
		part, err := mw.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {ct},
			"Content-Transfer-Encoding": {"base64"},
			"Content-Disposition":       {fmt.Sprintf(`attachment; filename=%q`, a.Filename)},
		})
		if err != nil {
			return nil, err
		}
		enc := base64.StdEncoding.EncodeToString(a.Content)
		// RFC 2045 caps base64 lines at 76 chars.
		for i := 0; i < len(enc); i += 76 {
			end := i + 76
			if end > len(enc) {
				end = len(enc)
			}
			if _, err := part.Write([]byte(enc[i:end] + "\r\n")); err != nil {
				return nil, err
			}
		}
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}

	fmt.Fprintf(&msg, "Content-Type: multipart/mixed; boundary=%s\r\n", mw.Boundary())
	msg.WriteString("\r\n")
	msg.Write(body.Bytes())
	return msg.Bytes(), nil
}
