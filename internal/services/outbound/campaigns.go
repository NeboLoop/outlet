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

// An outreach campaign is an Outlet sequence of type 'outreach' on an
// outreach list. Enrolling a prospect is the sequence engine's own start;
// the engine writes and queues each step under outreach's rules.

// StepInput is one email in a sequence. An empty subject on a later step
// replies in the same thread ("Re: " the first subject).
type StepInput struct {
	DelayDays int    `json:"delay_days,omitempty"` // days after the previous step; weekends are skipped by the sending window
	Subject   string `json:"subject,omitempty"`
	Body      string `json:"body"`
}

// CampaignInput creates a campaign and enrolls its prospects in one go.
type CampaignInput struct {
	Name            string      `json:"name"`
	List            string      `json:"list,omitempty"` // the outreach list (default Outreach)
	Steps           []StepInput `json:"steps"`
	ProspectIDs     []string    `json:"prospect_ids"`
	ApprovalMode    string      `json:"approval_mode,omitempty"` // every (default), first, automatic
	TrackOpens      bool        `json:"track_opens,omitempty"`
	TrackClicks     bool        `json:"track_clicks,omitempty"`
	WindowStartHour int         `json:"window_start_hour,omitempty"`
	WindowEndHour   int         `json:"window_end_hour,omitempty"`
	IdempotencyKey  string      `json:"idempotency_key,omitempty"`
	DryRun          bool        `json:"dry_run,omitempty"`
}

// Preview is one written outreach email, for a person to read.
type Preview struct {
	ProspectID string `json:"prospect_id"`
	To         string `json:"to"`
	From       string `json:"from"`
	Subject    string `json:"subject"`
	Body       string `json:"body"`
	SendsAfter string `json:"sends_after,omitempty"`
}

// Draft is a saved outreach email waiting for approval: its id (which
// approve, skip and edit take), and the campaign and step it belongs to.
type Draft struct {
	MessageID    string `json:"message_id"`
	CampaignID   string `json:"campaign_id"`
	CampaignName string `json:"campaign_name"`
	Step         int64  `json:"step"`
	Preview
}

// CampaignResult reports what draft_outreach did: the saved drafts, or with
// dry_run the previews of the first emails.
type CampaignResult struct {
	CampaignID string            `json:"campaign_id,omitempty"`
	Enrolled   int               `json:"enrolled"`
	Drafts     []Draft           `json:"drafts,omitempty"`
	Previews   []Preview         `json:"previews,omitempty"`
	Skipped    map[string]string `json:"skipped,omitempty"` // prospect id → why
	Next       string            `json:"next"`
}

const maxSteps = 5

func (in *CampaignInput) check() error {
	if strings.TrimSpace(in.Name) == "" {
		return refuse("name is required")
	}
	if len(in.Steps) == 0 || len(in.Steps) > maxSteps {
		return refuse("pass 1 to %d steps; 3 over about 12 days works best for creators", maxSteps)
	}
	if strings.TrimSpace(in.Steps[0].Subject) == "" {
		return refuse("the first step needs a subject")
	}
	for i, st := range in.Steps {
		if strings.TrimSpace(st.Body) == "" {
			return refuse("step %d has no body", i+1)
		}
		if i > 0 && st.DelayDays < 1 {
			return refuse("step %d needs delay_days of at least 1", i+1)
		}
		if n := strings.Count(st.Body, "http://") + strings.Count(st.Body, "https://"); n > 1 && i == 0 {
			return refuse("the first email may have at most one link; cold email with several links lands in spam")
		}
	}
	if len(in.ProspectIDs) == 0 || len(in.ProspectIDs) > maxBatch {
		return refuse("pass 1 to %d prospect_ids (add them first with add_prospects)", maxBatch)
	}
	switch in.ApprovalMode {
	case "":
		in.ApprovalMode = "every"
	case "every", "first", "automatic":
	default:
		return refuse("approval_mode must be every, first or automatic")
	}
	if in.WindowStartHour == 0 && in.WindowEndHour == 0 {
		in.WindowStartHour, in.WindowEndHour = 8, 17
	}
	if in.WindowStartHour < 0 || in.WindowEndHour > 24 || in.WindowStartHour >= in.WindowEndHour {
		return refuse("the sending window must run from window_start_hour to a later window_end_hour, 0 to 24")
	}
	return nil
}

// DraftOutreach creates the campaign, enrolls its prospects and writes every
// first email. With dry_run it only writes them.
func (s *Service) DraftOutreach(ctx context.Context, org db.Organization, in CampaignInput) (CampaignResult, error) {
	if err := s.requireEnabled(org); err != nil {
		return CampaignResult{}, err
	}
	if err := in.check(); err != nil {
		return CampaignResult{}, err
	}
	if strings.TrimSpace(org.MailingAddress) == "" {
		return CampaignResult{}, refuse("Set the brand's mailing address first: setup_sending(action: settings, mailing_address: ...). Every outreach email carries it.")
	}
	slug := "outreach-" + slugify(in.IdempotencyKey)
	if !in.DryRun && in.IdempotencyKey != "" {
		if seq, err := s.db.GetOutreachSequenceBySlug(ctx, db.GetOutreachSequenceBySlugParams{OrgID: sql.NullString{String: org.ID, Valid: true}, Slug: slug}); err == nil {
			return CampaignResult{CampaignID: seq.ID, Next: "This idempotency_key already made campaign " + seq.ID + "; nothing new was created."}, nil
		}
	}
	pool, err := s.activeMailboxes(ctx, org.ID)
	if err != nil {
		return CampaignResult{}, err
	}
	if in.TrackClicks || in.TrackOpens {
		if err := s.trackingReady(ctx, org.ID, pool); err != nil {
			return CampaignResult{}, err
		}
	}
	if in.ApprovalMode == "automatic" {
		for _, mb := range pool {
			if s.now().Sub(parseTS(mb.WarmupStartedAt)) < 30*24*time.Hour {
				return CampaignResult{}, refuse("approval_mode automatic needs 30 days of history on every mailbox; %s is newer. Use every or first.", mb.Email)
			}
		}
	}
	list, err := s.outreachList(ctx, org, in.List)
	if err != nil {
		return CampaignResult{}, err
	}

	res := CampaignResult{Skipped: map[string]string{}}
	var ready []db.Contact
	for _, id := range in.ProspectIDs {
		c, reason := s.enrollable(ctx, org, list, id)
		if reason != "" {
			res.Skipped[id] = reason
			continue
		}
		ready = append(ready, c)
	}

	seq := db.EmailSequence{
		ID: "preview", OrgID: sql.NullString{String: org.ID, Valid: true}, ListID: sql.NullInt64{Int64: list.ID, Valid: true},
		ApprovalMode: in.ApprovalMode, WindowStartHour: int64(in.WindowStartHour), WindowEndHour: int64(in.WindowEndHour),
		SendTimezone: sql.NullString{String: org.OutboundTimezone, Valid: true},
	}
	if in.DryRun {
		first := email.OutreachTemplate{Subject: in.Steps[0].Subject, Body: in.Steps[0].Body}
		load := map[string]int{}
		for _, c := range ready {
			mb := leastLoaded(pool, load)
			w, err := s.seq.WriteOutreachStep(ctx, seq, first, c, mb, 1, s.now(), s.rand())
			if err != nil {
				res.Skipped[c.ID] = err.Error()
				continue
			}
			load[mb.ID]++
			res.Previews = append(res.Previews, Preview{ProspectID: c.ID, To: c.Email, From: mb.Email, Subject: w.Subject, Body: w.Body, SendsAfter: ts(w.SendAfter)})
		}
		res.Enrolled = len(res.Previews)
		res.Next = "Dry run: nothing saved. Call again without dry_run to create the campaign."
		return res, nil
	}
	if len(ready) == 0 {
		return res, refuse("No prospect can be enrolled; see skipped.")
	}

	if in.IdempotencyKey == "" {
		slug = "outreach-" + uuid.NewString()[:8]
	}
	created, err := s.db.CreateSequence(ctx, db.CreateSequenceParams{
		ID: uuid.NewString(), OrgID: sql.NullString{String: org.ID, Valid: true}, ListID: sql.NullInt64{Int64: list.ID, Valid: true},
		Slug: slug, Name: strings.TrimSpace(in.Name), TriggerEvent: "manual", IsActive: sql.NullInt64{Int64: 1, Valid: true}, SequenceType: "outreach",
	})
	if err != nil {
		return res, err
	}
	res.CampaignID = created.ID
	if err := s.db.SetOutreachSequenceOptions(ctx, db.SetOutreachSequenceOptionsParams{
		ApprovalMode: in.ApprovalMode, WindowStartHour: int64(in.WindowStartHour), WindowEndHour: int64(in.WindowEndHour),
		TrackOpens: boolInt(in.TrackOpens), TrackClicks: boolInt(in.TrackClicks),
		SendTimezone: sql.NullString{String: org.OutboundTimezone, Valid: true}, ID: created.ID, OrgID: sql.NullString{String: org.ID, Valid: true},
	}); err != nil {
		return res, err
	}
	for i, st := range in.Steps {
		if _, err := s.db.CreateTemplate(ctx, db.CreateTemplateParams{
			ID: uuid.NewString(), SequenceID: sql.NullString{String: created.ID, Valid: true}, Position: int64(i + 1),
			DelayHours: int64(st.DelayDays) * 24, Subject: st.Subject, PlainText: sql.NullString{String: st.Body, Valid: true},
			TemplateType: sql.NullString{String: "none", Valid: true}, IsActive: sql.NullInt64{Int64: 1, Valid: true},
		}); err != nil {
			return res, err
		}
	}
	for _, c := range ready {
		if err := s.seq.StartSequenceByID(ctx, c.ID, created.ID); err != nil {
			res.Skipped[c.ID] = err.Error()
			continue
		}
		st, err := s.db.GetSequenceStateFull(ctx, db.GetSequenceStateFullParams{ContactID: sql.NullString{String: c.ID, Valid: true}, SequenceID: sql.NullString{String: created.ID, Valid: true}})
		if err == nil && st.StoppedReason != "" {
			res.Skipped[c.ID] = st.StoppedReason
			continue
		}
		res.Enrolled++
	}
	if rows, err := s.db.ListOutreachEmailsForSequence(ctx, sql.NullString{String: created.ID, Valid: true}); err == nil {
		for _, r := range rows {
			res.Drafts = append(res.Drafts, Draft{MessageID: r.ID, CampaignID: created.ID, CampaignName: created.Name, Step: r.Position,
				Preview: Preview{ProspectID: r.ContactID.String, To: r.ToEmail, From: r.FromEmail, Subject: r.Subject.String, Body: r.Body.String, SendsAfter: r.ScheduledFor}})
		}
	}
	if in.ApprovalMode == "automatic" {
		res.Next = fmt.Sprintf("%d first emails will send inside the window, paced per mailbox.", res.Enrolled)
	} else {
		res.Next = fmt.Sprintf("%d first emails wait for approval: review_drafts(action: list), then review_drafts(action: approve, ids: [...]).", res.Enrolled)
	}
	if len(res.Skipped) == 0 {
		res.Skipped = nil
	}
	return res, nil
}

// enrollable says why a contact can't be enrolled from a list, or "".
func (s *Service) enrollable(ctx context.Context, org db.Organization, list db.EmailList, contactID string) (db.Contact, string) {
	c, err := s.db.GetContactByID(ctx, contactID)
	if err != nil || c.OrgID.String != org.ID {
		return c, "no such prospect in this brand"
	}
	sub, err := s.db.GetListSubscriber(ctx, db.GetListSubscriberParams{ListID: list.ID, ContactID: c.ID})
	switch {
	case err != nil:
		return c, "not on the outreach list " + list.Name
	case sub.Status.String == "held":
		return c, "held: failed verification"
	case sub.Status.String != "prospect":
		return c, "status on " + list.Name + " is " + sub.Status.String
	}
	if s.suppressed(ctx, org.ID, c.Email) {
		return c, "suppressed"
	}
	return c, ""
}

// activeMailboxes is the brand's mailbox pool.
func (s *Service) activeMailboxes(ctx context.Context, orgID string) ([]db.Mailbox, error) {
	all, err := s.db.ListMailboxes(ctx, orgID)
	if err != nil {
		return nil, err
	}
	var pool []db.Mailbox
	for _, mb := range all {
		if mb.Status == "active" {
			pool = append(pool, mb)
		}
	}
	if len(pool) == 0 {
		return nil, refuse("No active mailbox to send from. Connect one with setup_sending(action: add_mailboxes), or check outreach_overview for paused ones.")
	}
	return pool, nil
}

func leastLoaded(pool []db.Mailbox, load map[string]int) db.Mailbox {
	best := pool[0]
	for _, mb := range pool[1:] {
		if load[mb.ID] < load[best.ID] {
			best = mb
		}
	}
	return best
}

// trackingReady refuses open and click tracking unless every mailbox in the
// pool is on a domain whose go. subdomain points at this server.
func (s *Service) trackingReady(ctx context.Context, orgID string, pool []db.Mailbox) error {
	for _, mb := range pool {
		d, err := s.db.GetOutreachDomain(ctx, db.GetOutreachDomainParams{ID: mb.DomainID, OrgID: orgID})
		if err != nil {
			return err
		}
		if d.TrackingHost == "" {
			return refuse("%s has no tracking subdomain. Run setup_sending(action: add_domain, domain: %q, click_tracking: true), or create the campaign without tracking.", d.Domain, d.Domain)
		}
		var c DNSCheck
		_ = json.Unmarshal([]byte(d.DnsCheck), &c)
		if c.Tracking == nil || !*c.Tracking {
			return refuse("%s doesn't point at this server yet (CNAME go to %s). Add it, then setup_sending(action: check_domain).", d.TrackingHost, s.tracking)
		}
	}
	return nil
}
