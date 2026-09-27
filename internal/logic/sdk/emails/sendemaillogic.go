package emails

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
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

	var subject, htmlBody, plainText string
	var templateID string

	if req.TemplateSlug != "" {
		// Load template by slug
		template, err := l.svcCtx.DB.GetTransactionalEmailBySlug(l.ctx, db.GetTransactionalEmailBySlugParams{
			Slug:  req.TemplateSlug,
			OrgID: org.ID,
		})
		if err != nil {
			// Template not found - fall back to direct body if provided
			l.Infof("Template '%s' not found, falling back to direct body", req.TemplateSlug)
			if req.Subject != "" && req.Body != "" {
				subject = req.Subject
				htmlBody = req.Body
				plainText = req.TextBody
			} else {
				// No direct body provided either - log warning and use empty content
				l.Infof("Warning: Template '%s' not found and no direct body provided", req.TemplateSlug)
				subject = req.Subject
				htmlBody = req.Body
				if subject == "" {
					subject = "(No Subject)"
				}
				if htmlBody == "" {
					htmlBody = "<p>Email content not available</p>"
				}
			}
		} else if template.IsActive.Int64 != 1 {
			// Template exists but is inactive - fall back to direct body
			l.Infof("Template '%s' is inactive, falling back to direct body", req.TemplateSlug)
			if req.Subject != "" && req.Body != "" {
				subject = req.Subject
				htmlBody = req.Body
				plainText = req.TextBody
			} else {
				subject = req.Subject
				htmlBody = req.Body
				if subject == "" {
					subject = "(No Subject)"
				}
				if htmlBody == "" {
					htmlBody = "<p>Email content not available</p>"
				}
			}
		} else {
			templateID = template.ID
			subject = template.Subject
			htmlBody = template.HtmlBody
			if template.PlainText.Valid {
				plainText = template.PlainText.String
			}

			// Apply variable substitution if provided
			if len(req.Variables) > 0 {
				for key, value := range req.Variables {
					placeholder := "{{" + key + "}}"
					subject = strings.ReplaceAll(subject, placeholder, value)
					htmlBody = strings.ReplaceAll(htmlBody, placeholder, value)
					if plainText != "" {
						plainText = strings.ReplaceAll(plainText, placeholder, value)
					}
				}
			}
		}
	} else {
		// Use direct body
		if req.Subject == "" {
			return &types.SendEmailResponse{
				Success: false,
				Status:  "failed",
				Message: "Subject is required when not using a template",
			}, nil
		}
		if req.Body == "" {
			return &types.SendEmailResponse{
				Success: false,
				Status:  "failed",
				Message: "Body is required when not using a template",
			}, nil
		}
		subject = req.Subject
		htmlBody = req.Body
		plainText = req.TextBody
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

	// Get org email settings for sending from org's configured address
	orgSettings, _ := l.svcCtx.DB.GetOrgEmailSettings(l.ctx, org.ID)
	fromEmail := ""
	fromName := ""
	if orgSettings.FromEmail.Valid {
		fromEmail = orgSettings.FromEmail.String
	}
	if orgSettings.FromName.Valid {
		fromName = orgSettings.FromName.String
	}

	// Per-message sender overrides: a From address on the brand's own sending
	// domain (never another domain — the brand is verified for one), a
	// display name, a Reply-To, a text alternative and threading headers.
	if req.FromEmail != "" && sameDomain(req.FromEmail, fromEmail) {
		fromEmail = strings.TrimSpace(req.FromEmail)
	}
	if req.FromName != "" {
		fromName = strings.TrimSpace(req.FromName)
	}

	// Actually send the email via the email service
	var sendOpts []email.SendOption
	if len(attachments) > 0 {
		sendOpts = append(sendOpts, email.WithAttachments(attachments))
	}
	// Reply-To: the message's own, else the brand's configured Reply-To.
	replyTo := strings.TrimSpace(req.ReplyTo)
	if replyTo == "" && orgSettings.ReplyTo.Valid {
		replyTo = strings.TrimSpace(orgSettings.ReplyTo.String)
	}
	if replyTo != "" {
		sendOpts = append(sendOpts, email.WithReplyTo(replyTo))
	}
	if plainText != "" {
		sendOpts = append(sendOpts, email.WithTextBody(plainText))
	}
	if len(req.Headers) > 0 {
		sendOpts = append(sendOpts, email.WithHeaders(req.Headers))
	}
	sendErr := l.svcCtx.EmailService.SendEmailFrom(l.ctx, fromEmail, fromName, req.To, subject, htmlBody, sendOpts...)

	if sendErr != nil {
		// Update status to failed
		_ = l.svcCtx.DB.UpdateTransactionalSendStatus(l.ctx, db.UpdateTransactionalSendStatusParams{
			ID:           send.ID,
			Status:       sql.NullString{String: "failed", Valid: true},
			ErrorMessage: sql.NullString{String: sendErr.Error(), Valid: true},
		})

		l.Errorf("Failed to send email: %v", sendErr)
		return &types.SendEmailResponse{
			Success:   false,
			MessageId: trackingToken,
			Status:    "failed",
			Message:   "Failed to send email",
		}, nil
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
