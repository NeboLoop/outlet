package emails

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/middleware"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type SendEmailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSendEmailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendEmailLogic {
	return &SendEmailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SendEmailLogic) SendEmail(req *types.SendEmailRequest) (resp *types.SendEmailResponse, err error) {
	// Get org from context (set by API key middleware)
	org, ok := l.ctx.Value(middleware.OrgKey).(db.Organization)
	if !ok {
		return &types.SendEmailResponse{
			Success: false,
			Status:  "failed",
			Message: "Organization not found in context",
		}, nil
	}

	// Validate required fields
	if req.To == "" {
		return &types.SendEmailResponse{
			Success: false,
			Status:  "failed",
			Message: "Recipient email (to) is required",
		}, nil
	}

	// Decode optional attachments up front so a bad payload fails before we
	// create a send record.
	var attachments []email.Attachment
	for _, a := range req.Attachments {
		raw, decErr := base64.StdEncoding.DecodeString(a.Content)
		if decErr != nil {
			return &types.SendEmailResponse{
				Success: false,
				Status:  "failed",
				Message: "Invalid base64 content for attachment " + a.Filename,
			}, nil
		}
		attachments = append(attachments, email.Attachment{
			Filename:    a.Filename,
			ContentType: a.ContentType,
			Content:     raw,
		})
	}

	fail := func(message string) (*types.SendEmailResponse, error) {
		return &types.SendEmailResponse{Success: false, Status: "failed", Message: message}, nil
	}

	var subject, htmlBody, plainText string
	var templateID string
	var tmplFromName, tmplFromEmail, tmplReplyTo string

	if req.TemplateSlug != "" {
		template, err := l.svcCtx.DB.GetTransactionalEmailBySlug(l.ctx, db.GetTransactionalEmailBySlugParams{
			Slug:  req.TemplateSlug,
			OrgID: org.ID,
		})
		switch {
		case err != nil:
			return fail("Template '" + req.TemplateSlug + "' not found")
		case template.IsActive.Int64 != 1:
			return fail("Template '" + req.TemplateSlug + "' is inactive")
		}
		templateID = template.ID
		subject = template.Subject
		htmlBody = template.HtmlBody
		plainText = template.PlainText.String
		tmplFromName, tmplFromEmail, tmplReplyTo = template.FromName.String, template.FromEmail.String, template.ReplyTo.String

		for key, value := range req.Variables {
			placeholder := "{{" + key + "}}"
			subject = strings.ReplaceAll(subject, placeholder, value)
			htmlBody = strings.ReplaceAll(htmlBody, placeholder, value)
			plainText = strings.ReplaceAll(plainText, placeholder, value)
		}
	} else {
		if req.Subject == "" {
			return fail("Subject is required when not using a template")
		}
		if req.Body == "" {
			return fail("Body is required when not using a template")
		}
		subject = req.Subject
		htmlBody = req.Body
		plainText = req.TextBody
	}

	// Request fields override the template's; the email service falls back to
	// the brand, then the platform, for anything still empty.
	message := email.Message{
		OrgID:       org.ID,
		FromName:    firstNonEmpty(req.FromName, tmplFromName),
		FromEmail:   firstNonEmpty(req.FromEmail, tmplFromEmail),
		ReplyTo:     firstNonEmpty(req.ReplyTo, tmplReplyTo),
		To:          req.To,
		Subject:     subject,
		HTML:        htmlBody,
		Text:        plainText,
		Attachments: attachments,
		// Threading headers keep a reply in its conversation (the hub's bot
		// email answers threads this way).
		MessageID:  headerValue(req.Headers, "Message-ID"),
		InReplyTo:  headerValue(req.Headers, "In-Reply-To"),
		References: headerValue(req.Headers, "References"),
	}

	trackingToken := generateTrackingToken()

	var contextData sql.NullString
	if len(req.Meta) > 0 || len(req.Tags) > 0 {
		contextMap := make(map[string]interface{})
		if len(req.Meta) > 0 {
			contextMap["meta"] = req.Meta
		}
		if len(req.Tags) > 0 {
			contextMap["tags"] = req.Tags
		}
		jsonBytes, _ := json.Marshal(contextMap)
		contextData = sql.NullString{String: string(jsonBytes), Valid: true}
	}

	if templateID == "" {
		// For ad-hoc sends without a template, we need at least one template to exist
		// Create or get a default "adhoc" template for the org
		adhocTemplate, err := l.getOrCreateAdhocTemplate(org.ID)
		if err != nil {
			l.Errorf("Failed to get adhoc template: %v", err)
			return &types.SendEmailResponse{
				Success: false,
				Status:  "failed",
				Message: "Failed to prepare email for sending",
			}, nil
		}
		templateID = adhocTemplate.ID
	}

	send, err := l.svcCtx.DB.CreateTransactionalSend(l.ctx, db.CreateTransactionalSendParams{
		ID:            uuid.New().String(),
		TemplateID:    templateID,
		OrgID:         org.ID,
		ToEmail:       req.To,
		ToName:        sql.NullString{},
		ContactID:     sql.NullString{},
		Status:        sql.NullString{String: "pending", Valid: true},
		TrackingToken: sql.NullString{String: trackingToken, Valid: true},
		ContextData:   contextData,
	})
	if err != nil {
		l.Errorf("Failed to create send record: %v", err)
		return &types.SendEmailResponse{
			Success: false,
			Status:  "failed",
			Message: "Failed to queue email for sending",
		}, nil
	}

	sendErr := l.svcCtx.EmailService.Send(l.ctx, message)

	if sendErr != nil {
		// Update status to failed
		_ = l.svcCtx.DB.UpdateTransactionalSendStatus(l.ctx, db.UpdateTransactionalSendStatusParams{
			ID:           send.ID,
			Status:       sql.NullString{String: "failed", Valid: true},
			ErrorMessage: sql.NullString{String: sendErr.Error(), Valid: true},
		})

		l.Errorf("Failed to send email: %v", sendErr)
		resp := &types.SendEmailResponse{
			Success:   false,
			MessageId: trackingToken,
			Status:    "failed",
			Message:   "Failed to send email",
		}
		switch {
		case errors.Is(sendErr, email.ErrSuppressed):
			resp.Status = "suppressed"
			resp.Message = "Not sent: the recipient bounced, complained, or is on the suppression list"
		case errors.Is(sendErr, email.ErrInvalidMessage):
			resp.Message = sendErr.Error()
		}
		return resp, nil
	}

	// Update status to sent
	_ = l.svcCtx.DB.UpdateTransactionalSendStatus(l.ctx, db.UpdateTransactionalSendStatusParams{
		ID:           send.ID,
		Status:       sql.NullString{String: "sent", Valid: true},
		ErrorMessage: sql.NullString{},
	})

	l.Infof("SendEmail: org=%s to=%s messageId=%s status=sent", org.ID, req.To, trackingToken)

	return &types.SendEmailResponse{
		Success:   true,
		MessageId: trackingToken,
		Status:    "sent",
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// generateTrackingToken creates a unique tracking token for the email
func generateTrackingToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (l *SendEmailLogic) getOrCreateAdhocTemplate(orgID string) (db.TransactionalEmail, error) {
	template, err := l.svcCtx.DB.GetTransactionalEmailBySlug(l.ctx, db.GetTransactionalEmailBySlugParams{
		Slug:  "_adhoc",
		OrgID: orgID,
	})
	if err == nil {
		return template, nil
	}

	template, err = l.svcCtx.DB.CreateTransactionalEmail(l.ctx, db.CreateTransactionalEmailParams{
		ID:          uuid.New().String(),
		OrgID:       orgID,
		DesignID:    sql.NullInt64{},
		Name:        "Ad-hoc Emails",
		Slug:        "_adhoc",
		Description: sql.NullString{String: "System template for tracking ad-hoc transactional emails", Valid: true},
		Subject:     "{{subject}}",
		HtmlBody:    "{{body}}",
		PlainText:   sql.NullString{},
		FromName:    sql.NullString{},
		FromEmail:   sql.NullString{},
		ReplyTo:     sql.NullString{},
		IsActive:    sql.NullInt64{Int64: 1, Valid: true},
	})
	if err != nil {
		return db.TransactionalEmail{}, err
	}

	return template, nil
}

// sameDomain reports whether two addresses share a domain: a brand may send
// from any address on the domain it is verified for, and only that domain.
func sameDomain(a, b string) bool {
	da, db := domainOf(a), domainOf(b)
	return da != "" && da == db
}

func domainOf(addr string) string {
	addr = strings.ToLower(strings.TrimSpace(addr))
	at := strings.LastIndex(addr, "@")
	if at < 0 || strings.ContainsAny(addr, " <>\r\n") {
		return ""
	}
	return addr[at+1:]
}

// headerValue reads one header from the request's headers, by any case.
func headerValue(h map[string]string, name string) string {
	for k, v := range h {
		if strings.EqualFold(strings.TrimSpace(k), name) {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
