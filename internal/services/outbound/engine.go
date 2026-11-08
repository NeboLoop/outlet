package outbound

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
)

// Tick is one pass of the sending engine: recover crashed sends, draft due
// follow-ups, send at most one email per mailbox, then check health. It is
// called every 30 seconds, and the per-mailbox gap (4 to 9 minutes) does the
// pacing.
func (s *Service) Tick(ctx context.Context) {
	s.recoverStuck(ctx)
	s.draftNextSteps(ctx)
	mailboxes, err := s.db.ListActiveOutboundMailboxes(ctx)
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
	stuck, err := s.db.ListStuckOutboundMessages(ctx, sql.NullString{String: ts(s.now().Add(-stuckAfter)), Valid: true})
	if err != nil {
		return
	}
	for _, m := range stuck {
		_ = s.db.MarkOutboundMessageFailed(ctx, db.MarkOutboundMessageFailedParams{Error: "interrupted mid-send; not retried, to never send twice", ID: m.ID})
		if m.EnrollmentID.Valid {
			_ = s.db.SetOutboundEnrollmentStatus(ctx, db.SetOutboundEnrollmentStatusParams{Status: "stopped", ID: m.EnrollmentID.String})
		}
	}
}

// startOfDay is midnight today in the brand's time zone, as stored text.
func startOfDay(now time.Time, loc *time.Location) string {
	l := now.In(loc)
	return ts(time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc))
}

// sendOne sends the mailbox's next due email, if every limit allows it.
func (s *Service) sendOne(ctx context.Context, mb db.OutboundMailbox) error {
	now := s.now()
	if mb.NextSendAt.Valid && mb.NextSendAt.String > ts(now) {
		return nil
	}
	org, err := s.db.GetOrganizationByID(ctx, mb.OrgID)
	if err != nil || !s.Enabled(org) {
		return nil
	}
	st, err := s.settings(ctx, mb.OrgID)
	if err != nil || st.Paused == 1 {
		return err
	}
	loc := loadLocation(st.Timezone, "UTC")
	sentToday, err := s.db.CountOutboundSentSince(ctx, db.CountOutboundSentSinceParams{MailboxID: mb.ID, SentAt: sql.NullString{String: startOfDay(now, loc), Valid: true}})
	if err != nil || sentToday >= allowance(mb, now) {
		return err
	}
	m, err := s.db.NextOutboundMessageForMailbox(ctx, db.NextOutboundMessageForMailboxParams{MailboxID: mb.ID, NotBefore: ts(now)})
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if err := s.inWindow(ctx, m, st, now); err == errOutsideWindow {
		return nil
	} else if err != nil {
		return err
	}
	if s.suppressed(ctx, m.OrgID, m.ToEmail) {
		_, _ = s.db.SkipOutboundMessage(ctx, db.SkipOutboundMessageParams{ID: m.ID, OrgID: m.OrgID})
		if m.EnrollmentID.Valid {
			_ = s.db.SetOutboundEnrollmentStatus(ctx, db.SetOutboundEnrollmentStatusParams{Status: "unsubscribed", ID: m.EnrollmentID.String})
		}
		return nil
	}
	if n, err := s.db.ClaimOutboundMessage(ctx, db.ClaimOutboundMessageParams{ClaimedAt: sql.NullString{String: ts(now), Valid: true}, ID: m.ID}); err != nil || n == 0 {
		return err
	}
	smtpPass, _, err := s.passwords(mb)
	if err != nil {
		_ = s.db.MarkOutboundMessageFailed(ctx, db.MarkOutboundMessageFailedParams{Error: err.Error(), ID: m.ID})
		return err
	}
	var refs []string
	if m.Refs != "" {
		refs = strings.Fields(m.Refs)
	}
	body := m.Body
	if host := s.trackingHostFor(ctx, m, mb); host != "" {
		if body, err = s.trackLinks(ctx, m, host); err != nil {
			_ = s.db.ReleaseOutboundMessage(ctx, db.ReleaseOutboundMessageParams{Error: err.Error(), NotBefore: ts(now.Add(15 * time.Minute)), ID: m.ID})
			return err
		}
	}
	transient, sendErr := sendSMTP(mb, smtpPass, outMessage{
		FromName: mb.FromName, From: mb.Email, To: m.ToEmail, Subject: m.Subject, Body: body,
		MessageID: m.MessageID, InReplyTo: m.InReplyTo, References: refs, Date: now,
	})
	_ = s.db.SetOutboundMailboxNextSend(ctx, db.SetOutboundMailboxNextSendParams{NextSendAt: sql.NullString{String: ts(now.Add(s.gap())), Valid: true}, ID: mb.ID})
	if sendErr != nil {
		_ = s.db.SetOutboundMailboxError(ctx, db.SetOutboundMailboxErrorParams{LastError: sendErr.Error(), ID: mb.ID})
		if transient {
			return s.db.ReleaseOutboundMessage(ctx, db.ReleaseOutboundMessageParams{Error: sendErr.Error(), NotBefore: ts(now.Add(15 * time.Minute)), ID: m.ID})
		}
		_ = s.db.MarkOutboundMessageFailed(ctx, db.MarkOutboundMessageFailedParams{Error: sendErr.Error(), ID: m.ID})
		if m.EnrollmentID.Valid {
			_ = s.db.SetOutboundEnrollmentStatus(ctx, db.SetOutboundEnrollmentStatusParams{Status: "stopped", ID: m.EnrollmentID.String})
		}
		return sendErr
	}
	if err := s.db.MarkOutboundMessageSent(ctx, db.MarkOutboundMessageSentParams{SentAt: sql.NullString{String: ts(now), Valid: true}, ID: m.ID}); err != nil {
		return err
	}
	_ = s.db.SetOutboundMailboxError(ctx, db.SetOutboundMailboxErrorParams{LastError: "", ID: mb.ID})
	if m.EnrollmentID.Valid && m.Position > 0 {
		return s.scheduleNext(ctx, m.EnrollmentID.String, now)
	}
	return nil
}

// inWindow moves an email whose window has closed to the next opening.
func (s *Service) inWindow(ctx context.Context, m db.OutboundMessage, st db.OutboundSetting, now time.Time) error {
	start, end := 8, 17
	if m.EnrollmentID.Valid {
		if e, err := s.db.GetOutboundEnrollment(ctx, m.EnrollmentID.String); err == nil {
			if c, err := s.db.GetOutboundCampaign(ctx, db.GetOutboundCampaignParams{ID: e.CampaignID, OrgID: e.OrgID}); err == nil {
				start, end = int(c.WindowStartHour), int(c.WindowEndHour)
			}
		}
	} else {
		return nil // a reply in a conversation goes when it's written
	}
	tz := ""
	if p, err := s.db.GetOutboundProspect(ctx, db.GetOutboundProspectParams{ID: m.ProspectID, OrgID: m.OrgID}); err == nil {
		tz = p.Timezone
	}
	next := nextWindow(now, loadLocation(tz, st.Timezone), start, end)
	if next.After(now) {
		if err := s.db.SetOutboundMessageNotBefore(ctx, db.SetOutboundMessageNotBeforeParams{NotBefore: ts(next), ID: m.ID}); err != nil {
			return err
		}
		return errOutsideWindow
	}
	return nil
}

var errOutsideWindow = fmt.Errorf("outside the sending window; moved to the next opening")

// scheduleNext sets when the enrollment's next step is drafted, or closes it
// when the sequence is over.
func (s *Service) scheduleNext(ctx context.Context, enrollmentID string, sentAt time.Time) error {
	e, err := s.db.GetOutboundEnrollment(ctx, enrollmentID)
	if err != nil || e.Status != "active" {
		return err
	}
	steps, err := s.db.ListOutboundSteps(ctx, e.CampaignID)
	if err != nil {
		return err
	}
	for _, st := range steps {
		if st.Position == e.NextPosition {
			at := addBusinessDays(sentAt, int(st.DelayDays))
			return s.db.SetOutboundEnrollmentNext(ctx, db.SetOutboundEnrollmentNextParams{NextPosition: e.NextPosition, NextAt: sql.NullString{String: ts(at), Valid: true}, ID: e.ID})
		}
	}
	return s.db.SetOutboundEnrollmentStatus(ctx, db.SetOutboundEnrollmentStatusParams{Status: "done", ID: e.ID})
}

// checkHealth pauses a mailbox whose bounces or complaints cross the limits.
func (s *Service) checkHealth(ctx context.Context, mb db.OutboundMailbox) {
	since := ts(s.now().AddDate(0, 0, -7))
	sent, err := s.db.CountOutboundSentSince(ctx, db.CountOutboundSentSinceParams{MailboxID: mb.ID, SentAt: sql.NullString{String: since, Valid: true}})
	if err != nil {
		return
	}
	bounces, _ := s.db.CountOutboundBouncesSince(ctx, db.CountOutboundBouncesSinceParams{MailboxID: mb.ID, ReceivedAt: since})
	complaints, _ := s.db.CountOutboundComplaintsSince(ctx, db.CountOutboundComplaintsSinceParams{MailboxID: mb.ID, ReceivedAt: since})
	reason := ""
	switch {
	case bounces >= minBouncesToPause && sent > 0 && float64(bounces)/float64(sent) > bounceRateLimit:
		reason = fmt.Sprintf("bounce rate %.1f%% in 7 days (%d of %d); limit %.0f%%", 100*float64(bounces)/float64(sent), bounces, sent, 100*bounceRateLimit)
	case complaints >= complaintLimit:
		reason = fmt.Sprintf("%d complaints in 7 days; limit %d", complaints, complaintLimit)
	}
	if reason != "" {
		_ = s.db.SetOutboundMailboxStatus(ctx, db.SetOutboundMailboxStatusParams{Status: "paused", PausedReason: "Health: " + reason, ID: mb.ID})
	}
}
