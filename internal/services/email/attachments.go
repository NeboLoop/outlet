package email

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"strings"
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
	replyTo     string
	textBody    string
	headers     map[string]string
}

// threadingHeaders are the only extra headers a sender may set: the ones a
// reply needs to thread in the recipient's mail client.
var threadingHeaders = map[string]string{
	"in-reply-to": "In-Reply-To",
	"references":  "References",
	"message-id":  "Message-ID",
}

// WithReplyTo sets the Reply-To header.
func WithReplyTo(addr string) SendOption {
	return func(o *sendOpts) { o.replyTo = strings.TrimSpace(addr) }
}

// WithTextBody adds a plain-text alternative to the HTML body.
func WithTextBody(text string) SendOption {
	return func(o *sendOpts) { o.textBody = text }
}

// WithHeaders sets threading headers (In-Reply-To, References, Message-ID);
// any other name, and any value with a line break, is ignored.
func WithHeaders(h map[string]string) SendOption {
	return func(o *sendOpts) {
		for k, v := range h {
			name, ok := threadingHeaders[strings.ToLower(strings.TrimSpace(k))]
			if !ok || v == "" || strings.ContainsAny(v, "\r\n") {
				continue
			}
			if o.headers == nil {
				o.headers = map[string]string{}
			}
			o.headers[name] = strings.TrimSpace(v)
		}
	}
}

// needsRaw: anything the SES SendEmail API can't carry (attachments, a text
// alternative, threading headers) is sent as a raw MIME message.
func (o sendOpts) needsRaw() bool {
	return len(o.attachments) > 0 || o.textBody != "" || len(o.headers) > 0
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

// buildRawMessage builds a complete RFC 822 message suitable for both SES
// SendRawEmail and SMTP: the HTML body (with a plain-text alternative when
// one is given), base64 attachments, Reply-To and threading headers.
func buildRawMessage(from, to, subject, htmlBody string, o sendOpts) ([]byte, error) {
	// Address headers are written verbatim: a line break in any of them
	// would inject headers (SES SendRawEmail delivers to a smuggled Bcc).
	for _, v := range []string{from, to, o.replyTo} {
		if strings.ContainsAny(v, "\r\n") {
			return nil, fmt.Errorf("invalid address header %q", v)
		}
	}

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)

	if o.textBody != "" {
		alt := &bytes.Buffer{}
		aw := multipart.NewWriter(alt)
		for _, p := range []struct{ ct, content string }{
			{"text/plain; charset=UTF-8", o.textBody},
			{"text/html; charset=UTF-8", htmlBody},
		} {
			w, err := aw.CreatePart(textproto.MIMEHeader{
				"Content-Type":              {p.ct},
				"Content-Transfer-Encoding": {"quoted-printable"},
			})
			if err != nil {
				return nil, err
			}
			qw := quotedprintable.NewWriter(w)
			if _, err := qw.Write([]byte(p.content)); err != nil {
				return nil, err
			}
			if err := qw.Close(); err != nil {
				return nil, err
			}
		}
		if err := aw.Close(); err != nil {
			return nil, err
		}
		part, err := mw.CreatePart(textproto.MIMEHeader{
			"Content-Type": {"multipart/alternative; boundary=" + aw.Boundary()},
		})
		if err != nil {
			return nil, err
		}
		if _, err := part.Write(alt.Bytes()); err != nil {
			return nil, err
		}
	} else {
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
	}

	for _, a := range o.attachments {
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
	if o.replyTo != "" {
		fmt.Fprintf(&msg, "Reply-To: %s\r\n", o.replyTo)
	}
	fmt.Fprintf(&msg, "Subject: %s\r\n", mime.QEncoding.Encode("UTF-8", subject))
	for _, name := range []string{"Message-ID", "In-Reply-To", "References"} {
		if v := o.headers[name]; v != "" {
			fmt.Fprintf(&msg, "%s: %s\r\n", name, v)
		}
	}
	msg.WriteString("MIME-Version: 1.0\r\n")
	fmt.Fprintf(&msg, "Content-Type: multipart/mixed; boundary=%s\r\n", mw.Boundary())
	msg.WriteString("\r\n")
	msg.Write(body.Bytes())
	return msg.Bytes(), nil
}
