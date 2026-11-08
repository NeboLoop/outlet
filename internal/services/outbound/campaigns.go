package outbound

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/db"

	"github.com/google/uuid"
)

// StepInput is one email in a sequence. An empty subject on a later step
// means a reply in the same thread ("Re: " the first subject).
type StepInput struct {
	DelayDays int    `json:"delay_days,omitempty"` // business days after the previous step; the first step ignores it
	Subject   string `json:"subject,omitempty"`
	Body      string `json:"body"`
}

// CampaignInput creates a campaign, its steps, its mailbox pool and its
// prospects in one go.
type CampaignInput struct {
	Name            string      `json:"name"`
	Steps           []StepInput `json:"steps"`
	ProspectIDs     []string    `json:"prospect_ids"`
	MailboxIDs      []string    `json:"mailbox_ids,omitempty"`   // default: every active mailbox of the brand
	ApprovalMode    string      `json:"approval_mode,omitempty"` // every (default), first, automatic
	TrackClicks     bool        `json:"track_clicks,omitempty"`  // links go through each domain's go. subdomain
	WindowStartHour int         `json:"window_start_hour,omitempty"`
	WindowEndHour   int         `json:"window_end_hour,omitempty"`
	IdempotencyKey  string      `json:"idempotency_key,omitempty"`
	DryRun          bool        `json:"dry_run,omitempty"`
}

// Draft is one rendered first email, for the caller to read.
type Draft struct {
	MessageID  string `json:"message_id,omitempty"`
	ProspectID string `json:"prospect_id"`
	To         string `json:"to"`
	From       string `json:"from"`
	Subject    string `json:"subject"`
	Body       string `json:"body"`
	SendsAfter string `json:"sends_after,omitempty"`
}

// CampaignResult reports what draft_outreach did.
type CampaignResult struct {
	CampaignID string            `json:"campaign_id,omitempty"`
	Enrolled   int               `json:"enrolled"`
	Drafts     []Draft           `json:"drafts,omitempty"`
	Skipped    map[string]string `json:"skipped,omitempty"` // prospect id → why
	Next       string            `json:"next"`
}

const maxSteps = 5

func (in *CampaignInput) check() error {
	if strings.TrimSpace(in.Name) == "" {
		return refuse("name is required")
	}
	if len(in.Steps) == 0 || len(in.Steps) > maxSteps {
		return refuse("pass 1 to %d steps; 3 over about 12 business days works best for creators", maxSteps)
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

// footer is what every first email ends with: the opt-out line and the
// brand's mailing address (CAN-SPAM). Without an address nothing launches.
func footer(st db.OutboundSetting) string {
	return "\n\n" + strings.TrimSpace(st.OptOutLine) + "\n" + strings.TrimSpace(st.MailingAddress)
}

// DraftOutreach creates the campaign and drafts every first email. With
// dry_run it only renders them.
func (s *Service) DraftOutreach(ctx context.Context, org db.Organization, in CampaignInput) (CampaignResult, error) {
	if err := s.requireEnabled(org); err != nil {
		return CampaignResult{}, err
	}
	if err := in.check(); err != nil {
		return CampaignResult{}, err
	}
	st, err := s.settings(ctx, org.ID)
	if err != nil {
		return CampaignResult{}, err
	}
	if strings.TrimSpace(st.MailingAddress) == "" {
		return CampaignResult{}, refuse("Set the brand's mailing address first: setup_sending(action: settings, mailing_address: ...). Every first email carries it.")
	}
	if strings.TrimSpace(st.OptOutLine) == "" {
		return CampaignResult{}, refuse("Set an opt-out line first: setup_sending(action: settings, opt_out_line: ...).")
	}
	if !in.DryRun && in.IdempotencyKey != "" {
		if c, err := s.db.GetOutboundCampaignByKey(ctx, db.GetOutboundCampaignByKeyParams{OrgID: org.ID, IdempotencyKey: in.IdempotencyKey}); err == nil {
			return CampaignResult{CampaignID: c.ID, Next: "This idempotency_key already made campaign " + c.ID + "; nothing new was created."}, nil
		}
	}
	pool, err := s.pool(ctx, org.ID, in.MailboxIDs)
	if err != nil {
		return CampaignResult{}, err
	}
	if in.TrackClicks {
		if err := s.trackingReady(ctx, org.ID, pool); err != nil {
			return CampaignResult{}, err
		}
	}
	if in.ApprovalMode == "automatic" {
		for _, mb := range pool {
			if s.now().Sub(parseTS(mb.WarmupStartedAt)) < 30*24*time.Hour {
				return CampaignResult{}, refuse("approval_mode automatic needs 30 days of history on every mailbox in the pool; %s is newer. Use every or first.", mb.Email)
			}
		}
	}
	prospects, err := s.db.ListOutboundProspectsByIDs(ctx, db.ListOutboundProspectsByIDsParams{OrgID: org.ID, Ids: in.ProspectIDs})
	if err != nil {
		return CampaignResult{}, err
	}
	res := CampaignResult{Skipped: map[string]string{}}
	found := map[string]bool{}
	for _, p := range prospects {
		found[p.ID] = true
	}
	for _, id := range in.ProspectIDs {
		if !found[id] {
			res.Skipped[id] = "no such prospect in this brand"
		}
	}

	// Render first, so a template with a missing value fails before anything is saved.
	type planned struct {
		p       db.OutboundProspect
		mb      db.OutboundMailbox
		subject string
		body    string
	}
	load := map[string]int{}
	var plan []planned
	r := s.rand()
	for _, p := range prospects {
		if reason := sendable(p); reason != "" {
			res.Skipped[p.ID] = reason
			continue
		}
		if s.suppressed(ctx, org.ID, p.Email) {
			res.Skipped[p.ID] = "suppressed"
			continue
		}
		mb := leastLoaded(pool, load)
		fields := mergeFields(p, mb)
		subject, err := render(in.Steps[0].Subject, fields, r)
		if err != nil {
			res.Skipped[p.ID] = "subject: " + err.Error()
			continue
		}
		body, err := render(in.Steps[0].Body, fields, r)
		if err != nil {
			res.Skipped[p.ID] = "body: " + err.Error()
			continue
		}
		load[mb.ID]++
		plan = append(plan, planned{p, mb, subject, signed(body, mb) + footer(st)})
	}

	if in.DryRun {
		for _, pl := range plan {
			res.Drafts = append(res.Drafts, Draft{ProspectID: pl.p.ID, To: pl.p.Email, From: pl.mb.Email, Subject: pl.subject, Body: pl.body})
		}
		res.Enrolled = len(plan)
		res.Next = "Dry run: nothing saved. Call again without dry_run to create the campaign."
		return res, nil
	}
	if len(plan) == 0 {
		return res, refuse("No prospect can be enrolled; see skipped.")
	}

	status := "draft"
	if in.ApprovalMode == "automatic" {
		status = "approved"
	}
	camp, err := s.db.CreateOutboundCampaign(ctx, db.CreateOutboundCampaignParams{
		ID: uuid.NewString(), OrgID: org.ID, Name: strings.TrimSpace(in.Name), ApprovalMode: in.ApprovalMode, TrackClicks: boolInt(in.TrackClicks),
		WindowStartHour: int64(in.WindowStartHour), WindowEndHour: int64(in.WindowEndHour), IdempotencyKey: in.IdempotencyKey,
	})
	if err != nil {
		return res, err
	}
	res.CampaignID = camp.ID
	for i, step := range in.Steps {
		if _, err := s.db.CreateOutboundStep(ctx, db.CreateOutboundStepParams{
			ID: uuid.NewString(), CampaignID: camp.ID, Position: int64(i + 1), DelayDays: int64(step.DelayDays),
			Subject: step.Subject, Body: step.Body,
		}); err != nil {
			return res, err
		}
	}
	now := s.now()
	for _, pl := range plan {
		notBefore := nextWindow(now, loadLocation(pl.p.Timezone, st.Timezone), in.WindowStartHour, in.WindowEndHour)
		err := s.db.ExecTx(ctx, func(q *db.Queries) error {
			e, err := q.CreateOutboundEnrollment(ctx, db.CreateOutboundEnrollmentParams{
				ID: uuid.NewString(), OrgID: org.ID, CampaignID: camp.ID, ProspectID: pl.p.ID, MailboxID: pl.mb.ID,
				NextPosition: 2, ThreadSubject: pl.subject,
			})
			if err != nil {
				return err
			}
			m, err := q.CreateOutboundMessage(ctx, db.CreateOutboundMessageParams{
				ID: uuid.NewString(), OrgID: org.ID, EnrollmentID: sql.NullString{String: e.ID, Valid: true},
				MailboxID: pl.mb.ID, ProspectID: pl.p.ID, Position: 1, ToEmail: pl.p.Email,
				Subject: pl.subject, Body: pl.body, Status: status,
				MessageID: newMessageID(pl.mb.Email), NotBefore: ts(notBefore),
			})
			if err != nil {
				return err
			}
			if err := q.SetOutboundProspectStatus(ctx, db.SetOutboundProspectStatusParams{Status: "in_sequence", ID: pl.p.ID}); err != nil {
				return err
			}
			res.Drafts = append(res.Drafts, Draft{MessageID: m.ID, ProspectID: pl.p.ID, To: pl.p.Email, From: pl.mb.Email, Subject: pl.subject, Body: pl.body, SendsAfter: m.NotBefore})
			return nil
		})
		if err != nil {
			res.Skipped[pl.p.ID] = err.Error()
			continue
		}
		res.Enrolled++
	}
	if status == "draft" {
		res.Next = fmt.Sprintf("%d first emails wait for approval: review_drafts(action: list), then review_drafts(action: approve, ids: [...]).", res.Enrolled)
	} else {
		res.Next = fmt.Sprintf("%d first emails will send inside the window, paced per mailbox.", res.Enrolled)
	}
	if len(res.Skipped) == 0 {
		res.Skipped = nil
	}
	return res, nil
}

// sendable says why a prospect can't be enrolled, or "".
func sendable(p db.OutboundProspect) string {
	switch p.Status {
	case "new":
		return ""
	case "held":
		return "held: failed verification (" + p.VerifyResult + ")"
	default:
		return "status is " + p.Status
	}
}

// pool is the campaign's mailboxes: the ones named, or every active one.
func (s *Service) pool(ctx context.Context, orgID string, ids []string) ([]db.OutboundMailbox, error) {
	all, err := s.db.ListOutboundMailboxes(ctx, orgID)
	if err != nil {
		return nil, err
	}
	want := map[string]bool{}
	for _, id := range ids {
		want[id] = true
	}
	var pool []db.OutboundMailbox
	for _, mb := range all {
		if mb.Status == "active" && (len(ids) == 0 || want[mb.ID]) {
			pool = append(pool, mb)
		}
	}
	if len(pool) == 0 {
		return nil, refuse("No active mailbox to send from. Connect one with setup_sending(action: add_mailboxes), or check outreach_overview for paused ones.")
	}
	sort.Slice(pool, func(i, j int) bool { return pool[i].Email < pool[j].Email })
	return pool, nil
}

func leastLoaded(pool []db.OutboundMailbox, load map[string]int) db.OutboundMailbox {
	best := pool[0]
	for _, mb := range pool[1:] {
		if load[mb.ID] < load[best.ID] {
			best = mb
		}
	}
	return best
}

func signed(body string, mb db.OutboundMailbox) string {
	body = strings.TrimRight(body, "\n ")
	if sig := strings.TrimSpace(mb.Signature); sig != "" {
		body += "\n\n" + sig
	}
	return body
}

func newMessageID(fromEmail string) string {
	return uuid.NewString() + "@" + domainOf(fromEmail)
}

// draftNextSteps writes the next step's email for every enrollment that is
// due. A follow-up replies in the same thread from the same mailbox.
func (s *Service) draftNextSteps(ctx context.Context) {
	now := s.now()
	due, err := s.db.ListDueOutboundEnrollments(ctx, db.ListDueOutboundEnrollmentsParams{NextAt: sql.NullString{String: ts(now), Valid: true}, Limit: 200})
	if err != nil {
		return
	}
	for _, e := range due {
		if err := s.draftStep(ctx, e, now); err != nil {
			_ = s.db.SetOutboundEnrollmentStatus(ctx, db.SetOutboundEnrollmentStatusParams{Status: "stopped", ID: e.ID})
		}
	}
}

func (s *Service) draftStep(ctx context.Context, e db.OutboundEnrollment, now time.Time) error {
	camp, err := s.db.GetOutboundCampaign(ctx, db.GetOutboundCampaignParams{ID: e.CampaignID, OrgID: e.OrgID})
	if err != nil {
		return err
	}
	steps, err := s.db.ListOutboundSteps(ctx, e.CampaignID)
	if err != nil {
		return err
	}
	var step *db.OutboundStep
	for i := range steps {
		if steps[i].Position == e.NextPosition {
			step = &steps[i]
		}
	}
	if step == nil {
		return s.db.SetOutboundEnrollmentStatus(ctx, db.SetOutboundEnrollmentStatusParams{Status: "done", ID: e.ID})
	}
	p, err := s.db.GetOutboundProspect(ctx, db.GetOutboundProspectParams{ID: e.ProspectID, OrgID: e.OrgID})
	if err != nil {
		return err
	}
	mb, err := s.db.GetOutboundMailbox(ctx, db.GetOutboundMailboxParams{ID: e.MailboxID, OrgID: e.OrgID})
	if err != nil {
		return err
	}
	if s.suppressed(ctx, e.OrgID, p.Email) {
		return s.db.SetOutboundEnrollmentStatus(ctx, db.SetOutboundEnrollmentStatusParams{Status: "unsubscribed", ID: e.ID})
	}
	thread, err := s.db.ListOutboundThread(ctx, sql.NullString{String: e.ID, Valid: true})
	if err != nil || len(thread) == 0 {
		return errors.New("no sent email to follow up on")
	}
	st, err := s.settings(ctx, e.OrgID)
	if err != nil {
		return err
	}
	r := s.rand()
	fields := mergeFields(p, mb)
	subject := "Re: " + e.ThreadSubject
	if strings.TrimSpace(step.Subject) != "" {
		if subject, err = render(step.Subject, fields, r); err != nil {
			return err
		}
	}
	body, err := render(step.Body, fields, r)
	if err != nil {
		return err
	}
	refs := make([]string, 0, len(thread))
	for _, m := range thread {
		refs = append(refs, m.MessageID)
	}
	status := "draft"
	if camp.ApprovalMode != "every" {
		status = "approved"
	}
	notBefore := nextWindow(now, loadLocation(p.Timezone, st.Timezone), int(camp.WindowStartHour), int(camp.WindowEndHour))
	return s.db.ExecTx(ctx, func(q *db.Queries) error {
		if _, err := q.CreateOutboundMessage(ctx, db.CreateOutboundMessageParams{
			ID: uuid.NewString(), OrgID: e.OrgID, EnrollmentID: sql.NullString{String: e.ID, Valid: true},
			MailboxID: mb.ID, ProspectID: p.ID, Position: step.Position, ToEmail: p.Email,
			Subject: subject, Body: signed(body, mb) + footer(st), Status: status,
			MessageID: newMessageID(mb.Email), InReplyTo: thread[len(thread)-1].MessageID,
			Refs: strings.Join(refs, " "), NotBefore: ts(notBefore),
		}); err != nil {
			return err
		}
		// The next step is scheduled when this one is sent.
		return q.SetOutboundEnrollmentNext(ctx, db.SetOutboundEnrollmentNextParams{NextPosition: step.Position + 1, NextAt: sql.NullString{}, ID: e.ID})
	})
}
