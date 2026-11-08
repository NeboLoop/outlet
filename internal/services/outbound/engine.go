package outbound

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/email"
)

// Tick is one pass of the outreach sender: recover crashed sends, then send
// at most one email per mailbox, then check each mailbox's health. It runs
// every 30 seconds; the per-mailbox gap (4 to 9 minutes) does the pacing.
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

// startOfDay is midnight today in a time zone, as stored text.
func startOfDay(now time.Time, loc *time.Location) string {
	l := now.In(loc)
	return ts(time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc))
}

// sendOne sends the mailbox's next due email, if every limit allows it.
func (s *Service) sendOne(ctx context.Context, mb db.Mailbox) error {
	now := s.now()
	if mb.NextSendAt.Valid && mb.NextSendAt.String > ts(now) {
		return nil
	}
	org, err := s.db.GetOrganizationByID(ctx, mb.OrgID)
	if err != nil || !s.Enabled(org) || org.OutboundPausedReason != "" {
		return nil
	}
	sentToday, err := s.db.CountMailboxSentSince(ctx, db.CountMailboxSentSinceParams{MailboxID: sql.NullString{String: mb.ID, Valid: true}, Since: sql.NullString{String: startOfDay(now, email.LoadLocation(org.OutboundTimezone)), Valid: true}})
	if err != nil || sentToday >= allowance(mb, now) {
		return err
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
	_ = s.db.SetMailboxNextSend(ctx, db.SetMailboxNextSendParams{NextSendAt: sql.NullString{String: ts(now.Add(s.gap())), Valid: true}, ID: mb.ID})
	switch {
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
// cross the limits.
func (s *Service) checkHealth(ctx context.Context, mb db.Mailbox) {
	since := ts(s.now().AddDate(0, 0, -7))
	sent, err := s.db.CountMailboxSentSince(ctx, db.CountMailboxSentSinceParams{MailboxID: sql.NullString{String: mb.ID, Valid: true}, Since: sql.NullString{String: since, Valid: true}})
	if err != nil {
		return
	}
	count := func(class string) int64 {
		n, _ := s.db.CountMailboxRepliesSince(ctx, db.CountMailboxRepliesSinceParams{MailboxID: mb.ID, Classification: class, Since: since})
		return n
	}
	bounces, spam, complaints := count("bounce"), count("spam_block"), count("complaint")
	reason := ""
	switch {
	case spam >= spamBlockLimit:
		reason = fmt.Sprintf("%d emails rejected as spam in 7 days; limit %d", spam, spamBlockLimit)
	case bounces >= minBouncesToPause && sent > 0 && float64(bounces)/float64(sent) > bounceRateLimit:
		reason = fmt.Sprintf("bounce rate %.1f%% in 7 days (%d of %d); limit %.0f%%", 100*float64(bounces)/float64(sent), bounces, sent, 100*bounceRateLimit)
	case complaints >= complaintLimit:
		reason = fmt.Sprintf("%d complaints in 7 days; limit %d", complaints, complaintLimit)
	}
	if reason != "" {
		_ = s.db.SetMailboxStatus(ctx, db.SetMailboxStatusParams{Status: "paused", PausedReason: "Health: " + reason, ID: mb.ID})
	}
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
