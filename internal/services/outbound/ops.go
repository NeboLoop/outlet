package outbound

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/db"

	"github.com/google/uuid"
)

// MailboxHealth is one mailbox's state for the overview.
type MailboxHealth struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	Status       string `json:"status"`
	Reason       string `json:"reason,omitempty"`
	SentToday    int64  `json:"sent_today"`
	AllowToday   int64  `json:"allowed_today"`
	Bounces7d    int64  `json:"bounces_7d"`
	Complaints7d int64  `json:"complaints_7d"`
	LastError    string `json:"last_error,omitempty"`
}

// Overview is outreach at a glance, with the next things to do.
type Overview struct {
	Enabled        bool              `json:"enabled"`
	Paused         string            `json:"paused,omitempty"`
	MailingAddress string            `json:"mailing_address"`
	Domains        []map[string]any  `json:"domains"`
	Mailboxes      []MailboxHealth   `json:"mailboxes"`
	Prospects      map[string]int64  `json:"prospects"`
	Messages       map[string]int64  `json:"messages"`
	RepliesWaiting int64             `json:"replies_waiting"`
	Campaigns      []CampaignResults `json:"campaigns"`
	Next           []string          `json:"next"`
}

func (s *Service) Overview(ctx context.Context, org db.Organization) (Overview, error) {
	o := Overview{Enabled: s.Enabled(org), Prospects: map[string]int64{}, Messages: map[string]int64{}}
	st, err := s.settings(ctx, org.ID)
	if err != nil {
		return o, err
	}
	o.MailingAddress = st.MailingAddress
	if st.Paused == 1 {
		o.Paused = "all outreach paused: " + st.PausedReason
	}
	domains, _ := s.db.ListOutboundDomains(ctx, org.ID)
	for _, d := range domains {
		o.Domains = append(o.Domains, map[string]any{"id": d.ID, "domain": d.Domain, "status": d.Status, "check": d.CheckResult})
	}
	now := s.now()
	loc := loadLocation(st.Timezone, "UTC")
	since7 := ts(now.AddDate(0, 0, -7))
	mbs, _ := s.db.ListOutboundMailboxes(ctx, org.ID)
	for _, mb := range mbs {
		sent, _ := s.db.CountOutboundSentSince(ctx, db.CountOutboundSentSinceParams{MailboxID: mb.ID, SentAt: sql.NullString{String: startOfDay(now, loc), Valid: true}})
		b, _ := s.db.CountOutboundBouncesSince(ctx, db.CountOutboundBouncesSinceParams{MailboxID: mb.ID, ReceivedAt: since7})
		c, _ := s.db.CountOutboundComplaintsSince(ctx, db.CountOutboundComplaintsSinceParams{MailboxID: mb.ID, ReceivedAt: since7})
		o.Mailboxes = append(o.Mailboxes, MailboxHealth{ID: mb.ID, Email: mb.Email, Status: mb.Status, Reason: mb.PausedReason,
			SentToday: sent, AllowToday: allowance(mb, now), Bounces7d: b, Complaints7d: c, LastError: mb.LastError})
	}
	if rows, err := s.db.CountOutboundProspectsByStatus(ctx, org.ID); err == nil {
		for _, r := range rows {
			o.Prospects[r.Status] = r.N
		}
	}
	if rows, err := s.db.CountOutboundMessagesByStatus(ctx, org.ID); err == nil {
		for _, r := range rows {
			o.Messages[r.Status] = r.N
		}
	}
	o.RepliesWaiting, _ = s.db.CountUnhandledOutboundReplies(ctx, org.ID)
	camps, _ := s.db.ListOutboundCampaigns(ctx, org.ID)
	for _, c := range camps {
		if r, err := s.CampaignResults(ctx, c, false); err == nil {
			o.Campaigns = append(o.Campaigns, r)
		}
	}

	switch {
	case !o.Enabled:
		o.Next = append(o.Next, "Outbound is off for this brand. An admin adds its slug to OUTBOUND_BRANDS in the server config.")
	case st.MailingAddress == "":
		o.Next = append(o.Next, "Set the mailing address: setup_sending(action: settings, mailing_address: ...).")
	case len(domains) == 0:
		o.Next = append(o.Next, "Add an outreach domain: setup_sending(action: add_domain, domain: ...).")
	case len(mbs) == 0:
		o.Next = append(o.Next, "Connect mailboxes: setup_sending(action: add_mailboxes, mailboxes: [...]).")
	}
	for _, mb := range o.Mailboxes {
		if mb.Status == "paused" {
			o.Next = append(o.Next, fmt.Sprintf("%s is paused (%s); explain(mailbox_id: %q) says how to fix it.", mb.Email, mb.Reason, mb.ID))
		}
	}
	if n := o.Messages["draft"]; n > 0 {
		o.Next = append(o.Next, fmt.Sprintf("%d drafts wait for approval: review_drafts(action: list).", n))
	}
	if o.RepliesWaiting > 0 {
		o.Next = append(o.Next, fmt.Sprintf("%d replies need you: conversations(action: list).", o.RepliesWaiting))
	}
	if o.Prospects["new"] > 0 {
		o.Next = append(o.Next, fmt.Sprintf("%d verified prospects aren't in a campaign yet: draft_outreach.", o.Prospects["new"]))
	}
	return o, nil
}

// SettingsInput changes the brand's outreach settings. None of them is a limit.
type SettingsInput struct {
	MailingAddress string `json:"mailing_address,omitempty"`
	OptOutLine     string `json:"opt_out_line,omitempty"`
	Timezone       string `json:"timezone,omitempty"`
}

func (s *Service) UpdateSettings(ctx context.Context, org db.Organization, in SettingsInput) (db.OutboundSetting, error) {
	st, err := s.settings(ctx, org.ID)
	if err != nil {
		return st, err
	}
	if in.MailingAddress != "" {
		st.MailingAddress = strings.TrimSpace(in.MailingAddress)
	}
	if in.OptOutLine != "" {
		st.OptOutLine = strings.TrimSpace(in.OptOutLine)
	}
	if in.Timezone != "" {
		if _, err := time.LoadLocation(in.Timezone); err != nil {
			return st, refuse("unknown time zone %q; use an IANA name like America/Denver", in.Timezone)
		}
		st.Timezone = in.Timezone
	}
	return s.db.UpsertOutboundSettings(ctx, db.UpsertOutboundSettingsParams{
		OrgID: org.ID, MailingAddress: st.MailingAddress, OptOutLine: st.OptOutLine, Timezone: st.Timezone,
	})
}

// ListDrafts returns emails waiting for approval, oldest first.
func (s *Service) ListDrafts(ctx context.Context, org db.Organization, limit int) ([]Draft, error) {
	if limit <= 0 || limit > maxBatch {
		limit = 25
	}
	msgs, err := s.db.ListOutboundMessagesByStatus(ctx, db.ListOutboundMessagesByStatusParams{OrgID: org.ID, Status: "draft", Limit: int64(limit)})
	if err != nil {
		return nil, err
	}
	mbs := s.mailboxEmails(ctx, org.ID)
	out := make([]Draft, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, Draft{MessageID: m.ID, ProspectID: m.ProspectID, To: m.ToEmail, From: mbs[m.MailboxID], Subject: m.Subject, Body: m.Body, SendsAfter: m.NotBefore})
	}
	return out, nil
}

func (s *Service) mailboxEmails(ctx context.Context, orgID string) map[string]string {
	out := map[string]string{}
	if mbs, err := s.db.ListOutboundMailboxes(ctx, orgID); err == nil {
		for _, mb := range mbs {
			out[mb.ID] = mb.Email
		}
	}
	return out
}

// ReviewResult counts what a review call changed, and why the rest didn't.
type ReviewResult struct {
	Done    int               `json:"done"`
	Refused map[string]string `json:"refused,omitempty"`
}

// Review approves or skips drafts by ID.
func (s *Service) Review(ctx context.Context, org db.Organization, action string, ids []string) (ReviewResult, error) {
	if len(ids) == 0 || len(ids) > maxBatch {
		return ReviewResult{}, refuse("pass 1 to %d ids", maxBatch)
	}
	if action == "approve" {
		if err := s.requireEnabled(org); err != nil {
			return ReviewResult{}, err
		}
	}
	res := ReviewResult{Refused: map[string]string{}}
	for _, id := range ids {
		var n int64
		var err error
		if action == "approve" {
			n, err = s.db.ApproveOutboundMessage(ctx, db.ApproveOutboundMessageParams{ID: id, OrgID: org.ID})
		} else {
			n, err = s.db.SkipOutboundMessage(ctx, db.SkipOutboundMessageParams{ID: id, OrgID: org.ID})
		}
		switch {
		case err != nil:
			res.Refused[id] = err.Error()
		case n == 0:
			res.Refused[id] = "not a draft of this brand (already approved, sent or skipped?)"
		default:
			res.Done++
		}
	}
	if len(res.Refused) == 0 {
		res.Refused = nil
	}
	return res, nil
}

// EditDraft rewrites one waiting email.
func (s *Service) EditDraft(ctx context.Context, org db.Organization, id, subject, body string) error {
	m, err := s.db.GetOutboundMessage(ctx, db.GetOutboundMessageParams{ID: id, OrgID: org.ID})
	if err != nil {
		return refuse("no such draft")
	}
	if subject == "" {
		subject = m.Subject
	}
	if body == "" {
		body = m.Body
	}
	n, err := s.db.EditOutboundMessage(ctx, db.EditOutboundMessageParams{Subject: subject, Body: body, ID: id, OrgID: org.ID})
	if err == nil && n == 0 {
		return refuse("that email was already sent or skipped")
	}
	return err
}

// ListReplies returns replies, unhandled ones by default.
func (s *Service) ListReplies(ctx context.Context, org db.Organization, handled bool, limit int) ([]db.OutboundReply, error) {
	if limit <= 0 || limit > maxBatch {
		limit = 25
	}
	h := int64(0)
	if handled {
		h = 1
	}
	return s.db.ListOutboundReplies(ctx, db.ListOutboundRepliesParams{OrgID: org.ID, Handled: h, Limit: int64(limit)})
}

var classes = map[string]bool{
	"interested": true, "not_interested": true, "question": true, "out_of_office": true,
	"wrong_person": true, "unsubscribe": true, "complaint": true,
}

// Classify records what a reply means and applies it.
func (s *Service) Classify(ctx context.Context, org db.Organization, replyID, class string) error {
	if !classes[class] {
		return refuse("classification must be one of interested, not_interested, question, out_of_office, wrong_person, unsubscribe, complaint")
	}
	r, err := s.db.GetOutboundReply(ctx, db.GetOutboundReplyParams{ID: replyID, OrgID: org.ID})
	if err != nil {
		return refuse("no such reply")
	}
	handled := int64(1)
	if class == "interested" || class == "question" {
		handled = 0 // still needs an answer
	}
	if err := s.db.SetOutboundReplyClassification(ctx, db.SetOutboundReplyClassificationParams{Classification: class, Handled: handled, ID: r.ID, OrgID: org.ID}); err != nil {
		return err
	}
	if r.ProspectID.Valid {
		if p, err := s.db.GetOutboundProspect(ctx, db.GetOutboundProspectParams{ID: r.ProspectID.String, OrgID: org.ID}); err == nil {
			s.applyClassification(ctx, org.ID, p, class)
		}
	}
	return nil
}

// Answer replies to a reply: same mailbox, same thread. It goes out on the
// next engine pass, inside the mailbox's daily cap and gap.
func (s *Service) Answer(ctx context.Context, org db.Organization, replyID, body string) (string, error) {
	if err := s.requireEnabled(org); err != nil {
		return "", err
	}
	if strings.TrimSpace(body) == "" {
		return "", refuse("body is required")
	}
	r, err := s.db.GetOutboundReply(ctx, db.GetOutboundReplyParams{ID: replyID, OrgID: org.ID})
	if err != nil || !r.ProspectID.Valid {
		return "", refuse("no such reply")
	}
	if s.suppressed(ctx, org.ID, r.FromEmail) {
		return "", refuse("%s opted out or bounced; outreach never writes to them again", r.FromEmail)
	}
	mb, err := s.db.GetOutboundMailbox(ctx, db.GetOutboundMailboxParams{ID: r.MailboxID, OrgID: org.ID})
	if err != nil {
		return "", err
	}
	refs := strings.Fields(r.Refs)
	refs = append(refs, r.MessageID)
	subject := r.Subject
	if !strings.HasPrefix(strings.ToLower(subject), "re:") {
		subject = "Re: " + subject
	}
	m, err := s.db.CreateOutboundMessage(ctx, db.CreateOutboundMessageParams{
		ID: uuid.NewString(), OrgID: org.ID, MailboxID: mb.ID, ProspectID: r.ProspectID.String, Position: 0,
		ToEmail: r.FromEmail, Subject: subject, Body: signed(body, mb), Status: "approved",
		MessageID: newMessageID(mb.Email), InReplyTo: r.MessageID, Refs: strings.Join(refs, " "), NotBefore: ts(s.now()),
	})
	if err != nil {
		return "", err
	}
	_ = s.db.SetOutboundReplyHandled(ctx, db.SetOutboundReplyHandledParams{ID: r.ID, OrgID: org.ID})
	return m.ID, nil
}

// MarkPartner records that a creator said yes.
func (s *Service) MarkPartner(ctx context.Context, org db.Organization, prospectID string) error {
	p, err := s.db.GetOutboundProspect(ctx, db.GetOutboundProspectParams{ID: prospectID, OrgID: org.ID})
	if err != nil {
		return refuse("no such prospect")
	}
	return s.db.SetOutboundProspectStatus(ctx, db.SetOutboundProspectStatusParams{Status: "partner", ID: p.ID})
}

// Pause stops or resumes everything, one mailbox or one campaign. Resuming a
// mailbox paused for health or DNS re-pauses it on the next check if the
// problem is still there.
func (s *Service) Pause(ctx context.Context, org db.Organization, scope, id string, resume bool) error {
	switch scope {
	case "all":
		reason := "paused through MCP"
		paused := int64(1)
		if resume {
			if err := s.requireEnabled(org); err != nil {
				return err
			}
			paused, reason = 0, ""
		}
		return s.db.SetOutboundPaused(ctx, db.SetOutboundPausedParams{OrgID: org.ID, Paused: paused, PausedReason: reason})
	case "mailbox":
		mb, err := s.db.GetOutboundMailbox(ctx, db.GetOutboundMailboxParams{ID: id, OrgID: org.ID})
		if err != nil {
			return refuse("no such mailbox")
		}
		if !resume {
			return s.db.SetOutboundMailboxStatus(ctx, db.SetOutboundMailboxStatusParams{Status: "paused", PausedReason: "paused through MCP", ID: mb.ID})
		}
		if strings.HasPrefix(mb.PausedReason, "DNS:") {
			d, err := s.db.GetOutboundDomain(ctx, db.GetOutboundDomainParams{ID: mb.DomainID, OrgID: org.ID})
			if err != nil {
				return err
			}
			if c, err := s.CheckDomain(ctx, d); err == nil && !c.OK() {
				return refuse("%s still lacks %s; fix the DNS records, then resume", d.Domain, strings.Join(c.Missing, ", "))
			}
		}
		if err := s.db.SetOutboundMailboxStatus(ctx, db.SetOutboundMailboxStatusParams{Status: "active", PausedReason: "", ID: mb.ID}); err != nil {
			return err
		}
		mb.Status = "active"
		s.checkHealth(ctx, mb)
		if again, err := s.db.GetOutboundMailbox(ctx, db.GetOutboundMailboxParams{ID: id, OrgID: org.ID}); err == nil && again.Status == "paused" {
			return refuse("%s is still over a health limit (%s); it can resume once the 7-day window clears", again.Email, again.PausedReason)
		}
		return nil
	case "campaign":
		status := "paused"
		if resume {
			status = "active"
		}
		if _, err := s.db.GetOutboundCampaign(ctx, db.GetOutboundCampaignParams{ID: id, OrgID: org.ID}); err != nil {
			return refuse("no such campaign")
		}
		return s.db.SetOutboundCampaignStatus(ctx, db.SetOutboundCampaignStatusParams{Status: status, ID: id, OrgID: org.ID})
	}
	return refuse("scope must be all, mailbox or campaign")
}

// Explain answers "why didn't this send?" and "why is this mailbox paused?"
// in plain words.
func (s *Service) Explain(ctx context.Context, org db.Organization, messageID, mailboxID, prospectID string) (string, error) {
	now := s.now()
	switch {
	case messageID != "":
		m, err := s.db.GetOutboundMessage(ctx, db.GetOutboundMessageParams{ID: messageID, OrgID: org.ID})
		if err != nil {
			return "", refuse("no such email")
		}
		switch m.Status {
		case "draft":
			return "It is a draft waiting for approval: review_drafts(action: approve, ids: [\"" + m.ID + "\"]).", nil
		case "sent":
			return "It was sent at " + m.SentAt.String + " UTC.", nil
		case "skipped":
			return "It was skipped: by a person, or because the prospect replied, opted out or bounced first.", nil
		case "failed":
			return "Sending failed: " + m.Error + ". It is not retried.", nil
		case "sending":
			return "It is being sent right now.", nil
		}
		mb, err := s.db.GetOutboundMailbox(ctx, db.GetOutboundMailboxParams{ID: m.MailboxID, OrgID: org.ID})
		if err != nil {
			return "", err
		}
		var why []string
		if !s.Enabled(org) {
			why = append(why, "outbound is off for this brand in the server config")
		}
		if st, err := s.settings(ctx, org.ID); err == nil && st.Paused == 1 {
			why = append(why, "all outreach is paused ("+st.PausedReason+")")
		}
		if mb.Status != "active" {
			why = append(why, mb.Email+" is paused: "+mb.PausedReason)
		}
		if m.NotBefore > ts(now) {
			why = append(why, "it waits for the sending window, which opens "+m.NotBefore+" UTC")
		}
		if mb.NextSendAt.Valid && mb.NextSendAt.String > ts(now) {
			why = append(why, "the mailbox's next send is spaced to "+mb.NextSendAt.String+" UTC")
		}
		if st, err := s.settings(ctx, org.ID); err == nil {
			sent, _ := s.db.CountOutboundSentSince(ctx, db.CountOutboundSentSinceParams{MailboxID: mb.ID, SentAt: sql.NullString{String: startOfDay(now, loadLocation(st.Timezone, "UTC")), Valid: true}})
			if a := allowance(mb, now); sent >= a {
				why = append(why, fmt.Sprintf("the mailbox reached today's limit (%d of %d; warm-up allows %d)", sent, a, a))
			}
		}
		if mb.LastError != "" {
			why = append(why, "the mailbox's last error: "+mb.LastError)
		}
		if len(why) == 0 {
			return "It is approved and goes out on one of the next passes (every 30 seconds), behind older emails from the same mailbox.", nil
		}
		return "Approved, but waiting: " + strings.Join(why, "; ") + ".", nil
	case mailboxID != "":
		mb, err := s.db.GetOutboundMailbox(ctx, db.GetOutboundMailboxParams{ID: mailboxID, OrgID: org.ID})
		if err != nil {
			return "", refuse("no such mailbox")
		}
		if mb.Status == "active" {
			return fmt.Sprintf("%s is active; warm-up allows %d today (at most %d).", mb.Email, allowance(mb, now), MaxDailyCap), nil
		}
		r := mb.PausedReason
		switch {
		case strings.HasPrefix(r, "DNS:"):
			return mb.Email + " is paused because " + strings.TrimPrefix(r, "DNS: ") + ". Fix those records, then pause(scope: mailbox, id, resume: true).", nil
		case strings.HasPrefix(r, "Health:"):
			return mb.Email + " is paused to protect its reputation: " + strings.TrimPrefix(r, "Health: ") + ". Check the bounced and complaining addresses, verify the rest of the list, and resume once the 7-day window clears.", nil
		}
		return mb.Email + " is paused: " + r + ". Resume with pause(scope: mailbox, id, resume: true).", nil
	case prospectID != "":
		p, err := s.db.GetOutboundProspect(ctx, db.GetOutboundProspectParams{ID: prospectID, OrgID: org.ID})
		if err != nil {
			return "", refuse("no such prospect")
		}
		if reason := sendable(p); reason != "" {
			return p.Email + " can't be enrolled: " + reason + ".", nil
		}
		if s.suppressed(ctx, org.ID, p.Email) {
			return p.Email + " is on the suppression list; nothing is ever sent to it.", nil
		}
		return p.Email + " is verified and ready for draft_outreach.", nil
	}
	return "", refuse("pass message_id, mailbox_id or prospect_id")
}
