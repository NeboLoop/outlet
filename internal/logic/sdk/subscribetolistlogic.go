package sdk

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/middleware"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type SubscribeToListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSubscribeToListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SubscribeToListLogic {
	return &SubscribeToListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SubscribeToListLogic) SubscribeToList(req *types.SubscribeRequest) (resp *types.Response, err error) {
	// Get org ID from context (set by API key middleware)
	orgID, ok := l.ctx.Value(middleware.OrgIDKey).(string)
	if !ok {
		return &types.Response{Success: false, Message: "Organization not found"}, nil
	}

	// Validate required fields
	if req.Email == "" {
		return &types.Response{Success: false, Message: "Email is required"}, nil
	}
	if req.Slug == "" {
		return &types.Response{Success: false, Message: "List slug is required"}, nil
	}

	// Get the email list by slug
	list, err := l.svcCtx.DB.GetEmailListByOrgAndSlug(l.ctx, db.GetEmailListByOrgAndSlugParams{
		OrgID: orgID,
		Slug:  req.Slug,
	})
	if err != nil {
		l.Errorf("List not found: %s, error: %v", req.Slug, err)
		return &types.Response{Success: false, Message: "List not found"}, nil
	}

	// Get or create contact
	contact, err := l.svcCtx.DB.GetContactByOrgAndEmail(l.ctx, db.GetContactByOrgAndEmailParams{
		OrgID: sql.NullString{String: orgID, Valid: true},
		Email: req.Email,
	})
	if err != nil {
		// Contact doesn't exist - create it
		contact, err = l.svcCtx.DB.CreateContact(l.ctx, db.CreateContactParams{
			ID:     uuid.New().String(),
			OrgID:  sql.NullString{String: orgID, Valid: true},
			Name:   req.Name,
			Email:  req.Email,
			Source: sql.NullString{String: "list:" + req.Slug, Valid: true},
			Status: "new",
		})
		if err != nil {
			l.Errorf("Failed to create contact: %v", err)
			return &types.Response{Success: false, Message: "Failed to subscribe"}, nil
		}
	}

	// Handle double opt-in if enabled
	if list.DoubleOptin.Valid && list.DoubleOptin.Int64 == 1 {
		// Get org from context (cached by API key middleware)
		org, ok := l.ctx.Value(middleware.OrgKey).(db.Organization)
		if !ok {
			l.Errorf("Organization not found in context")
			return &types.Response{Success: false, Message: "Failed to subscribe"}, nil
		}

		// Generate verification token
		tokenBytes := make([]byte, 32)
		if _, err := rand.Read(tokenBytes); err != nil {
			l.Errorf("Failed to generate verification token: %v", err)
			return &types.Response{Success: false, Message: "Failed to subscribe"}, nil
		}
		verificationToken := hex.EncodeToString(tokenBytes)

		// Subscribe as pending (handles new, pending, and unsubscribed cases)
		subscriber, err := l.svcCtx.DB.SubscribeToListPending(l.ctx, db.SubscribeToListPendingParams{
			ID:                uuid.New().String(),
			ListID:            list.ID,
			ContactID:         contact.ID,
			VerificationToken: sql.NullString{String: verificationToken, Valid: true},
		})
		if err != nil {
			l.Errorf("Failed to add subscriber: %v", err)
			return &types.Response{Success: false, Message: "Failed to subscribe"}, nil
		}

		// If already active, return success without sending email
		if subscriber.Status.String == "active" {
			l.Infof("Already subscribed: %s to list %s", req.Email, req.Slug)
			return &types.Response{Success: true, Message: "Already subscribed"}, nil
		}

		// Save custom field values if provided
		if len(req.CustomFields) > 0 {
			valuesJSON, err := json.Marshal(req.CustomFields)
			if err != nil {
				l.Errorf("Failed to marshal custom fields: %v", err)
			} else {
				err = l.svcCtx.DB.BulkCreateCustomFieldValues(l.ctx, db.BulkCreateCustomFieldValuesParams{
					SubscriberID: subscriber.ID,
					ValuesJson:   string(valuesJSON),
					ListID:       list.ID,
				})
				if err != nil {
					l.Errorf("Failed to save custom field values: %v", err)
					// Non-fatal error - subscription still succeeded
				}
			}
		}

		// Capture values for goroutine
		fromEmail := org.FromEmail.String
		fromName := org.FromName.String
		appUrl := org.AppUrl.String
		if appUrl == "" {
			appUrl = l.svcCtx.Config.App.BaseURL
		}
		toEmail := req.Email
		contactName := contact.Name
		listName := list.Name
		orgName := org.Name
		customSubject := list.ConfirmationEmailSubject.String
		customBody := list.ConfirmationEmailBody.String
		capturedOrgID := orgID

		// Send confirmation email asynchronously
		go func() {
			name := contactName
			if name == "" {
				name = toEmail
			}

			// Get first name from name
			firstName := name
			if idx := len(name); idx > 0 {
				for i, c := range name {
					if c == ' ' {
						firstName = name[:i]
						break
					}
				}
			}

			// Build confirmation link
			confirmUrl := fmt.Sprintf("%s/confirm-email?token=%s", appUrl, verificationToken)

			// Determine subject
			subject := customSubject
			if subject == "" {
				subject = fmt.Sprintf("Confirm your email for %s", listName)
			}
			// Replace variables in subject
			subject = replaceEmailVariables(subject, name, firstName, toEmail, listName, confirmUrl)

			// Determine body
			var body string
			if customBody != "" {
				// Use custom body with variable substitution
				body = replaceEmailVariables(customBody, name, firstName, toEmail, listName, confirmUrl)
			} else {
				// Use default email template
				confirmSection := fmt.Sprintf(`
<p style="margin: 30px 0;">
  <a href="%s" style="display: inline-block; background-color: #4F46E5; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: 500;">
    Confirm your Email
  </a>
</p>
<p style="color: #6b7280; font-size: 14px;">Or copy this link: <a href="%s" style="color: #4F46E5;">%s</a></p>`, confirmUrl, confirmUrl, confirmUrl)

				body = fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="margin: 0; padding: 0; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background-color: #f9fafb;">
  <div style="max-width: 600px; margin: 0 auto; padding: 40px 20px;">
    <div style="background: white; border-radius: 8px; padding: 40px; box-shadow: 0 1px 3px rgba(0,0,0,0.1);">
      <h2 style="margin: 0 0 20px 0; color: #111827;">Confirm your email</h2>
      <p style="color: #374151; line-height: 1.6;">Hi %s,</p>
      <p style="color: #374151; line-height: 1.6;">Please confirm your email to subscribe to <strong>%s</strong>.</p>
      %s
      <p style="color: #6b7280; font-size: 14px; margin-top: 30px;">If you didn't request this subscription, you can safely ignore this email.</p>
    </div>
    <div style="text-align: center; margin-top: 20px; color: #9ca3af; font-size: 12px;">
      <p>%s</p>
    </div>
  </div>
</body>
</html>`, name, listName, confirmSection, orgName)
			}

			// The brand sends only from its own domains.
			err := email.CheckSender(context.Background(), l.svcCtx.DB, capturedOrgID, fromEmail)
			if err == nil {
				err = l.svcCtx.EmailService.SendEmailFrom(
					context.Background(),
					fromEmail,
					fromName,
					toEmail,
					subject,
					body,
				)
			}
			if err != nil {
				logx.Errorf("Failed to send confirmation email to %s: %v", toEmail, err)
			}
		}()

		l.Infof("Confirmation email sent: org=%s email=%s list=%s", orgID, req.Email, req.Slug)
		return &types.Response{Success: true, Message: "Confirmation email sent"}, nil
	}

	// Single opt-in: check if already subscribed
	existing, err := l.svcCtx.DB.GetListSubscriber(l.ctx, db.GetListSubscriberParams{
		ListID:    list.ID,
		ContactID: contact.ID,
	})
	if err == nil && existing.Status.String == "active" {
		l.Infof("Already subscribed: %s to list %s", req.Email, req.Slug)
		return &types.Response{Success: true, Message: "Already subscribed"}, nil
	}

	// Add subscriber to list (immediate active)
	subscriber, err := l.svcCtx.DB.SubscribeToList(l.ctx, db.SubscribeToListParams{
		ID:        uuid.New().String(),
		ListID:    list.ID,
		ContactID: contact.ID,
	})
	if err != nil {
		l.Errorf("Failed to add subscriber: %v", err)
		return &types.Response{Success: false, Message: "Failed to subscribe"}, nil
	}

	// Save custom field values if provided
	if len(req.CustomFields) > 0 {
		valuesJSON, err := json.Marshal(req.CustomFields)
		if err != nil {
			l.Errorf("Failed to marshal custom fields: %v", err)
		} else {
			err = l.svcCtx.DB.BulkCreateCustomFieldValues(l.ctx, db.BulkCreateCustomFieldValuesParams{
				SubscriberID: subscriber.ID,
				ValuesJson:   string(valuesJSON),
				ListID:       list.ID,
			})
			if err != nil {
				l.Errorf("Failed to save custom field values: %v", err)
				// Non-fatal error - subscription still succeeded
			}
		}
	}

	l.Infof("Subscribed: org=%s email=%s list=%s", orgID, req.Email, req.Slug)
	return &types.Response{Success: true, Message: "Subscribed"}, nil
}

// replaceEmailVariables replaces template variables in email content
func replaceEmailVariables(content, name, firstName, email, listName, confirmUrl string) string {
	result := content
	result = strings.ReplaceAll(result, "{{name}}", name)
	result = strings.ReplaceAll(result, "{{first_name}}", firstName)
	result = strings.ReplaceAll(result, "{{email}}", email)
	result = strings.ReplaceAll(result, "{{list_name}}", listName)
	result = strings.ReplaceAll(result, "{{confirm_url}}", confirmUrl)
	return result
}

// replaceTemplateContent replaces {{content}} placeholder in design template
func replaceTemplateContent(template, content string) string {
	return strings.ReplaceAll(template, "{{content}}", content)
}
