package smtp

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"

	gosmtp "github.com/emersion/go-smtp"
	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

// EmailProcessor handles parsing and sending emails received via SMTP
type EmailProcessor struct {
	svcCtx     *svc.ServiceContext
	org        db.Organization
	from       string
	recipients []string
}

// NewEmailProcessor creates a new email processor
func NewEmailProcessor(svcCtx *svc.ServiceContext, org db.Organization, from string, recipients []string) *EmailProcessor {
	return &EmailProcessor{
		svcCtx:     svcCtx,
		org:        org,
		from:       from,
		recipients: recipients,
	}
}

// Process parses and sends an email
func (p *EmailProcessor) Process(r io.Reader) (string, error) {
	// Read all data
	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("failed to read email data: %w", err)
	}

	// Parse the email
	msg, err := mail.ReadMessage(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to parse email: %w", err)
	}

	// Extract subject
	subject := msg.Header.Get("Subject")
	if subject == "" {
		subject = "(No Subject)"
	}
	// Decode MIME-encoded subject
	dec := new(mime.WordDecoder)
	if decoded, err := dec.DecodeHeader(subject); err == nil {
		subject = decoded
	}

	// Parse Outlet custom headers
	headers := ParseOutletHeaders(msg)

	// Extract body (HTML and plain text)
	htmlBody, plainText, err := p.extractBody(msg)
	if err != nil {
		return "", fmt.Errorf("failed to extract body: %w", err)
	}

	// From and Reply-To come from the message headers, as any mail client sets
	// them. The brand's own From address wins when it has one; the email
	// service refuses a domain that belongs to another brand.
	var fromName, fromEmail, replyTo string
	if addr, err := mail.ParseAddress(msg.Header.Get("From")); err == nil {
		fromName, fromEmail = addr.Name, addr.Address
	}
	if addr, err := mail.ParseAddress(msg.Header.Get("Reply-To")); err == nil {
		replyTo = addr.Address
	}
	if orgSettings, err := p.svcCtx.DB.GetOrgEmailSettings(context.Background(), p.org.ID); err == nil && orgSettings.FromEmail.String != "" {
		fromName, fromEmail = "", ""
	}

	// Each recipient gets its own send record and tracking token
	var messageIDs []string
	var failed []string
	permanent := true
	for _, recipient := range p.recipients {
		message := email.Message{
			OrgID:     p.org.ID,
			FromName:  fromName,
			FromEmail: fromEmail,
			ReplyTo:   replyTo,
			To:        recipient,
			Subject:   subject,
			HTML:      htmlBody,
			Text:      plainText,
		}
		id, err := p.sendToRecipient(message, headers)
		if err != nil {
			logx.Errorf("SMTP: Failed to send to %s: %v", recipient, err)
			failed = append(failed, recipient)
			if !errors.Is(err, email.ErrSuppressed) && !errors.Is(err, email.ErrInvalidMessage) {
				permanent = false
			}
			continue
		}
		messageIDs = append(messageIDs, id)
	}

	// SMTP answers DATA once for the whole message. When every recipient
	// failed, say so, so the client retries or bounces instead of believing
	// the mail went out. Partial failures are recorded per send.
	if len(messageIDs) == 0 {
		if permanent {
			return "", &gosmtp.SMTPError{Code: 554, EnhancedCode: gosmtp.EnhancedCode{5, 1, 1}, Message: "No recipient could be sent to: " + strings.Join(failed, ", ")}
		}
		return "", &gosmtp.SMTPError{Code: 451, EnhancedCode: gosmtp.EnhancedCode{4, 3, 0}, Message: "Temporary failure sending the message, try again later"}
	}
	return strings.Join(messageIDs, ","), nil
}

// extractBody extracts HTML and plain text body from the email.
//
// Real-world senders (Drupal mimemail, Outlook, Apple Mail…) nest parts —
// multipart/mixed → multipart/related → multipart/alternative — and encode
// them as base64 or quoted-printable, so we walk the tree recursively and
// decode each leaf according to its Content-Transfer-Encoding.
func (p *EmailProcessor) extractBody(msg *mail.Message) (htmlBody, plainText string, err error) {
	htmlBody, plainText = walkPart(msg.Header.Get("Content-Type"), msg.Header.Get("Content-Transfer-Encoding"), msg.Body, 0)

	// If we only have plain text, wrap it in basic HTML
	if htmlBody == "" && plainText != "" {
		htmlBody = "<pre>" + html.EscapeString(plainText) + "</pre>"
	}

	return htmlBody, plainText, nil
}

// walkPart returns the first text/html and text/plain bodies found under a
// MIME part, descending into nested multiparts (bounded to avoid abuse).
func walkPart(contentType, transferEncoding string, body io.Reader, depth int) (htmlBody, plainText string) {
	if depth > 8 {
		return "", ""
	}
	if contentType == "" {
		contentType = "text/plain"
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		raw, _ := io.ReadAll(body)
		return "", string(raw)
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary == "" {
			raw, _ := io.ReadAll(body)
			return "", string(raw)
		}
		mr := multipart.NewReader(body, boundary)
		for {
			part, err := mr.NextPart()
			if err != nil {
				break // io.EOF or malformed remainder: keep what we have
			}
			h, t := walkPart(part.Header.Get("Content-Type"), part.Header.Get("Content-Transfer-Encoding"), part, depth+1)
			if htmlBody == "" {
				htmlBody = h
			}
			if plainText == "" {
				plainText = t
			}
			if htmlBody != "" && plainText != "" {
				break
			}
		}
		return htmlBody, plainText
	}

	raw, _ := io.ReadAll(decodeTransfer(body, transferEncoding))
	text := string(raw)
	switch mediaType {
	case "text/html":
		return text, ""
	case "text/plain":
		return "", text
	}
	return "", "" // attachments/images: ignored
}

// decodeTransfer undoes base64 / quoted-printable transfer encoding.
func decodeTransfer(r io.Reader, encoding string) io.Reader {
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "base64":
		return base64.NewDecoder(base64.StdEncoding, &whitespaceStripper{r: r})
	case "quoted-printable":
		return quotedprintable.NewReader(r)
	}
	return r
}

// whitespaceStripper drops line breaks so wrapped base64 decodes cleanly.
type whitespaceStripper struct{ r io.Reader }

func (w *whitespaceStripper) Read(p []byte) (int, error) {
	n, err := w.r.Read(p)
	out := p[:0]
	for _, b := range p[:n] {
		if b != '\r' && b != '\n' && b != ' ' && b != '\t' {
			out = append(out, b)
		}
	}
	return len(out), err
}

// sendToRecipient records and sends the email to one recipient, returning its
// tracking token.
func (p *EmailProcessor) sendToRecipient(message email.Message, headers *OutletHeaders) (string, error) {
	ctx := context.Background()
	trackingToken := p.generateTrackingToken()

	// Prepare context data (meta + tags)
	var contextData sql.NullString
	if len(headers.Meta) > 0 || len(headers.Tags) > 0 {
		contextMap := make(map[string]interface{})
		if len(headers.Meta) > 0 {
			contextMap["meta"] = headers.Meta
		}
		if len(headers.Tags) > 0 {
			contextMap["tags"] = headers.Tags
		}
		jsonBytes, _ := json.Marshal(contextMap)
		contextData = sql.NullString{String: string(jsonBytes), Valid: true}
	}

	// Get or create adhoc template for tracking
	templateID, err := p.getOrCreateAdhocTemplate()
	if err != nil {
		return "", fmt.Errorf("failed to get adhoc template: %w", err)
	}

	send, err := p.svcCtx.DB.CreateTransactionalSend(ctx, db.CreateTransactionalSendParams{
		ID:            uuid.New().String(),
		TemplateID:    templateID,
		OrgID:         p.org.ID,
		ToEmail:       message.To,
		ToName:        sql.NullString{},
		ContactID:     sql.NullString{},
		Status:        sql.NullString{String: "pending", Valid: true},
		TrackingToken: sql.NullString{String: trackingToken, Valid: true},
		ContextData:   contextData,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create send record: %w", err)
	}

	if sendErr := p.svcCtx.EmailService.Send(ctx, message); sendErr != nil {
		_ = p.svcCtx.DB.UpdateTransactionalSendStatus(ctx, db.UpdateTransactionalSendStatusParams{
			ID:           send.ID,
			Status:       sql.NullString{String: "failed", Valid: true},
			ErrorMessage: sql.NullString{String: sendErr.Error(), Valid: true},
		})
		return "", fmt.Errorf("failed to send: %w", sendErr)
	}

	_ = p.svcCtx.DB.UpdateTransactionalSendStatus(ctx, db.UpdateTransactionalSendStatusParams{
		ID:           send.ID,
		Status:       sql.NullString{String: "sent", Valid: true},
		ErrorMessage: sql.NullString{},
	})

	logx.Infof("SMTP: Email sent to=%s subject=%q org=%s msgId=%s", message.To, message.Subject, p.org.Slug, trackingToken)
	return trackingToken, nil
}

// getOrCreateAdhocTemplate gets or creates the _smtp_adhoc template for SMTP sends
func (p *EmailProcessor) getOrCreateAdhocTemplate() (string, error) {
	ctx := context.Background()

	template, err := p.svcCtx.DB.GetTransactionalEmailBySlug(ctx, db.GetTransactionalEmailBySlugParams{
		Slug:  "_smtp_adhoc",
		OrgID: p.org.ID,
	})
	if err == nil {
		return template.ID, nil
	}

	// Create the template
	template, err = p.svcCtx.DB.CreateTransactionalEmail(ctx, db.CreateTransactionalEmailParams{
		ID:          uuid.New().String(),
		OrgID:       p.org.ID,
		DesignID:    sql.NullInt64{},
		Name:        "SMTP Emails",
		Slug:        "_smtp_adhoc",
		Description: sql.NullString{String: "System template for tracking emails received via SMTP", Valid: true},
		Subject:     "{{subject}}",
		HtmlBody:    "{{body}}",
		PlainText:   sql.NullString{},
		FromName:    sql.NullString{},
		FromEmail:   sql.NullString{},
		ReplyTo:     sql.NullString{},
		IsActive:    sql.NullInt64{Int64: 1, Valid: true},
	})
	if err != nil {
		return "", err
	}

	return template.ID, nil
}

// generateTrackingToken creates a unique tracking token
func (p *EmailProcessor) generateTrackingToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
