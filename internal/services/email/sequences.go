package email

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/db"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

// SequenceService handles email sequence processing
type SequenceService struct {
	db      *db.Store
	sender  *Service
	baseURL string
	clock   func() time.Time // time.Now; outreach tests move it
}

func (s *SequenceService) now() time.Time {
	if s.clock != nil {
		return s.clock()
	}
	return time.Now()
}

// SetClock replaces the clock outreach scheduling reads (tests).
func (s *SequenceService) SetClock(now func() time.Time) { s.clock = now }

// NewSequenceService creates a new sequence service
func NewSequenceService(db *db.Store, sender *Service) *SequenceService {
	baseURL := "https://outlet.sh" // default
	if sender != nil {
		baseURL = sender.GetBaseURL()
	}
	return &SequenceService{
		db:      db,
		sender:  sender,
		baseURL: baseURL,
	}
}

// NewSequenceServiceWithBaseURL creates a new sequence service with custom base URL
func NewSequenceServiceWithBaseURL(db *db.Store, sender *Service, baseURL string) *SequenceService {
	return &SequenceService{
		db:      db,
		sender:  sender,
		baseURL: baseURL,
	}
}

// GetCustomFieldsForContact retrieves custom field values for a contact in a specific list
// Used for merge tag replacement in email templates
func (s *SequenceService) GetCustomFieldsForContact(ctx context.Context, contactID string, listID int64) map[string]string {
	result := make(map[string]string)

	// Get the subscriber ID from contact + list
	subscriber, err := s.db.GetListSubscriber(ctx, db.GetListSubscriberParams{
		ListID:    listID,
		ContactID: contactID,
	})
	if err != nil {
		return result // Return empty map if subscriber not found
	}

	// Get custom field values for this subscriber
	fields, err := s.db.GetSubscriberCustomFieldsForMerge(ctx, subscriber.ID)
	if err != nil {
		return result
	}

	// Build map of field_key -> value
	for _, field := range fields {
		if field.Value.Valid {
			result[field.FieldKey] = field.Value.String
		}
	}

	return result
}

// generateTrackingToken creates a cryptographically secure random token
func generateTrackingToken() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return hex.EncodeToString(bytes)
}

// SendConfirmationEmail sends the confirmation email (position 0) for double opt-in
// Returns the verification token that was generated
func (s *SequenceService) SendConfirmationEmail(ctx context.Context, contactID string, listID int64, triggerEvent string) (string, error) {
	// Get the sequence for this trigger
	sequence, err := s.db.GetSequenceByListAndTrigger(ctx, db.GetSequenceByListAndTriggerParams{
		ListID:       sql.NullInt64{Int64: listID, Valid: true},
		TriggerEvent: triggerEvent,
	})
	if err != nil {
		return "", fmt.Errorf("no email sequence found for list %d trigger %s: %w", listID, triggerEvent, err)
	}

	// Get all templates and find the confirmation template
	templates, err := s.db.ListTemplatesBySequence(ctx, sql.NullString{String: sequence.ID, Valid: true})
	if err != nil {
		return "", fmt.Errorf("failed to get templates: %w", err)
	}

	var confirmationTemplate *db.ListTemplatesBySequenceRow
	for i := range templates {
		if templates[i].TemplateType.Valid && templates[i].TemplateType.String == "confirmation" {
			confirmationTemplate = &templates[i]
			break
		}
	}

	if confirmationTemplate == nil {
		return "", fmt.Errorf("no confirmation template found for sequence %s", sequence.Slug)
	}

	// Get the contact
	contact, err := s.db.GetContactByID(ctx, contactID)
	if err != nil {
		return "", fmt.Errorf("contact not found: %w", err)
	}

	// Generate verification token
	verificationToken := generateTrackingToken()

	// Save token to contact
	err = s.db.SetContactVerificationToken(ctx, db.SetContactVerificationTokenParams{
		ID:    contactID,
		Token: sql.NullString{String: verificationToken, Valid: true},
	})
	if err != nil {
		return "", fmt.Errorf("failed to set verification token: %w", err)
	}

	// Process template variables with confirmation URL
	tplCtx := TemplateContext{
		Name:              contact.Name,
		Email:             contact.Email,
		VerificationToken: verificationToken,
	}
	htmlBody := s.processTemplateVariables(confirmationTemplate.HtmlBody, tplCtx, true)
	subject := s.processTemplateVariables(confirmationTemplate.Subject, tplCtx, false)

	// Confirmation emails don't get link tracking (we want them to click the raw confirmation URL)
	// Also, confirmation templates use template_type='confirmation' which uses raw HTML from DB

	// Send the email directly (not queued)
	err = s.sender.Send(ctx, Message{
		OrgID:   sequence.OrgID.String,
		To:      contact.Email,
		Subject: subject,
		HTML:    htmlBody,
	})
	if err != nil {
		return "", fmt.Errorf("failed to send confirmation email: %w", err)
	}

	logx.Infof("Sent confirmation email to %s for sequence %s", contact.Email, sequence.Slug)
	return verificationToken, nil
}

// TriggerSignup is the trigger of a list's "On signup" autoresponder.
const TriggerSignup = "signup"

// The moments a sequence entry rule (sequence_entry_rules.trigger_type) starts
// its sequence for a contact. source_id names what the moment is about.
const (
	TriggerListJoin         = "list_join"         // a subscription to list source_id became active
	TriggerSequenceComplete = "sequence_complete" // the contact finished sequence source_id
	TriggerTagAdded         = "tag_added"         // the contact was given tag source_id
)

// EntryRuleTriggers are the trigger types an entry rule can have.
var EntryRuleTriggers = []string{TriggerListJoin, TriggerSequenceComplete, TriggerTagAdded}

// CheckEntryRule says why an entry rule with this trigger and source could
// never start its sequence, or nil when it can. Every path that saves a rule
// checks it here.
func CheckEntryRule(trigger, sourceID string) error {
	if !slices.Contains(EntryRuleTriggers, trigger) {
		return fmt.Errorf("trigger_type must be one of: %s", strings.Join(EntryRuleTriggers, ", "))
	}
	if strings.TrimSpace(sourceID) == "" {
		return errors.New("source_id is required: the list ID, sequence ID or tag the rule waits on")
	}
	return nil
}

// SubscriptionActivated is called whenever a contact's subscription to a list
// becomes active: a double opt-in confirmed, or a single opt-in subscribe. It
// starts the list's signup sequence and every sequence whose entry rule waits
// on this list. A contact already in a sequence stays where they are, so
// calling it again for the same subscription is safe.
func (s *SequenceService) SubscriptionActivated(ctx context.Context, contactID string, listID int64) error {
	return errors.Join(
		s.StartSequence(ctx, contactID, listID, TriggerSignup),
		s.runEntryRules(ctx, TriggerListJoin, strconv.FormatInt(listID, 10), contactID),
	)
}

// TagAdded starts every sequence whose entry rule waits on this tag. The store
// calls it whenever a contact is given a tag it didn't have.
func (s *SequenceService) TagAdded(ctx context.Context, contactID, tag string) {
	if err := s.runEntryRules(ctx, TriggerTagAdded, tag, contactID); err != nil {
		logx.Errorf("Entry rules for tag %q on contact %s: %v", tag, contactID, err)
	}
}

// runEntryRules starts, through StartSequenceByID, the sequence of each active
// entry rule of the contact's brand waiting on this trigger and source.
func (s *SequenceService) runEntryRules(ctx context.Context, trigger, sourceID, contactID string) error {
	rules, err := s.db.GetActiveEntryRuleForTrigger(ctx, db.GetActiveEntryRuleForTriggerParams{
		ContactID: contactID, TriggerType: trigger, SourceID: sourceID,
	})
	if err != nil {
		return fmt.Errorf("entry rules for %s %s: %w", trigger, sourceID, err)
	}
	var errs []error
	for _, rule := range rules {
		if err := s.StartSequenceByID(ctx, contactID, rule.SequenceID); err != nil {
			errs = append(errs, fmt.Errorf("entry rule %s: %w", rule.ID, err))
		}
	}
	return errors.Join(errs...)
}

// StartSequence starts, through StartSequenceByID, every active sequence of
// the list that this trigger starts: a list can have several "On signup"
// autoresponders, and each runs once per contact.
func (s *SequenceService) StartSequence(ctx context.Context, contactID string, listID int64, triggerEvent string) error {
	sequences, err := s.db.ListActiveSequencesByListAndTrigger(ctx, db.ListActiveSequencesByListAndTriggerParams{
		ListID:       sql.NullInt64{Int64: listID, Valid: true},
		TriggerEvent: triggerEvent,
	})
	if err != nil {
		return fmt.Errorf("sequences for list %d trigger %s: %w", listID, triggerEvent, err)
	}
	var errs []error
	for _, seq := range sequences {
		if err := s.StartSequenceByID(ctx, contactID, seq.ID); err != nil {
			errs = append(errs, fmt.Errorf("sequence %s: %w", seq.Slug, err))
		}
	}
	return errors.Join(errs...)
}

// StartSequenceByID initiates an email sequence for a contact by sequence ID directly
func (s *SequenceService) StartSequenceByID(ctx context.Context, contactID string, sequenceID string) error {
	// Get the sequence
	sequence, err := s.db.GetSequenceByID(ctx, sequenceID)
	if err != nil {
		return fmt.Errorf("sequence not found: %w", err)
	}

	// Check if contact is already in this sequence
	_, err = s.db.GetContactSequenceState(ctx, db.GetContactSequenceStateParams{
		ContactID:  sql.NullString{String: contactID, Valid: true},
		SequenceID: sql.NullString{String: sequence.ID, Valid: true},
	})
	if err == nil {
		// Already in sequence
		logx.Infof("Contact %s already in sequence %s", contactID, sequence.Slug)
		return nil
	}

	// Create sequence state
	_, err = s.db.CreateContactSequenceState(ctx, db.CreateContactSequenceStateParams{
		ID:              uuid.NewString(),
		ContactID:       sql.NullString{String: contactID, Valid: true},
		SequenceID:      sql.NullString{String: sequence.ID, Valid: true},
		CurrentPosition: sql.NullInt64{Int64: 0, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to create sequence state: %w", err)
	}

	// Queue the first email (position 1)
	err = s.queueNextEmail(ctx, contactID, sequence.ID, 1)
	if err != nil {
		logx.Errorf("Failed to queue first email: %v", err)
	}

	logx.Infof("Started email sequence %s (ID %s) for contact %s", sequence.Slug, sequence.ID, contactID)
	return nil
}

// queueNextEmail queues the next email in the sequence
func (s *SequenceService) queueNextEmail(ctx context.Context, contactID string, sequenceID string, position int64) error {
	if seq, err := s.db.GetSequenceOptions(ctx, sequenceID); err == nil && seq.SequenceType.String == "outreach" {
		return s.queueOutreach(ctx, seq, contactID, position)
	}
	// Get the template at this position
	template, err := s.db.GetNextTemplate(ctx, db.GetNextTemplateParams{
		SequenceID: sql.NullString{String: sequenceID, Valid: true},
		Position:   position,
	})
	if err != nil {
		// No more templates - sequence complete
		return nil
	}

	// Get the sequence to check for send_hour setting
	sequence, err := s.db.GetSequenceByID(ctx, sequenceID)
	if err != nil {
		return fmt.Errorf("failed to get sequence: %w", err)
	}

	// Calculate scheduled time
	var scheduledFor time.Time
	if sequence.SendHour.Valid {
		// Use fixed send time - schedule for the next occurrence of send_hour in the sequence timezone
		scheduledFor = s.calculateNextSendTime(int32(sequence.SendHour.Int64), sequence.SendTimezone.String, int32(template.DelayHours))
	} else {
		// Use delay_hours from opt-in time (original behavior)
		scheduledFor = time.Now().Add(time.Duration(template.DelayHours) * time.Hour)
	}

	// Generate tracking token for this email
	trackingToken := generateTrackingToken()

	// Queue the email with tracking token
	_, err = s.db.QueueEmail(ctx, db.QueueEmailParams{
		ID:            uuid.NewString(),
		ContactID:     sql.NullString{String: contactID, Valid: true},
		TemplateID:    sql.NullString{String: template.ID, Valid: true},
		ScheduledFor:  scheduledFor.UTC().Format(time.RFC3339),
		TrackingToken: sql.NullString{String: trackingToken, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to queue email: %w", err)
	}

	logx.Infof("Queued email template %s for contact %s at %s", template.ID, contactID, scheduledFor.Format(time.RFC3339))
	return nil
}

// calculateNextSendTime calculates the next send time based on send_hour and timezone
// delay_hours is used to determine which day to send (e.g., delay_hours=24 means tomorrow)
func (s *SequenceService) calculateNextSendTime(sendHour int32, timezone string, delayHours int32) time.Time {
	// Load the timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		// Fall back to America/New_York if timezone is invalid
		loc, _ = time.LoadLocation("America/New_York")
		logx.Errorf("Invalid timezone %s, falling back to America/New_York", timezone)
	}

	// Get current time in the target timezone
	now := time.Now().In(loc)

	// Calculate the target date based on delay_hours
	// delay_hours / 24 gives us the number of days to add
	daysToAdd := int(delayHours) / 24
	if daysToAdd < 1 {
		daysToAdd = 1 // Minimum 1 day delay for send_hour mode
	}

	// Create the target datetime: today + daysToAdd at send_hour
	targetDate := now.AddDate(0, 0, daysToAdd)
	targetTime := time.Date(
		targetDate.Year(), targetDate.Month(), targetDate.Day(),
		int(sendHour), 0, 0, 0, loc,
	)

	// If somehow the target time is in the past (edge case), add another day
	if targetTime.Before(time.Now()) {
		targetTime = targetTime.AddDate(0, 0, 1)
	}

	return targetTime
}

// ProcessPendingEmails sends all pending emails that are due
func (s *SequenceService) ProcessPendingEmails(ctx context.Context, batchSize int64) (int, error) {
	// Get pending emails
	pendingEmails, err := claimDueEmails(ctx, s.db, batchSize)
	if err != nil {
		return 0, fmt.Errorf("failed to get pending emails: %w", err)
	}

	sent := 0
	for _, email := range pendingEmails {
		subject, htmlBody := s.renderQueued(ctx, email)

		// Send the email
		err := s.sender.Send(ctx, s.sequenceMessage(email, subject, htmlBody))
		if s.deferIfTemporary(ctx, email, err) {
			continue
		}
		if err != nil {
			logx.Errorf("Failed to send email %s to %s: %v", email.ID, email.Email, err)
			_ = s.db.MarkEmailFailed(ctx, db.MarkEmailFailedParams{
				ID:           email.ID,
				ErrorMessage: sql.NullString{String: err.Error(), Valid: true},
			})
			continue
		}

		// Mark as sent
		err = s.db.MarkEmailSent(ctx, db.MarkEmailSentParams{ID: email.ID, Subject: sql.NullString{String: subject, Valid: true}})
		if err != nil {
			logx.Errorf("Failed to mark email %s as sent: %v", email.ID, err)
			continue
		}

		sent++
		logx.Infof("Sent email %s to %s: %s", email.ID, email.Email, subject)

		s.advanceSequence(ctx, email)
	}

	return sent, nil
}

// renderQueued builds a queued sequence email's subject and HTML. The content
// goes into the template's design, or when it has none the built-in layout its
// template_type names ('simple' footer, 'branded' header and footer, 'none'
// raw), through ApplyDesign as campaigns do. Merge tags are then filled in
// the whole email and its links rewritten for click tracking.
func (s *SequenceService) renderQueued(ctx context.Context, email db.GetPendingEmailsRow) (subject, htmlBody string) {
	tplCtx := TemplateContext{
		Name:          email.Name,
		Email:         email.Email,
		TrackingToken: email.TrackingToken.String,
		CustomFields:  make(map[string]string),
	}
	if email.ContactID.Valid && email.TemplateID.Valid {
		template, err := s.db.GetTemplateByID(ctx, email.TemplateID.String)
		if err == nil && template.SequenceID.Valid {
			sequence, err := s.db.GetSequenceByID(ctx, template.SequenceID.String)
			if err == nil && sequence.ListID.Valid {
				tplCtx.CustomFields = s.GetCustomFieldsForContact(ctx, email.ContactID.String, sequence.ListID.Int64)
			}
		}
	}

	design := email.DesignHtml
	if design == "" {
		isTransactional := email.IsTransactional.Valid && email.IsTransactional.Int64 == 1
		design = s.builtinLayout(email.TemplateType.String, isTransactional, email.TrackingToken.String)
	}
	htmlBody = s.processTemplateVariables(ApplyDesign(design, email.HtmlBody), tplCtx, true)
	if email.TrackingToken.Valid && email.TrackingToken.String != "" {
		htmlBody = s.rewriteLinksForTracking(htmlBody, email.TrackingToken.String)
	}
	return s.processTemplateVariables(email.Subject, tplCtx, false), htmlBody
}

// builtinLayout is the design a sequence email without one of its own gets,
// by its template_type: 'none' is no layout, 'branded' a header and footer,
// anything else ('simple') a footer. The unsubscribe link is left out of
// transactional steps.
func (s *SequenceService) builtinLayout(templateType string, isTransactional bool, trackingToken string) string {
	switch templateType {
	case "none":
		return ""
	case "branded":
		return s.brandedLayout(isTransactional, trackingToken)
	default:
		return s.simpleLayout(isTransactional, trackingToken)
	}
}

// deferIfTemporary puts a sequence email back in the queue, unclaimed and
// pending, when its send failed only for now (its pathway's limit, a
// mailbox's 4xx), to go again when the limit lifts. A send the mail server
// keeps refusing is given up after GiveUpAfter. It reports whether it did.
func (s *SequenceService) deferIfTemporary(ctx context.Context, email db.GetPendingEmailsRow, err error) bool {
	if err == nil || !IsTemporary(err) {
		return false
	}
	now := s.now()
	refused := RefusedAt(err, now)
	if refused.Valid {
		if since, _ := s.db.QueuedEmailRefusedSince(ctx, email.ID); GaveUp(since, now) {
			logx.Infof("Email %s to %s: refused for %s, giving up: %v", email.ID, email.Email, GiveUpAfter, err)
			return false
		}
	}
	at := RetryAt(err, now.Add(15*time.Minute))
	if derr := s.db.DeferQueuedEmail(ctx, db.DeferQueuedEmailParams{ScheduledFor: at.UTC().Format(time.RFC3339), RefusedAt: refused, ID: email.ID}); derr != nil {
		logx.Errorf("Failed to defer email %s: %v", email.ID, derr)
	}
	logx.Infof("Email %s to %s waits until %s: %v", email.ID, email.Email, at.UTC().Format(time.RFC3339), err)
	return true
}

// claimDueEmails claims up to limit due sequence emails for this worker and
// returns them. A claimed row is never handed to another worker, so polling
// again while these wait for the rate limiter cannot send them twice.
func claimDueEmails(ctx context.Context, store *db.Store, limit int64) ([]db.GetPendingEmailsRow, error) {
	claim := uuid.NewString()
	if err := store.ClaimPendingEmails(ctx, db.ClaimPendingEmailsParams{
		ClaimToken:      sql.NullString{String: claim, Valid: true},
		ScheduledBefore: time.Now().UTC().Format(time.RFC3339),
		LimitCount:      limit,
	}); err != nil {
		return nil, err
	}
	return store.GetPendingEmails(ctx, sql.NullString{String: claim, Valid: true})
}

// advanceSequence runs after a sequence email is sent: it records the
// position, queues the next step, and when the sequence is finished marks it
// complete and starts the sequences waiting on its completion: those of its
// entry rules, and the chained sequence, if one is configured.
func (s *SequenceService) advanceSequence(ctx context.Context, email db.GetPendingEmailsRow) {
	if !email.ContactID.Valid || !email.TemplateID.Valid {
		return
	}
	template, err := s.db.GetTemplateByID(ctx, email.TemplateID.String)
	if err != nil || !template.SequenceID.Valid {
		return
	}

	_ = s.db.UpdateContactSequencePosition(ctx, db.UpdateContactSequencePositionParams{
		ContactID:       email.ContactID,
		SequenceID:      template.SequenceID,
		CurrentPosition: sql.NullInt64{Int64: template.Position, Valid: true},
	})

	if err := s.queueNextEmail(ctx, email.ContactID.String, template.SequenceID.String, template.Position+1); err != nil {
		logx.Errorf("Failed to queue next email: %v", err)
	}

	nextTemplate, err := s.db.GetNextTemplate(ctx, db.GetNextTemplateParams{
		SequenceID: template.SequenceID,
		Position:   template.Position + 1,
	})
	if err == nil && nextTemplate.ID != "" {
		return
	}

	_ = s.db.CompleteContactSequence(ctx, db.CompleteContactSequenceParams{
		ContactID:  email.ContactID,
		SequenceID: template.SequenceID,
	})
	logx.Infof("Completed sequence for contact %s", email.ContactID.String)

	if err := s.runEntryRules(ctx, TriggerSequenceComplete, template.SequenceID.String, email.ContactID.String); err != nil {
		logx.Errorf("Entry rules after sequence %s for contact %s: %v", template.SequenceID.String, email.ContactID.String, err)
	}

	seq, err := s.db.GetSequenceByID(ctx, template.SequenceID.String)
	if err != nil || !seq.OnCompletionSequenceID.Valid || seq.OnCompletionSequenceID.String == "" {
		return
	}
	if err := s.StartSequenceByID(ctx, email.ContactID.String, seq.OnCompletionSequenceID.String); err != nil {
		logx.Errorf("Failed to start chained sequence %s for contact %s: %v",
			seq.OnCompletionSequenceID.String, email.ContactID.String, err)
		return
	}
	logx.Infof("Chained contact %s from sequence %s to sequence %s",
		email.ContactID.String, template.SequenceID.String, seq.OnCompletionSequenceID.String)
}

// TemplateContext holds variables for template processing
type TemplateContext struct {
	Name              string
	Email             string
	VerificationToken string
	TrackingToken     string
	CustomFields      map[string]string // Custom field values keyed by field_key
}

// processTemplateVariables replaces placeholders in email content
// processTemplateVariables fills merge tags; escapeHTML is true for HTML bodies
func (s *SequenceService) processTemplateVariables(content string, ctx TemplateContext, escapeHTML bool) string {
	return FillMergeTags(content, MergeContext{
		BaseURL:           s.baseURL,
		Name:              ctx.Name,
		Email:             ctx.Email,
		TrackingToken:     ctx.TrackingToken,
		VerificationToken: ctx.VerificationToken,
		CustomFields:      ctx.CustomFields,
	}, escapeHTML)
}

// sequenceMessage addresses a queued sequence email from its brand. A step is
// marketing mail unless it is marked transactional; marketing steps carry the
// one-click List-Unsubscribe headers (when they have a tracking token).
func (s *SequenceService) sequenceMessage(email db.GetPendingEmailsRow, subject, htmlBody string) Message {
	m := Message{OrgID: email.OrgID, Kind: KindMarketing, To: email.Email, Subject: subject, HTML: htmlBody}
	if email.IsTransactional.Valid && email.IsTransactional.Int64 == 1 {
		m.Kind = KindTransactional
	} else if email.TrackingToken.String != "" {
		m.ListUnsubscribe = unsubscribeURL(s.baseURL, email.TrackingToken.String)
	}
	return m
}

// simpleLayout is the built-in design with just a footer.
func (s *SequenceService) simpleLayout(isTransactional bool, trackingToken string) string {
	unsubscribeSection := ""
	if !isTransactional && trackingToken != "" {
		link := unsubscribeURL(s.baseURL, trackingToken)
		unsubscribeSection = fmt.Sprintf(`<p style="margin-top: 15px;"><a href="%s">Unsubscribe</a></p>`, link)
	}

	// Use baseURL for footer link, fall back to empty if not set
	footerLink := ""
	if s.baseURL != "" {
		footerLink = fmt.Sprintf(`<p><a href="%s">%s</a></p>`, s.baseURL, s.baseURL)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; }
        .button { display: inline-block; background: #f97316; color: white; padding: 14px 28px; text-decoration: none; border-radius: 6px; font-weight: bold; }
        .footer { text-align: center; color: #6b7280; font-size: 12px; padding: 30px 0 10px 0; border-top: 1px solid #e5e7eb; margin-top: 30px; }
        .footer a { color: #6b7280; }
        p { margin: 0 0 16px 0; }
        a { color: #f97316; }
    </style>
</head>
<body>
    {{content}}
    <div class="footer">
        %s
        %s
    </div>
</body>
</html>`, footerLink, unsubscribeSection)
}

// brandedLayout is the built-in design with a header and footer.
func (s *SequenceService) brandedLayout(isTransactional bool, trackingToken string) string {
	unsubscribeSection := ""
	if !isTransactional && trackingToken != "" {
		link := unsubscribeURL(s.baseURL, trackingToken)
		unsubscribeSection = fmt.Sprintf(`<p style="margin-top: 15px;"><a href="%s">Unsubscribe</a></p>`, link)
	}

	// Use baseURL for footer link, fall back to empty if not set
	footerLink := ""
	if s.baseURL != "" {
		footerLink = fmt.Sprintf(`<p><a href="%s">%s</a></p>`, s.baseURL, s.baseURL)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; }
        .header { background: linear-gradient(135deg, #0f172a 0%%, #1e293b 100%%); color: white; padding: 30px; text-align: center; }
        .header h1 { font-size: 24px; margin: 0; font-weight: 600; }
        .content { padding: 30px; }
        .button { display: inline-block; background: #f97316; color: white; padding: 14px 28px; text-decoration: none; border-radius: 6px; font-weight: bold; }
        .footer { text-align: center; color: #6b7280; font-size: 12px; padding: 20px; background: #f8fafc; border-top: 1px solid #e5e7eb; }
        .footer a { color: #6b7280; }
        p { margin: 0 0 16px 0; }
        a { color: #f97316; }
    </style>
</head>
<body>
    <div class="content">
        {{content}}
    </div>
    <div class="footer">
        %s
        %s
    </div>
</body>
</html>`, footerLink, unsubscribeSection)
}

// rewriteLinksForTracking replaces href URLs with signed click-tracking URLs
func (s *SequenceService) rewriteLinksForTracking(htmlBody, trackingToken string) string {
	var key []byte
	if s.sender != nil {
		key = s.sender.linkKey
	}
	return rewriteLinks(htmlBody, s.baseURL, key, trackingToken)
}
