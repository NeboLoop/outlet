package outbound

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/email"
)

// Tick is one pass of the outreach sender: recover crashed sends, then send
// at most one email per mailbox, then check each mailbox's health. It runs
// every 30 seconds; the pathway's spacing between sends does the pacing.
func (s *Service) Tick(ctx context.Context) {
	s.recoverStuck(ctx)
	mailboxes, err := s.db.ListActiveMailboxes(ctx)
	if err != nil {
		log.Printf("outbound: list mailboxes: %v", err)
		return
	}
	for _, mb := range mailboxes {
		if err := s.sendOne(ctx, mb); err != nil {
			log.Printf("outbound: %s: %v", mb.Email, err)
		}
		s.checkHealth(ctx, mb)
	}
}

// recoverStuck fails sends left claimed by a crash. Nobody knows whether
// they went, so they are never sent again: a duplicate is worse than a gap.
func (s *Service) recoverStuck(ctx context.Context) {
	stuck, err := s.db.ListStuckOutreach(ctx, sql.NullString{String: ts(s.now().Add(-stuckAfter)), Valid: true})
	if err != nil {
		return
	}
	for _, m := range stuck {
		_ = s.db.MarkEmailFailed(ctx, db.MarkEmailFailedParams{ID: m.ID, ErrorMessage: sql.NullString{String: "interrupted mid-send; not retried, to never send twice", Valid: true}})
	}
}

// sendOne sends the mailbox's next due email. email.Send enforces the
// pathway's limits; a mailbox still inside its spacing is skipped here only
// to save a claim.
func (s *Service) sendOne(ctx context.Context, mb db.Mailbox) error {
	now := s.now()
	if mb.NextSendAt.Valid && mb.NextSendAt.String > ts(now) {
		return nil
	}
	org, err := s.db.GetOrganizationByID(ctx, mb.OrgID)
	if err != nil || !s.Enabled(org) || org.OutboundPausedReason != "" {
		return nil
	}
	m, err := s.db.NextOutreachForMailbox(ctx, db.NextOutreachForMailboxParams{MailboxID: sql.NullString{String: mb.ID, Valid: true}, Now: ts(now)})
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	var seq db.EmailSequence
	if m.SequenceID.Valid {
		if seq, err = s.db.GetSequenceOptions(ctx, m.SequenceID.String); err != nil {
			return err
		}
		// The window may have closed since the email was queued.
		loc := email.LoadLocation(s.contactTimezone(ctx, seq, m.ContactID.String), seq.SendTimezone.String, org.OutboundTimezone)
		if next := email.NextWindow(now, loc, int(seq.WindowStartHour), int(seq.WindowEndHour)); next.After(now) {
			return s.db.SetOutreachSchedule(ctx, db.SetOutreachScheduleParams{ScheduledFor: ts(next), ID: m.ID})
		}
	}
	if n, err := s.db.ClaimOutreachEmail(ctx, db.ClaimOutreachEmailParams{ClaimedAt: sql.NullString{String: ts(now), Valid: true}, ID: m.ID}); err != nil || n == 0 {
		return err
	}

	text, html := m.Body.String, ""
	if host := s.trackingHost(ctx, mb); host != "" && m.TrackingToken.Valid {
		if seq.TrackClicks == 1 {
			text = s.trackLinks(text, host, m.TrackingToken.String)
		}
		if seq.TrackOpens == 1 {
			html = textHTML(text) + `<img src="https://` + host + `/api/e/o/` + m.TrackingToken.String + `" width="1" height="1" alt="">`
		}
	}
	sendErr := s.mailer.Send(ctx, email.Message{
		Kind: email.KindOutreach, OrgID: org.ID, MailboxID: mb.ID,
		To: m.ToEmail, Subject: m.Subject.String, Text: text, HTML: html,
		MessageID: m.MessageID.String, InReplyTo: m.InReplyTo.String, References: m.Refs.String,
		ListUnsubscribe: "mailto:" + mb.Email + "?subject=unsubscribe",
	})
	switch {
	case errors.Is(sendErr, email.ErrOverLimit):
		// Not the mailbox's fault: the email waits until the limit lifts.
		return s.db.ReleaseOutreachEmail(ctx, db.ReleaseOutreachEmailParams{ErrorMessage: sql.NullString{String: sendErr.Error(), Valid: true}, ScheduledFor: ts(email.RetryAt(sendErr, now.Add(15*time.Minute))), ID: m.ID})
	case errors.Is(sendErr, email.ErrSuppressed):
		_, _ = s.db.SkipOutreachEmail(ctx, db.SkipOutreachEmailParams{ID: m.ID, OrgID: sql.NullString{String: org.ID, Valid: true}})
		_ = s.db.MarkEmailFailed(ctx, db.MarkEmailFailedParams{ID: m.ID, ErrorMessage: sql.NullString{String: "suppressed", Valid: true}})
		s.stopContact(ctx, m.ContactID.String, "unsubscribed")
		return nil
	case email.IsTemporary(sendErr):
		_ = s.db.SetMailboxError(ctx, db.SetMailboxErrorParams{LastError: sendErr.Error(), ID: mb.ID})
		return s.db.ReleaseOutreachEmail(ctx, db.ReleaseOutreachEmailParams{ErrorMessage: sql.NullString{String: sendErr.Error(), Valid: true}, ScheduledFor: ts(now.Add(15 * time.Minute)), ID: m.ID})
	case sendErr != nil:
		_ = s.db.SetMailboxError(ctx, db.SetMailboxErrorParams{LastError: sendErr.Error(), ID: mb.ID})
		_ = s.db.MarkEmailFailed(ctx, db.MarkEmailFailedParams{ID: m.ID, ErrorMessage: sql.NullString{String: sendErr.Error(), Valid: true}})
		s.stopContact(ctx, m.ContactID.String, "failed")
		return sendErr
	}
	if err := s.db.MarkOutreachSent(ctx, db.MarkOutreachSentParams{SentAt: sql.NullString{String: ts(now), Valid: true}, ID: m.ID}); err != nil {
		return err
	}
	_ = s.db.SetMailboxError(ctx, db.SetMailboxErrorParams{LastError: "", ID: mb.ID})
	if m.TemplateID.Valid {
		s.seq.AdvanceAfterSend(ctx, m.ContactID.String, m.TemplateID.String)
	}
	return nil
}

// contactTimezone is the person's time zone field on the sequence's list.
func (s *Service) contactTimezone(ctx context.Context, seq db.EmailSequence, contactID string) string {
	if !seq.ListID.Valid {
		return ""
	}
	return s.seq.GetCustomFieldsForContact(ctx, contactID, seq.ListID.Int64)["timezone"]
}

// stopContact ends a person's active outreach and cancels what still waits.
func (s *Service) stopContact(ctx context.Context, contactID, reason string) {
	states, err := s.db.ListActiveOutreachStatesForContact(ctx, sql.NullString{String: contactID, Valid: true})
	if err == nil {
		for _, st := range states {
			_ = s.db.StopSequenceState(ctx, db.StopSequenceStateParams{StoppedReason: reason, ID: st.ID})
		}
	}
	_ = s.db.CancelOpenOutreachForContact(ctx, sql.NullString{String: contactID, Valid: true})
}

// checkHealth pauses a mailbox whose bounces, spam blocks or complaints
// cross its pool's thresholds.
func (s *Service) checkHealth(ctx context.Context, mb db.Mailbox) {
	h, p, err := s.health(ctx, mb)
	if err != nil {
		return
	}
	if reason := email.HealthPause(p, h); reason != "" {
		_ = s.db.SetMailboxStatus(ctx, db.SetMailboxStatusParams{Status: "paused", PausedReason: "Health: " + reason, ID: mb.ID})
	}
}

// health is a mailbox's last 7 days, and its pool.
func (s *Service) health(ctx context.Context, mb db.Mailbox) (email.Health, db.SendingPathway, error) {
	p, err := s.pool(ctx, mb)
	if err != nil {
		return email.Health{}, p, err
	}
	org, err := s.db.GetOrganizationByID(ctx, mb.OrgID)
	if err != nil {
		return email.Health{}, p, err
	}
	now := s.now()
	sent, err := s.mailer.SentSince(ctx, org, p.ID, mb.ID, now.AddDate(0, 0, -7))
	if err != nil {
		return email.Health{}, p, err
	}
	since := ts(now.AddDate(0, 0, -7))
	count := func(class string) int64 {
		n, _ := s.db.CountMailboxRepliesSince(ctx, db.CountMailboxRepliesSinceParams{MailboxID: mb.ID, Classification: class, Since: since})
		return n
	}
	return email.Health{Sent: sent, Bounces: count("bounce"), SpamBlocks: count("spam_block"), Complaints: count("complaint")}, p, nil
}

// textHTML is a plain-text email as minimal HTML, for when opens are tracked
// (a pixel needs an HTML part). Links stay clickable.
func textHTML(text string) string {
	var b strings.Builder
	b.WriteString(`<div style="font-family:sans-serif;white-space:pre-wrap">`)
	for i, line := range strings.Split(text, "\n") {
		if i > 0 {
			b.WriteString("<br>")
		}
		b.WriteString(linkRe.ReplaceAllStringFunc(htmlEscape(line), func(u string) string { return `<a href="` + u + `">` + u + `</a>` }))
	}
	b.WriteString("</div>")
	return b.String()
}
