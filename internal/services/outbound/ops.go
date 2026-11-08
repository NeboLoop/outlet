package outbound

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/email"

	"github.com/google/uuid"
)

// MailboxHealth is one mailbox's state, for the overview and the screens.
type MailboxHealth struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	FromName     string `json:"from_name"`
	Status       string `json:"status"`
	Reason       string `json:"reason,omitempty"`
	SentToday    int64  `json:"sent_today"`
	AllowToday   int64  `json:"allowed_today"`
	Sent7d       int64  `json:"sent_7d"`
	Bounces7d    int64  `json:"bounces_7d"`
	SpamBlocks7d int64  `json:"spam_blocks_7d"`
	Complaints7d int64  `json:"complaints_7d"`
	LastError    string `json:"last_error,omitempty"`
	PoolID       string `json:"pool_id,omitempty"`       // its mailbox pool (sending pathway)
	ConnectionID string `json:"connection_id,omitempty"` // the pool's connection its domain belongs to
	DailyLimit   *int64 `json:"daily_limit,omitempty"`   // its own daily limit; empty uses its connection's, else the pool's
	Limited      bool   `json:"limited"`                 // false: no daily limit applies, whatever allowed_today says
	// LimitInUse is the daily limit it sends under: its own, else its
	// connection's default, else its pool's. 0 is no limit.
	LimitInUse int64 `json:"limit_in_use"`
	// WarmingUp says its warm-up (its connection's, else the pool's) still
	// holds it below that limit:
	// allowed_today is today's step of the ramp, which climbs to LimitInUse
	// (with no limit, it keeps climbing).
	WarmingUp bool   `json:"warming_up"`
	Domain    string `json:"domain"`
}

// DomainInfo is one outreach domain, its last DNS check, and the records it
// needs with which of them that check found.
type DomainInfo struct {
	ID           string         `json:"id"`
	Domain       string         `json:"domain"`
	Status       string         `json:"status"`
	TrackingHost string         `json:"tracking_host,omitempty"`
	Check        DNSCheck       `json:"check"`
	CheckedAt    string         `json:"checked_at,omitempty"`
	Records      []DomainRecord `json:"records"`
}

// DomainInfoOf reads a domain row, its stored check and its records.
func DomainInfoOf(d db.DomainIdentity) DomainInfo {
	var c DNSCheck
	_ = json.Unmarshal([]byte(d.DnsCheck), &c)
	return DomainInfo{ID: d.ID, Domain: d.Domain, Status: d.VerificationStatus.String, TrackingHost: d.TrackingHost, Check: c, CheckedAt: d.LastCheckedAt.String,
		Records: domainRecords(d, c)}
}

// Overview is a brand's outreach at a glance, with the next things to do.
type Overview struct {
	Enabled        bool              `json:"enabled"`
	Paused         string            `json:"paused,omitempty"`
	MailingAddress string            `json:"mailing_address"`
	OptOutLine     string            `json:"opt_out_line"`
	Timezone       string            `json:"timezone"`
	DNSTokenSet    bool              `json:"dns_token_set"`
	Domains        []DomainInfo      `json:"domains"`
	Mailboxes      []MailboxHealth   `json:"mailboxes"`
	Prospects      map[string]int64  `json:"prospects"`
	Messages       map[string]int64  `json:"messages"`
	RepliesWaiting int64             `json:"replies_waiting"`
	DraftsWaiting  int64             `json:"drafts_waiting"`
	// ReplyRate is people who replied over people reached, across every
	// campaign, in percent.
	ReplyRate float64           `json:"reply_rate"`
	Campaigns []CampaignResults `json:"campaigns"`
	Next           []string          `json:"next"`
}

func (s *Service) mailboxHealth(ctx context.Context, mb db.Mailbox, org db.Organization) MailboxHealth {
	out := MailboxHealth{
		ID: mb.ID, Email: mb.Email, FromName: mb.FromName, Status: mb.Status, Reason: mb.PausedReason,
		LastError: mb.LastError, Domain: domainOf(mb.Email),
	}
	if mb.DailyLimit.Valid {
		out.DailyLimit = &mb.DailyLimit.Int64
	}
	h, p, err := s.health(ctx, mb)
	if err != nil {
		return out
	}
	now := s.now()
	out.PoolID = p.ID
	if d, err := s.db.GetDomainIdentity(ctx, mb.DomainID); err == nil {
		out.ConnectionID = d.ConnectionID.String
	}
	limits, err := s.mailer.MailboxLimits(ctx, p, mb)
	if err != nil {
		return out
	}
	out.SentToday, _ = s.mailer.SentSince(ctx, org, p.ID, mb.ID, now)
	start := parseTS(mb.WarmupStartedAt)
	out.AllowToday, out.Limited = email.Allowance(limits, start, now)
	out.LimitInUse = limits.DailyLimit
	out.WarmingUp = email.WarmingUp(limits, start, now)
	out.Sent7d, out.Bounces7d, out.SpamBlocks7d, out.Complaints7d = h.Sent, h.Bounces, h.SpamBlocks, h.Complaints
	return out
}

// Overview reads a brand's outreach.
func (s *Service) Overview(ctx context.Context, org db.Organization) (Overview, error) {
	o := Overview{
		Enabled: s.Enabled(org), Paused: org.OutboundPausedReason, MailingAddress: org.MailingAddress,
		OptOutLine: org.OutboundOptOutLine, Timezone: org.OutboundTimezone, DNSTokenSet: len(org.DnsTokenEnc) > 0,
		Prospects: map[string]int64{}, Messages: map[string]int64{},
		// Empty lists are [] in JSON, never null: the app reads their length.
		Domains: []DomainInfo{}, Mailboxes: []MailboxHealth{}, Campaigns: []CampaignResults{}, Next: []string{},
	}
	domains, _ := s.db.ListOutreachDomains(ctx, org.ID)
	for _, d := range domains {
		o.Domains = append(o.Domains, DomainInfoOf(d))
	}
	mbs, _ := s.db.ListMailboxes(ctx, org.ID)
	for _, mb := range mbs {
		o.Mailboxes = append(o.Mailboxes, s.mailboxHealth(ctx, mb, org))
	}
	if rows, err := s.db.CountProspectsByStatus(ctx, org.ID); err == nil {
		for _, r := range rows {
			o.Prospects[r.Status.String] = r.N
		}
	}
	if rows, err := s.db.CountOutreachEmailsByStatus(ctx, sql.NullString{String: org.ID, Valid: true}); err == nil {
		for _, r := range rows {
			o.Messages[r.Status.String] = r.N
		}
	}
	o.RepliesWaiting, _ = s.db.CountUnhandledEmailReplies(ctx, org.ID)
	o.DraftsWaiting = o.Messages["draft"]
	seqs, _ := s.db.ListOutreachSequences(ctx, sql.NullString{String: org.ID, Valid: true})
	var reached, replied int64
	for _, c := range seqs {
		if r, err := s.CampaignResults(ctx, c, false); err == nil {
			o.Campaigns = append(o.Campaigns, r)
			reached, replied = reached+r.Reached, replied+r.Replied
		}
	}
	o.ReplyRate = pct(replied, reached)

	switch {
	case !o.Enabled:
		o.Next = append(o.Next, "Outbound is off for this brand. Its admin turns it on in the Outlet app, in Outbound, Setup.")
	case org.MailingAddress == "":
		o.Next = append(o.Next, "Set the brand's mailing address in Outbound, Setup.")
	case len(domains) == 0:
		o.Next = append(o.Next, "Add an outreach domain in Outbound, Setup.")
	case len(mbs) == 0:
		o.Next = append(o.Next, "Connect mailboxes in Outbound, Setup.")
	}
	if o.Paused != "" {
		o.Next = append(o.Next, "All outreach is paused ("+o.Paused+"). Resume it to send again.")
	}
	for _, mb := range o.Mailboxes {
		if mb.Status == "paused" {
			o.Next = append(o.Next, fmt.Sprintf("%s is paused (%s). Explaining the mailbox says how to fix it.", mb.Email, mb.Reason))
		}
	}
	if o.DraftsWaiting > 0 {
		o.Next = append(o.Next, fmt.Sprintf("%d drafts wait for approval.", o.DraftsWaiting))
	}
	if o.RepliesWaiting > 0 {
		o.Next = append(o.Next, fmt.Sprintf("%d replies need you.", o.RepliesWaiting))
	}
	return o, nil
}

// perDay says a daily limit in words.
func perDay(limit int64) string {
	if limit <= 0 {
		return "no daily limit"
	}
	return fmt.Sprintf("%d a day", limit)
}

// SettingsInput changes the brand's outreach settings. None of them is a limit.
type SettingsInput struct {
	MailingAddress string `json:"mailing_address,omitempty"`
	OptOutLine     string `json:"opt_out_line,omitempty"`
	Timezone       string `json:"timezone,omitempty"`
}

func (s *Service) UpdateSettings(ctx context.Context, org db.Organization, in SettingsInput) (db.Organization, error) {
	if in.MailingAddress != "" {
		org.MailingAddress = strings.TrimSpace(in.MailingAddress)
	}
	if in.OptOutLine != "" {
		org.OutboundOptOutLine = strings.TrimSpace(in.OptOutLine)
	}
	if in.Timezone != "" {
		if _, err := time.LoadLocation(in.Timezone); err != nil {
			return org, refuse("Unknown time zone %q; use a name like America/Denver.", in.Timezone)
		}
		org.OutboundTimezone = in.Timezone
	}
	return org, s.db.UpdateOutboundSettings(ctx, db.UpdateOutboundSettingsParams{
		MailingAddress: org.MailingAddress, OutboundOptOutLine: org.OutboundOptOutLine, OutboundTimezone: org.OutboundTimezone, ID: org.ID,
	})
}

// ListDrafts returns outreach emails waiting for approval, oldest first.
func (s *Service) ListDrafts(ctx context.Context, org db.Organization, limit int) ([]Draft, error) {
	if limit <= 0 || limit > maxBatch {
		limit = 25
	}
	rows, err := s.db.ListOutreachEmails(ctx, db.ListOutreachEmailsParams{OrgID: sql.NullString{String: org.ID, Valid: true}, Status: sql.NullString{String: "draft", Valid: true}, LimitCount: int64(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]Draft, 0, len(rows))
	for _, r := range rows {
		out = append(out, Draft{MessageID: r.ID, CampaignID: r.CampaignID, CampaignName: r.CampaignName, Step: r.Position,
			Preview: Preview{ProspectID: r.ContactID.String, To: r.ToEmail, From: r.FromEmail, Subject: r.Subject.String, Body: r.Body.String, SendsAfter: r.ScheduledFor}})
	}
	return out, nil
}

// ReviewResult counts what a review call changed, and why the rest didn't.
type ReviewResult struct {
	Done    int               `json:"done"`
	Refused map[string]string `json:"refused,omitempty"`
}

// Review approves or skips drafts by ID.
func (s *Service) Review(ctx context.Context, org db.Organization, action string, ids []string) (ReviewResult, error) {
	if len(ids) == 0 || len(ids) > maxBatch {
		return ReviewResult{}, refuse("Pick 1 to %d drafts.", maxBatch)
	}
	if action == "approve" {
		if err := s.requireEnabled(org); err != nil {
			return ReviewResult{}, err
		}
	}
	res := ReviewResult{Refused: map[string]string{}}
	orgID := sql.NullString{String: org.ID, Valid: true}
	for _, id := range ids {
		var n int64
		var err error
		if action == "approve" {
			n, err = s.db.ApproveOutreachDraft(ctx, db.ApproveOutreachDraftParams{ID: id, OrgID: orgID})
		} else {
			n, err = s.db.SkipOutreachEmail(ctx, db.SkipOutreachEmailParams{ID: id, OrgID: orgID})
		}
		switch {
		case err != nil:
			res.Refused[id] = err.Error()
		case n == 0:
			res.Refused[id] = "not a waiting email of this brand (already approved, sent or skipped?)"
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
	m, err := s.db.GetOutreachEmail(ctx, db.GetOutreachEmailParams{ID: id, OrgID: sql.NullString{String: org.ID, Valid: true}})
	if err != nil {
		return refuse("no such email")
	}
	if subject == "" {
		subject = m.Subject.String
	}
	if body == "" {
		body = m.Body.String
	}
	n, err := s.db.EditOutreachEmail(ctx, db.EditOutreachEmailParams{Subject: sql.NullString{String: subject, Valid: true}, Body: sql.NullString{String: body, Valid: true}, ID: id, OrgID: sql.NullString{String: org.ID, Valid: true}})
	if err == nil && n == 0 {
		return refuse("that email was already sent or skipped")
	}
	return err
}

// ListReplies returns replies, unhandled ones by default, each with the name
// of the campaign it answers.
func (s *Service) ListReplies(ctx context.Context, org db.Organization, handled bool, limit int) ([]db.ListEmailRepliesRow, error) {
	if limit <= 0 || limit > maxBatch {
		limit = 25
	}
	return s.db.ListEmailReplies(ctx, db.ListEmailRepliesParams{OrgID: org.ID, Handled: boolInt(handled), Limit: int64(limit)})
}

var classes = map[string]bool{
	"interested": true, "not_interested": true, "question": true, "out_of_office": true,
	"wrong_person": true, "unsubscribe": true, "complaint": true,
}

// Classify records what a reply means and applies it.
func (s *Service) Classify(ctx context.Context, org db.Organization, replyID, class string) error {
	if !classes[class] {
		return refuse("A reply sorts as interested, not interested, question, out of office, wrong person, unsubscribe or complaint.")
	}
	r, err := s.db.GetEmailReply(ctx, db.GetEmailReplyParams{ID: replyID, OrgID: org.ID})
	if err != nil {
		return refuse("no such reply")
	}
	handled := int64(1)
	if class == "interested" || class == "question" {
		handled = 0 // still needs an answer
	}
	if err := s.db.SetEmailReplyClassification(ctx, db.SetEmailReplyClassificationParams{Classification: class, Handled: handled, ID: r.ID, OrgID: org.ID}); err != nil {
		return err
	}
	if r.ContactID.Valid {
		if c, err := s.db.GetContactByID(ctx, r.ContactID.String); err == nil {
			s.applyClassification(ctx, org.ID, c, class)
		}
	}
	return nil
}

// Answer replies to a reply: same mailbox, same thread. It goes out on the
// next pass, inside the mailbox's daily cap and gap.
func (s *Service) Answer(ctx context.Context, org db.Organization, replyID, body string) (string, error) {
	if err := s.requireEnabled(org); err != nil {
		return "", err
	}
	if strings.TrimSpace(body) == "" {
		return "", refuse("body is required")
	}
	r, err := s.db.GetEmailReply(ctx, db.GetEmailReplyParams{ID: replyID, OrgID: org.ID})
	if err != nil || !r.ContactID.Valid {
		return "", refuse("no such reply")
	}
	if s.suppressed(ctx, org.ID, r.FromEmail) {
		return "", refuse("%s opted out or bounced; outreach never writes to them again", r.FromEmail)
	}
	mb, err := s.db.GetMailbox(ctx, db.GetMailboxParams{ID: r.MailboxID, OrgID: org.ID})
	if err != nil {
		return "", err
	}
	refs := append(strings.Fields(r.Refs), r.MessageID)
	subject := r.Subject
	if !strings.HasPrefix(strings.ToLower(subject), "re:") {
		subject = "Re: " + subject
	}
	if sig := strings.TrimSpace(mb.Signature); sig != "" {
		body = strings.TrimRight(body, "\n ") + "\n\n" + sig
	}
	m, err := s.db.QueueOutreachEmail(ctx, db.QueueOutreachEmailParams{
		ID: uuid.NewString(), ContactID: r.ContactID, ScheduledFor: ts(s.now()),
		Status: sql.NullString{String: "pending", Valid: true}, TrackingToken: sql.NullString{String: uuid.NewString(), Valid: true},
		MailboxID: sql.NullString{String: mb.ID, Valid: true}, Subject: sql.NullString{String: subject, Valid: true}, Body: sql.NullString{String: body, Valid: true},
		MessageID: sql.NullString{String: uuid.NewString() + "@" + domainOf(mb.Email), Valid: true},
		InReplyTo: sql.NullString{String: r.MessageID, Valid: true}, Refs: sql.NullString{String: strings.Join(refs, " "), Valid: true},
	})
	if err != nil {
		return "", err
	}
	_ = s.db.SetEmailReplyHandled(ctx, db.SetEmailReplyHandledParams{ID: r.ID, OrgID: org.ID})
	return m.ID, nil
}

// MarkPartner records that a creator said yes (Outlet's partner tag).
func (s *Service) MarkPartner(ctx context.Context, org db.Organization, contactID string) error {
	c, err := s.db.GetContactByID(ctx, contactID)
	if err != nil || c.OrgID.String != org.ID {
		return refuse("no such prospect")
	}
	_, err = s.db.AddContactTag(ctx, db.AddContactTagParams{ContactID: sql.NullString{String: c.ID, Valid: true}, Tag: "partner"})
	return err
}

// Pause stops or resumes everything, one mailbox or one campaign. A mailbox
// paused for health or DNS resumes only once the problem is gone.
func (s *Service) Pause(ctx context.Context, org db.Organization, scope, id string, resume bool) error {
	switch scope {
	case "all":
		reason := "paused through MCP or the app"
		if resume {
			if err := s.requireEnabled(org); err != nil {
				return err
			}
			reason = ""
		}
		return s.db.SetOutboundPausedReason(ctx, db.SetOutboundPausedReasonParams{OutboundPausedReason: reason, ID: org.ID})
	case "mailbox":
		mb, err := s.db.GetMailbox(ctx, db.GetMailboxParams{ID: id, OrgID: org.ID})
		if err != nil {
			return refuse("no such mailbox")
		}
		if !resume {
			return s.db.SetMailboxStatus(ctx, db.SetMailboxStatusParams{Status: "paused", PausedReason: "paused by hand", ID: mb.ID})
		}
		if strings.HasPrefix(mb.PausedReason, deletedPrefix) {
			return refuse("%s was deleted at its mailbox service; it can't send again", mb.Email)
		}
		if strings.HasPrefix(mb.PausedReason, "DNS:") {
			d, err := s.db.GetOutreachDomain(ctx, db.GetOutreachDomainParams{ID: mb.DomainID, OrgID: org.ID})
			if err != nil {
				return err
			}
			if c, err := s.CheckDomain(ctx, d); err == nil && !c.OK() {
				return refuse("%s still lacks %s; fix the DNS records, then resume", d.Domain, strings.Join(c.Missing, ", "))
			}
		}
		if err := s.db.SetMailboxStatus(ctx, db.SetMailboxStatusParams{Status: "active", PausedReason: "", ID: mb.ID}); err != nil {
			return err
		}
		mb.Status = "active"
		s.checkHealth(ctx, mb)
		if again, err := s.db.GetMailbox(ctx, db.GetMailboxParams{ID: id, OrgID: org.ID}); err == nil && again.Status == "paused" {
			return refuse("%s is still over a health limit (%s); it can resume once the 7-day window clears", again.Email, again.PausedReason)
		}
		return nil
	case "campaign":
		if _, err := s.db.GetOutreachSequence(ctx, db.GetOutreachSequenceParams{ID: id, OrgID: sql.NullString{String: org.ID, Valid: true}}); err != nil {
			return refuse("no such campaign")
		}
		return s.db.SetSequenceActive(ctx, db.SetSequenceActiveParams{IsActive: sql.NullInt64{Int64: boolInt(resume), Valid: true}, ID: id, OrgID: sql.NullString{String: org.ID, Valid: true}})
	}
	return refuse("Pause all outreach, one mailbox or one campaign.")
}

// Explain answers "why didn't this send?" and "why is this mailbox paused?"
// in plain words.
func (s *Service) Explain(ctx context.Context, org db.Organization, messageID, mailboxID, prospectID string) (string, error) {
	now := s.now()
	switch {
	case messageID != "":
		m, err := s.db.GetOutreachEmail(ctx, db.GetOutreachEmailParams{ID: messageID, OrgID: sql.NullString{String: org.ID, Valid: true}})
		if err != nil {
			return "", refuse("no such email")
		}
		switch m.Status.String {
		case "draft":
			return "It is a draft waiting for approval.", nil
		case "sent":
			return "It was sent at " + m.SentAt.String + " UTC.", nil
		case "cancelled":
			return "It was skipped: by a person, or because the prospect replied, opted out or bounced first.", nil
		case "failed":
			return "Sending failed: " + m.ErrorMessage.String + ". It is not retried.", nil
		case "sending":
			return "It is being sent right now.", nil
		}
		mb, err := s.db.GetMailbox(ctx, db.GetMailboxParams{ID: m.MailboxID.String, OrgID: org.ID})
		if err != nil {
			return "", err
		}
		var why []string
		if !s.Enabled(org) {
			why = append(why, "outbound is off for this brand (its admin turns it on in the Outlet app)")
		}
		if org.OutboundPausedReason != "" {
			why = append(why, "all outreach is paused ("+org.OutboundPausedReason+")")
		}
		if mb.Status != "active" {
			why = append(why, mb.Email+" is paused: "+mb.PausedReason)
		}
		if m.ScheduledFor > ts(now) {
			why = append(why, "it waits for the sending window, which opens "+m.ScheduledFor+" UTC")
		}
		if mb.NextSendAt.Valid && mb.NextSendAt.String > ts(now) {
			why = append(why, "the mailbox's next send is spaced to "+mb.NextSendAt.String+" UTC")
		}
		h := s.mailboxHealth(ctx, mb, org)
		if h.Limited && h.SentToday >= h.AllowToday {
			why = append(why, fmt.Sprintf("the mailbox reached today's limit (%d of %d)", h.SentToday, h.AllowToday))
		}
		if mb.LastError != "" {
			why = append(why, "the mailbox's last error: "+mb.LastError)
		}
		if len(why) == 0 {
			return "It is approved and goes out on one of the next passes (every 30 seconds), behind older emails from the same mailbox.", nil
		}
		return "Approved, but waiting: " + strings.Join(why, "; ") + ".", nil
	case mailboxID != "":
		mb, err := s.db.GetMailbox(ctx, db.GetMailboxParams{ID: mailboxID, OrgID: org.ID})
		if err != nil {
			return "", refuse("no such mailbox")
		}
		if mb.Status == "active" {
			h := s.mailboxHealth(ctx, mb, org)
			if !h.Limited {
				return fmt.Sprintf("%s is active, with no daily limit; it sent %d today.", mb.Email, h.SentToday), nil
			}
			if h.WarmingUp {
				return fmt.Sprintf("%s is active and warming up: %d allowed today, climbing each day to %s; it sent %d.", mb.Email, h.AllowToday, perDay(h.LimitInUse), h.SentToday), nil
			}
			return fmt.Sprintf("%s is active; it may send %s and sent %d today.", mb.Email, perDay(h.LimitInUse), h.SentToday), nil
		}
		r := mb.PausedReason
		switch {
		case strings.HasPrefix(r, "DNS:"):
			return mb.Email + " is paused because " + strings.TrimPrefix(r, "DNS: ") + ". Fix those records, then resume the mailbox.", nil
		case strings.HasPrefix(r, "Health:"):
			return mb.Email + " is paused to protect its reputation: " + strings.TrimPrefix(r, "Health: ") + ". Check the bounced, blocked and complaining addresses, verify the rest of the list, and resume once the 7-day window clears.", nil
		}
		return mb.Email + " is paused: " + r + ". Resume the mailbox to send again.", nil
	case prospectID != "":
		c, err := s.db.GetContactByID(ctx, prospectID)
		if err != nil || c.OrgID.String != org.ID {
			return "", refuse("no such prospect")
		}
		if s.suppressed(ctx, org.ID, c.Email) {
			return c.Email + " is on the suppression list, or bounced or complained; nothing is ever sent to it.", nil
		}
		return c.Email + " can be enrolled in an outreach campaign if it's on an outreach list with status prospect.", nil
	}
	return "", refuse("Name an email, a mailbox or a prospect to explain.")
}
