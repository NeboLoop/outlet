package emails

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

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
	// The response says why a send failed; the API answers with it.
	resp, _ = Send(l.ctx, l.svcCtx, org, req, ViaAPI)
	return resp, nil
}

// Via is the way a send came in. A send without a template is recorded under
// the system template of its way in, so SMTP mail stays apart from API mail.
type Via int

const (
	ViaAPI  Via = iota // the SDK route and the admin test send
	ViaSMTP            // the SMTP ingress server
)

// ErrNotSubscribed means a marketing send's recipient isn't an active
// subscriber of its list. It wraps email.ErrSuppressed: the email must not go.
var ErrNotSubscribed = fmt.Errorf("%w: not an active subscriber of the list", email.ErrSuppressed)

// Send sends one transactional or marketing email for a brand and records
// it. It is the one such send: the SDK route calls it for the API key's
// brand, the admin test route for the brand the signed-in member works in,
// and the SMTP server for each recipient of a message. A send that fails
// comes back with Success false and a message, and an error that says the
// kind of failure: email.ErrInvalidMessage for a request that can't be sent
// as it is, email.ErrSuppressed for a recipient who mustn't get it,
// email.ErrOverLimit, or the delivery error.
func Send(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.SendEmailRequest, via Via) (*types.SendEmailResponse, error) {
	l := logx.WithContext(ctx)

	fail := func(message string) (*types.SendEmailResponse, error) {
		return &types.SendEmailResponse{Success: false, Status: "failed", Message: message}, fmt.Errorf("%w: %s", email.ErrInvalidMessage, message)
	}
	failInternal := func(message string, err error) (*types.SendEmailResponse, error) {
		return &types.SendEmailResponse{Success: false, Status: "failed", Message: message}, err
	}

	// Validate required fields
	if req.To == "" {
		return fail("Recipient email (to) is required")
	}

	// Decode optional attachments up front so a bad payload fails before we
	// create a send record.
	var attachments []email.Attachment
	for _, a := range req.Attachments {
		raw, decErr := base64.StdEncoding.DecodeString(a.Content)
		if decErr != nil {
			return fail("Invalid base64 content for attachment " + a.Filename)
		}
		attachments = append(attachments, email.Attachment{
			Filename:    a.Filename,
			ContentType: a.ContentType,
			Content:     raw,
		})
	}

	var subject, htmlBody, plainText string
	var templateID string
	var tmplFromName, tmplFromEmail, tmplReplyTo string

	if req.TemplateSlug != "" {
		template, err := svcCtx.DB.GetTransactionalEmailBySlug(ctx, db.GetTransactionalEmailBySlugParams{
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

		// The template's design wraps its content through ApplyDesign, as
		// campaigns and sequence emails are wrapped; variables are filled
		// after, in the design as in the content.
		if template.DesignID.Valid {
			design, err := svcCtx.DB.GetEmailDesign(ctx, db.GetEmailDesignParams{ID: template.DesignID.Int64, OrgID: org.ID})
			switch {
			case err == nil:
				htmlBody = email.ApplyDesign(design.HtmlBody, htmlBody)
				if plainText != "" {
					plainText = email.ApplyDesign(design.PlainText.String, plainText)
				}
			case !errors.Is(err, sql.ErrNoRows):
				l.Errorf("Failed to load design %d: %v", template.DesignID.Int64, err)
				return failInternal("Failed to prepare email for sending", err)
			}
		}

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

	// Marketing mail goes only to an active subscriber of its list, through
	// the brand's marketing pathway, with an unsubscribe link.
	kind := email.KindTransactional
	var contactID sql.NullString
	switch strings.ToLower(strings.TrimSpace(req.Type)) {
	case "", "transactional":
		if req.ListSlug != "" {
			return fail("list_slug is for marketing mail: set type to marketing, or leave list_slug out")
		}
	case "marketing":
		if req.ListSlug == "" {
			return fail("Marketing mail needs list_slug: the list the recipient subscribed to")
		}
		id, err := activeSubscriber(ctx, svcCtx.DB, org.ID, req.ListSlug, req.To)
		switch {
		case errors.Is(err, ErrNotSubscribed):
			return &types.SendEmailResponse{
				Success: false,
				Status:  "suppressed",
				Message: "Not sent: " + req.To + " isn't an active subscriber of list '" + req.ListSlug + "'",
			}, err
		case errors.Is(err, errNoList):
			return fail("List '" + req.ListSlug + "' not found")
		case err != nil:
			l.Errorf("Failed to check the list subscription: %v", err)
			return failInternal("Failed to prepare email for sending", err)
		}
		kind = email.KindMarketing
		contactID = sql.NullString{String: id, Valid: true}
	default:
		return fail("type must be transactional or marketing")
	}

	// Request fields override the template's; the email service falls back to
	// the brand, then the platform, for anything still empty.
	message := email.Message{
		OrgID:       org.ID,
		Kind:        kind,
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
	svcCtx.EmailService.Track(&message, email.Tracking{
		Token:       trackingToken,
		Unsubscribe: kind == email.KindMarketing,
		Opens:       req.TrackOpens,
		Clicks:      req.TrackClicks,
	})

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
		adhocTemplate, err := getOrCreateAdhocTemplate(ctx, svcCtx.DB, org.ID, via)
		if err != nil {
			l.Errorf("Failed to get adhoc template: %v", err)
			return failInternal("Failed to prepare email for sending", err)
		}
		templateID = adhocTemplate.ID
	}

	send, err := svcCtx.DB.CreateTransactionalSend(ctx, db.CreateTransactionalSendParams{
		ID:            uuid.New().String(),
		TemplateID:    templateID,
		OrgID:         org.ID,
		ToEmail:       req.To,
		ToName:        sql.NullString{},
		ContactID:     contactID,
		Status:        sql.NullString{String: "pending", Valid: true},
		TrackingToken: sql.NullString{String: trackingToken, Valid: true},
		ContextData:   contextData,
	})
	if err != nil {
		l.Errorf("Failed to create send record: %v", err)
		return failInternal("Failed to queue email for sending", err)
	}

	sendErr := svcCtx.EmailService.Send(ctx, message)

	if sendErr != nil {
		// Update status to failed
		_ = svcCtx.DB.UpdateTransactionalSendStatus(ctx, db.UpdateTransactionalSendStatusParams{
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
		case errors.Is(sendErr, email.ErrOverLimit):
			// The brand's transactional pathway is at its limit: the caller
			// sends again once it lifts.
			resp.Status = "limited"
			resp.Message = "Not sent: " + sendErr.Error() + ". Try again after " + email.RetryAt(sendErr, time.Now().Add(15*time.Minute)).UTC().Format(time.RFC3339) + "."
		case errors.Is(sendErr, email.ErrInvalidMessage):
			resp.Message = sendErr.Error()
		}
		return resp, sendErr
	}

	// Update status to sent
	_ = svcCtx.DB.UpdateTransactionalSendStatus(ctx, db.UpdateTransactionalSendStatusParams{
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

// adhocTemplates are the system templates sends without a template are
// recorded under, by the way they came in.
var adhocTemplates = map[Via]struct{ slug, name, description string }{
	ViaAPI:  {"_adhoc", "Ad-hoc Emails", "System template for tracking ad-hoc transactional emails"},
	ViaSMTP: {"_smtp_adhoc", "SMTP Emails", "System template for tracking emails received via SMTP"},
}

func getOrCreateAdhocTemplate(ctx context.Context, store *db.Store, orgID string, via Via) (db.TransactionalEmail, error) {
	t := adhocTemplates[via]
	template, err := store.GetTransactionalEmailBySlug(ctx, db.GetTransactionalEmailBySlugParams{
		Slug:  t.slug,
		OrgID: orgID,
	})
	if err == nil {
		return template, nil
	}

	template, err = store.CreateTransactionalEmail(ctx, db.CreateTransactionalEmailParams{
		ID:          uuid.New().String(),
		OrgID:       orgID,
		DesignID:    sql.NullInt64{},
		Name:        t.name,
		Slug:        t.slug,
		Description: sql.NullString{String: t.description, Valid: true},
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

var errNoList = errors.New("no such list")

// activeSubscriber returns the brand's contact for this address when it is
// an active subscriber of the list: subscribed, and not unsubscribed from
// everything or blocked. Otherwise it returns ErrNotSubscribed, or errNoList.
func activeSubscriber(ctx context.Context, store *db.Store, orgID, listSlug, to string) (string, error) {
	list, err := store.GetEmailListByOrgAndSlug(ctx, db.GetEmailListByOrgAndSlugParams{OrgID: orgID, Slug: listSlug})
	if errors.Is(err, sql.ErrNoRows) {
		return "", errNoList
	}
	if err != nil {
		return "", err
	}
	addr, err := mail.ParseAddress(to)
	if err != nil {
		return "", ErrNotSubscribed
	}
	contact, err := store.GetContactByOrgAndEmail(ctx, db.GetContactByOrgAndEmailParams{
		OrgID: sql.NullString{String: orgID, Valid: true},
		Email: strings.ToLower(addr.Address),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotSubscribed
	}
	if err != nil {
		return "", err
	}
	if contact.UnsubscribedAt.Valid || contact.BlockedAt.Valid {
		return "", ErrNotSubscribed
	}
	sub, err := store.GetListSubscriber(ctx, db.GetListSubscriberParams{ListID: list.ID, ContactID: contact.ID})
	if errors.Is(err, sql.ErrNoRows) || (err == nil && sub.Status.String != "active") {
		return "", ErrNotSubscribed
	}
	if err != nil {
		return "", err
	}
	return contact.ID, nil
}
