package public

import (
	"context"
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/google/uuid"
)

// Handler serves public pages for subscribe, confirm, unsubscribe, and web view
type Handler struct {
	templates    *template.Template
	svcCtx       *svc.ServiceContext
	emailService *email.Service
}

// NewHandler creates a new public pages handler
func NewHandler(svcCtx *svc.ServiceContext) (*Handler, error) {
	tmpl, err := template.ParseFS(Templates, "templates/*.html")
	if err != nil {
		return nil, err
	}

	return &Handler{
		templates:    tmpl,
		svcCtx:       svcCtx,
		emailService: svcCtx.EmailService,
	}, nil
}

// renderTemplate renders an HTML template with the given data
func (h *Handler) renderTemplate(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("Error rendering template %s: %v", name, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// renderError renders the error template
func (h *Handler) renderError(w http.ResponseWriter, title, message string, statusCode int) {
	w.WriteHeader(statusCode)
	h.renderTemplate(w, "error.html", map[string]string{
		"Title":   title,
		"Message": message,
	})
}

// HandleSubscribe handles GET (show form) and POST (process subscription) for /s/{public_id}
func (h *Handler) HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	// Extract public_id from URL path: /s/{public_id}
	publicID := strings.TrimPrefix(r.URL.Path, "/s/")
	if publicID == "" {
		h.renderError(w, "Not Found", "This subscription page doesn't exist.", http.StatusNotFound)
		return
	}

	// Get list by public_id
	list, err := h.svcCtx.DB.GetListByPublicIDForPublicPage(r.Context(), publicID)
	if err != nil {
		if err == sql.ErrNoRows {
			h.renderError(w, "Not Found", "This subscription page doesn't exist or is not available.", http.StatusNotFound)
			return
		}
		log.Printf("Error getting list by public_id %s: %v", publicID, err)
		h.renderError(w, "Error", "Something went wrong. Please try again later.", http.StatusInternalServerError)
		return
	}

	// Fetch custom fields for this list
	customFields, _ := h.svcCtx.DB.ListCustomFieldsByList(r.Context(), list.ID)

	// Build custom fields data for template
	type CustomFieldData struct {
		FieldKey     string
		Name         string
		FieldType    string
		Placeholder  string
		Required     bool
		DefaultValue string
		Options      []string
	}
	var customFieldsData []CustomFieldData
	for _, field := range customFields {
		cf := CustomFieldData{
			FieldKey:  field.FieldKey,
			Name:      field.Name,
			FieldType: field.FieldType,
			Required:  field.Required == 1,
		}
		if field.Placeholder.Valid {
			cf.Placeholder = field.Placeholder.String
		} else {
			cf.Placeholder = field.Name
		}
		if field.DefaultValue.Valid {
			cf.DefaultValue = field.DefaultValue.String
		}
		if field.Options.Valid && field.Options.String != "" {
			_ = json.Unmarshal([]byte(field.Options.String), &cf.Options)
		}
		customFieldsData = append(customFieldsData, cf)
	}

	data := map[string]interface{}{
		"ListName":     list.Name,
		"ListSlug":     list.Slug,
		"ListPublicID": list.PublicID,
		"Description":  list.Description.String,
		"OrgName":      list.OrgName,
		"CustomFields": customFieldsData,
	}

	if r.Method == http.MethodGet {
		h.renderTemplate(w, "subscribe.html", data)
		return
	}

	// POST - process subscription
	if err := r.ParseForm(); err != nil {
		data["Error"] = "Invalid form data"
		h.renderTemplate(w, "subscribe.html", data)
		return
	}

	emailAddr := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	name := strings.TrimSpace(r.FormValue("name"))

	if emailAddr == "" {
		data["Error"] = "Email address is required"
		h.renderTemplate(w, "subscribe.html", data)
		return
	}

	// Check if contact already exists for this org
	existingContact, err := h.svcCtx.DB.GetContactByOrgAndEmail(r.Context(), db.GetContactByOrgAndEmailParams{
		OrgID: sql.NullString{String: list.OrgID, Valid: true},
		Email: emailAddr,
	})

	var contactID string
	if err == sql.ErrNoRows {
		// Create new contact
		contactID = uuid.NewString()
		_, err = h.svcCtx.DB.CreateContact(r.Context(), db.CreateContactParams{
			ID:     contactID,
			OrgID:  sql.NullString{String: list.OrgID, Valid: true},
			Name:   name,
			Email:  emailAddr,
			Source: sql.NullString{String: "public_page", Valid: true},
			Status: sql.NullString{String: "active", Valid: true},
		})
		if err != nil {
			log.Printf("Error creating contact: %v", err)
			data["Error"] = "Something went wrong. Please try again."
			h.renderTemplate(w, "subscribe.html", data)
			return
		}
	} else if err != nil {
		log.Printf("Error checking existing contact: %v", err)
		data["Error"] = "Something went wrong. Please try again."
		h.renderTemplate(w, "subscribe.html", data)
		return
	} else {
		contactID = existingContact.ID
	}

	// Check double opt-in setting
	requiresConfirmation := list.DoubleOptin.Valid && list.DoubleOptin.Int64 == 1

	successData := map[string]interface{}{
		"Email":                emailAddr,
		"ListName":             list.Name,
		"OrgName":              list.OrgName,
		"NeedsConfirmation":    requiresConfirmation,
		"ThankYouRedirectURL":  list.ThankYouUrl.String,
	}

	// Collect custom field values from form
	customFieldValues := make(map[string]string)
	for key, values := range r.Form {
		if strings.HasPrefix(key, "custom_fields[") && strings.HasSuffix(key, "]") {
			fieldKey := key[14 : len(key)-1] // Extract field_key from custom_fields[field_key]
			if len(values) > 0 && values[0] != "" {
				customFieldValues[fieldKey] = values[0]
			}
		}
	}

	var subscriberID string
	if requiresConfirmation {
		// Create pending subscription with verification token
		verificationToken := uuid.NewString()
		subscriber, err := h.svcCtx.DB.SubscribeToListPending(r.Context(), db.SubscribeToListPendingParams{
			ID:                uuid.NewString(),
			ListID:            list.ID,
			ContactID:         contactID,
			VerificationToken: sql.NullString{String: verificationToken, Valid: true},
		})
		if err != nil {
			log.Printf("Error creating pending subscription: %v", err)
			data["Error"] = "Something went wrong. Please try again."
			h.renderTemplate(w, "subscribe.html", data)
			return
		}
		subscriberID = subscriber.ID

		// Send confirmation email
		if err := h.sendConfirmationEmail(r.Context(), list, emailAddr, name, verificationToken); err != nil {
			log.Printf("Error sending confirmation email: %v", err)
			// Still show success - the subscription was created
		}
	} else {
		// Direct subscription (no double opt-in)
		subscriber, err := h.svcCtx.DB.SubscribeToList(r.Context(), db.SubscribeToListParams{
			ID:        uuid.NewString(),
			ListID:    list.ID,
			ContactID: contactID,
		})
		if err != nil {
			log.Printf("Error subscribing to list: %v", err)
			data["Error"] = "Something went wrong. Please try again."
			h.renderTemplate(w, "subscribe.html", data)
			return
		}
		subscriberID = subscriber.ID
	}

	// Save custom field values if any
	if len(customFieldValues) > 0 && subscriberID != "" {
		valuesJSON, err := json.Marshal(customFieldValues)
		if err == nil {
			err = h.svcCtx.DB.BulkCreateCustomFieldValues(r.Context(), db.BulkCreateCustomFieldValuesParams{
				SubscriberID: subscriberID,
				ValuesJson:   string(valuesJSON),
				ListID:       list.ID,
			})
			if err != nil {
				log.Printf("Error saving custom field values: %v", err)
				// Non-fatal - subscription still succeeded
			}
		}
	}

	// Redirect to thank you URL if configured
	if list.ThankYouUrl.Valid && list.ThankYouUrl.String != "" {
		http.Redirect(w, r, list.ThankYouUrl.String, http.StatusSeeOther)
		return
	}

	h.renderTemplate(w, "subscribed.html", successData)
}

// HandleConfirm handles GET for /confirm/{token}
func (h *Handler) HandleConfirm(w http.ResponseWriter, r *http.Request) {
	// Extract token from URL path: /confirm/{token}
	token := strings.TrimPrefix(r.URL.Path, "/confirm/")
	if token == "" {
		h.renderError(w, "Invalid Link", "This confirmation link is invalid.", http.StatusBadRequest)
		return
	}

	// Get subscriber by verification token
	subscriber, err := h.svcCtx.DB.GetListSubscriberByToken(r.Context(), sql.NullString{String: token, Valid: true})
	if err != nil {
		if err == sql.ErrNoRows {
			h.renderError(w, "Invalid Link", "This confirmation link is invalid or has expired.", http.StatusNotFound)
			return
		}
		log.Printf("Error getting subscriber by token: %v", err)
		h.renderError(w, "Error", "Something went wrong. Please try again later.", http.StatusInternalServerError)
		return
	}

	// Confirm the subscription
	confirmedSub, err := h.svcCtx.DB.ConfirmListSubscription(r.Context(), sql.NullString{String: token, Valid: true})
	if err != nil {
		log.Printf("Error confirming subscription: %v", err)
		h.renderError(w, "Error", "Something went wrong. Please try again later.", http.StatusInternalServerError)
		return
	}

	// Get contact email for display
	contact, err := h.svcCtx.DB.GetContact(r.Context(), confirmedSub.ContactID)
	if err != nil {
		log.Printf("Error getting contact: %v", err)
	}

	// Get list info for redirect URL
	list, err := h.svcCtx.DB.GetListByIDForPublicPage(r.Context(), subscriber.ListID)
	if err != nil {
		log.Printf("Error getting list: %v", err)
	}

	// Redirect to confirm redirect URL if configured
	if list.ConfirmRedirectUrl.Valid && list.ConfirmRedirectUrl.String != "" {
		http.Redirect(w, r, list.ConfirmRedirectUrl.String, http.StatusSeeOther)
		return
	}

	data := map[string]interface{}{
		"Email":    contact.Email,
		"ListName": subscriber.ListName,
		"OrgName":  list.OrgName,
	}

	h.renderTemplate(w, "confirm.html", data)
}

// HandleUnsubscribe handles GET (show form) and POST (process unsubscribe) for /u/{token}
func (h *Handler) HandleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	// Extract token from URL path: /u/{token}
	token := strings.TrimPrefix(r.URL.Path, "/u/")
	if token == "" {
		h.renderError(w, "Invalid Link", "This unsubscribe link is invalid.", http.StatusBadRequest)
		return
	}

	// Try to find the send record by tracking token (could be campaign, sequence, or transactional)
	var emailAddr, listName, orgName, redirectURL string
	var contactID string

	// Try campaign sends first
	campaignSend, err := h.svcCtx.DB.GetCampaignSendByTrackingToken(r.Context(), sql.NullString{String: token, Valid: true})
	if err == nil {
		emailAddr = campaignSend.Email
		listName = campaignSend.ListName.String
		orgName = campaignSend.OrgName.String
		contactID = campaignSend.ContactID
		redirectURL = campaignSend.UnsubscribeRedirectUrl.String
	} else {
		// Try email queue (sequences)
		queueRecord, err := h.svcCtx.DB.GetEmailQueueByTrackingToken(r.Context(), sql.NullString{String: token, Valid: true})
		if err == nil {
			emailAddr = queueRecord.Email
			listName = queueRecord.ListName.String
			orgName = queueRecord.OrgName.String
			contactID = queueRecord.ContactID.String
			redirectURL = queueRecord.UnsubscribeRedirectUrl.String
		} else {
			// Try transactional sends
			transSend, err := h.svcCtx.DB.GetTransactionalSendByTrackingToken(r.Context(), sql.NullString{String: token, Valid: true})
			if err == nil {
				emailAddr = transSend.Email
				listName = "" // Transactional emails don't have a list
				orgName = transSend.OrgName
				contactID = transSend.ContactID.String
			} else {
				// Try contact by tracking token (fallback)
				contact, err := h.svcCtx.DB.GetContactByTrackingToken(r.Context(), sql.NullString{String: token, Valid: true})
				if err != nil {
					h.renderError(w, "Invalid Link", "This unsubscribe link is invalid or has expired.", http.StatusNotFound)
					return
				}
				emailAddr = contact.Email
				contactID = contact.ID
				listName = "all emails"
			}
		}
	}

	data := map[string]interface{}{
		"Email":    emailAddr,
		"ListName": listName,
		"OrgName":  orgName,
		"Token":    token,
	}

	if r.Method == http.MethodGet {
		h.renderTemplate(w, "unsubscribe.html", data)
		return
	}

	// POST - process unsubscribe
	if err := r.ParseForm(); err != nil {
		h.renderError(w, "Error", "Invalid form data", http.StatusBadRequest)
		return
	}

	confirm := r.FormValue("confirm")
	if confirm != "yes" {
		h.renderTemplate(w, "unsubscribe.html", data)
		return
	}

	// Perform unsubscribe
	if contactID != "" {
		if err := h.svcCtx.DB.UnsubscribeContact(r.Context(), contactID); err != nil {
			log.Printf("Error unsubscribing contact: %v", err)
		}
		// Also cancel pending emails
		if err := h.svcCtx.DB.CancelEmailsForContact(r.Context(), sql.NullString{String: contactID, Valid: true}); err != nil {
			log.Printf("Error canceling emails for contact: %v", err)
		}
	}

	// Redirect to unsubscribe redirect URL if configured
	if redirectURL != "" {
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}

	h.renderTemplate(w, "unsubscribed.html", data)
}

// HandleWebView handles GET for /w/{token} - view email in browser
func (h *Handler) HandleWebView(w http.ResponseWriter, r *http.Request) {
	// Extract token from URL path: /w/{token}
	token := strings.TrimPrefix(r.URL.Path, "/w/")
	if token == "" {
		h.renderError(w, "Invalid Link", "This link is invalid.", http.StatusBadRequest)
		return
	}

	var subject, htmlBody, orgName string

	// Try campaign sends first
	campaignSend, err := h.svcCtx.DB.GetCampaignSendByTrackingToken(r.Context(), sql.NullString{String: token, Valid: true})
	if err == nil {
		subject = campaignSend.Subject
		htmlBody = campaignSend.HtmlBody
		orgName = campaignSend.OrgName.String
	} else {
		// Try email queue (sequences)
		queueRecord, err := h.svcCtx.DB.GetEmailQueueByTrackingToken(r.Context(), sql.NullString{String: token, Valid: true})
		if err == nil {
			subject = queueRecord.Subject.String
			htmlBody = queueRecord.HtmlBody.String
			orgName = queueRecord.OrgName.String
		} else {
			// Try transactional sends
			transSend, err := h.svcCtx.DB.GetTransactionalSendByTrackingToken(r.Context(), sql.NullString{String: token, Valid: true})
			if err == nil {
				subject = transSend.Subject
				htmlBody = transSend.HtmlBody
				orgName = transSend.OrgName
			} else {
				h.renderError(w, "Not Found", "This email could not be found.", http.StatusNotFound)
				return
			}
		}
	}

	data := map[string]interface{}{
		"Subject":  subject,
		"HTMLBody": template.HTML(htmlBody), // Safe to render as HTML
		"OrgName":  orgName,
	}

	h.renderTemplate(w, "web_view.html", data)
}

// sendConfirmationEmail sends the double opt-in confirmation email
func (h *Handler) sendConfirmationEmail(ctx context.Context, list db.GetListByPublicIDForPublicPageRow, toEmail, toName, token string) error {
	// Build confirmation URL
	baseURL := h.svcCtx.Config.App.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:9888"
	}
	confirmURL := baseURL + "/confirm/" + token

	// Use custom subject/body if configured, otherwise defaults
	subject := "Please confirm your subscription"
	if list.ConfirmationEmailSubject.Valid && list.ConfirmationEmailSubject.String != "" {
		subject = list.ConfirmationEmailSubject.String
	}

	htmlBody := `
		<p>Hi` + conditionalName(toName) + `,</p>
		<p>Please confirm your subscription to <strong>` + list.Name + `</strong> by clicking the link below:</p>
		<p><a href="` + confirmURL + `">Confirm Subscription</a></p>
		<p>If you didn't request this, you can safely ignore this email.</p>
	`
	if list.ConfirmationEmailBody.Valid && list.ConfirmationEmailBody.String != "" {
		// Replace {{confirm_url}} placeholder
		htmlBody = strings.ReplaceAll(list.ConfirmationEmailBody.String, "{{confirm_url}}", confirmURL)
	}

	// Use list's from address if configured, otherwise let email service get defaults from platform_settings
	fromEmail := ""
	fromName := ""
	if list.FromEmail.Valid && list.FromEmail.String != "" {
		fromEmail = list.FromEmail.String
	}
	if list.FromName.Valid && list.FromName.String != "" {
		fromName = list.FromName.String
	}

	// The brand sends only from its own domains.
	if err := email.CheckSender(ctx, h.svcCtx.DB, list.OrgID, fromEmail); err != nil {
		return err
	}
	return h.emailService.SendEmailFrom(ctx, fromEmail, fromName, toEmail, subject, htmlBody)
}

func conditionalName(name string) string {
	if name != "" {
		return " " + name
	}
	return ""
}
