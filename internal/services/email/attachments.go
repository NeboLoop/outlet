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
	attachments []Attachment
}

// SendOption configures an email send (e.g. attachments) without changing the
// base SendEmailFrom signature for existing callers.
type SendOption func(*sendOpts)

// WithAttachments attaches files to the email. When present, the message is sent
// as a multipart/mixed MIME (SES SendRawEmail / raw SMTP) instead of body-only.
func WithAttachments(atts []Attachment) SendOption {
	return func(o *sendOpts) { o.attachments = atts }
}

func applyOpts(opts []SendOption) sendOpts {
	var o sendOpts
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// buildRawMessage builds a complete RFC 822 multipart/mixed message (headers +
// HTML part + base64 attachments) suitable for both SES SendRawEmail and SMTP.
func buildRawMessage(from, to, subject, htmlBody string, atts []Attachment) ([]byte, error) {
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)

	htmlPart, err := mw.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {"text/html; charset=UTF-8"},
		"Content-Transfer-Encoding": {"7bit"},
	})
	if err != nil {
		return nil, err
	}
	if _, err := htmlPart.Write([]byte(htmlBody)); err != nil {
		return nil, err
	}

	for _, a := range atts {
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

	var msg bytes.Buffer
	fmt.Fprintf(&msg, "From: %s\r\n", from)
	fmt.Fprintf(&msg, "To: %s\r\n", to)
	fmt.Fprintf(&msg, "Subject: %s\r\n", subject)
	msg.WriteString("MIME-Version: 1.0\r\n")
	fmt.Fprintf(&msg, "Content-Type: multipart/mixed; boundary=%s\r\n", mw.Boundary())
	msg.WriteString("\r\n")
	msg.Write(body.Bytes())
	return msg.Bytes(), nil
}
