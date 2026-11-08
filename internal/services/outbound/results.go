package outbound

import (
	"context"
	"database/sql"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/email"
)

// StepResult is how one step of a campaign did.
type StepResult struct {
	Step      int64   `json:"step"`
	Sent      int64   `json:"sent"`
	Replies   int64   `json:"replies"`    // people whose first reply came after this step
	ReplyRate float64 `json:"reply_rate"` // replies / sent, percent
	Opened    int64   `json:"opened,omitempty"`
	Clicked   int64   `json:"clicked,omitempty"`
}

// CampaignResults is how a campaign is doing. Reply rate is the measure:
// people who wrote back, out of people who got the first email. Opens are
// inflated by Apple Mail and count only when the campaign tracks them.
type CampaignResults struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Status     string           `json:"status"`
	Enrolled   int64            `json:"enrolled"`
	Reached    int64            `json:"reached"`
	Replied    int64            `json:"replied"`
	ReplyRate  float64          `json:"reply_rate"`
	Opened     int64            `json:"opened"`
	OpenRate   float64          `json:"open_rate"`
	Clicked    int64            `json:"clicked"`
	ClickRate  float64          `json:"click_rate"`
	Interested int64            `json:"interested"`
	Partners   int64            `json:"partners"`
	Bounced    int64            `json:"bounced"`
	BounceRate float64          `json:"bounce_rate"`
	// BounceLimit is the bounce rate, in percent, at which the pathway
	// carrying the brand's outreach pauses a mailbox; 0 when it never does.
	BounceLimit float64 `json:"bounce_limit"`
	SpamBlocks int64            `json:"spam_blocks"`
	OptedOut   int64            `json:"opted_out"`
	InProgress int64            `json:"in_progress"`
	NoReply    int64            `json:"finished_no_reply"`
	Waiting    int64            `json:"drafts_waiting"`
	TrackOpens bool             `json:"track_opens"`
	TrackClick bool             `json:"track_clicks"`
	Steps      []StepResult     `json:"steps,omitempty"`
	Replies    map[string]int64 `json:"replies_by_kind,omitempty"`
}

func pct(n, d int64) float64 {
	if d == 0 {
		return 0
	}
	return float64(int64(1000*float64(n)/float64(d)+0.5)) / 10
}

// CampaignResults measures one outreach sequence; steps adds the per-step view.
func (s *Service) CampaignResults(ctx context.Context, c db.EmailSequence, steps bool) (CampaignResults, error) {
	r := CampaignResults{ID: c.ID, Name: c.Name, Status: "active", Replies: map[string]int64{}, TrackOpens: c.TrackOpens == 1, TrackClick: c.TrackClicks == 1}
	if c.IsActive.Int64 != 1 {
		r.Status = "paused"
	}
	seqID := sql.NullString{String: c.ID, Valid: true}
	states, err := s.db.OutreachSequenceStates(ctx, seqID)
	if err != nil {
		return r, err
	}
	for _, st := range states {
		r.Enrolled += st.N
		switch st.State {
		case "active":
			r.InProgress += st.N
		case "done":
			r.NoReply += st.N
		}
	}
	sent, err := s.db.OutreachSentByStep(ctx, seqID)
	if err != nil {
		return r, err
	}
	byStep := map[int64]db.OutreachSentByStepRow{}
	for _, x := range sent {
		byStep[x.Position] = x
	}
	r.Reached = byStep[1].Sent
	// Drafts waiting are the campaign's emails still to approve; an approved
	// email waiting for its window or spacing is not one.
	if r.Waiting, err = s.db.CountOutreachDraftsInSequence(ctx, seqID); err != nil {
		return r, err
	}
	if e, err := s.db.OutreachPeopleEngaged(ctx, seqID); err == nil {
		r.Opened, r.Clicked = e.Opened, e.Clicked
	}
	kinds, err := s.db.OutreachRepliesByKind(ctx, seqID)
	if err != nil {
		return r, err
	}
	for _, x := range kinds {
		r.Replies[x.Classification] = x.N
		switch x.Classification {
		case "bounce":
			r.Bounced = x.N
		case "spam_block":
			r.SpamBlocks = x.N
		case "out_of_office":
		case "unsubscribe", "complaint":
			r.OptedOut += x.N
			r.Replied += x.N
		case "interested":
			r.Interested += x.N
			r.Replied += x.N
		default:
			r.Replied += x.N
		}
	}
	r.Partners, _ = s.db.OutreachPartners(ctx, seqID)
	r.Interested = max(r.Interested, r.Partners)
	r.ReplyRate, r.OpenRate, r.ClickRate, r.BounceRate = pct(r.Replied, r.Reached), pct(r.Opened, r.Reached), pct(r.Clicked, r.Reached), pct(r.Bounced, r.Reached)
	if p, err := s.db.GetPathwayForKind(ctx, db.GetPathwayForKindParams{OrgID: c.OrgID.String, Kind: email.KindOutreach.Name()}); err == nil {
		r.BounceLimit = float64(int64(1000*p.PauseBounceRate+0.5)) / 10
	}
	if !steps {
		r.Replies = nil
		return r, nil
	}
	replies, err := s.db.OutreachRepliesByStep(ctx, seqID)
	if err != nil {
		return r, err
	}
	repliesBy := map[int64]int64{}
	for _, x := range replies {
		repliesBy[x.Step] = x.N
	}
	all, err := s.db.ListSequenceSteps(ctx, seqID)
	if err != nil {
		return r, err
	}
	for _, st := range all {
		x := byStep[st.Position]
		r.Steps = append(r.Steps, StepResult{Step: st.Position, Sent: x.Sent, Replies: repliesBy[st.Position], ReplyRate: pct(repliesBy[st.Position], x.Sent), Opened: x.Opened, Clicked: x.Clicked})
	}
	return r, nil
}
