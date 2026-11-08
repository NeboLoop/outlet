package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/db"
)

// StepResult is how one step of a sequence did.
type StepResult struct {
	Step      int64   `json:"step"`
	Sent      int64   `json:"sent"`
	Replies   int64   `json:"replies"`           // people whose first reply came after this step
	ReplyRate float64 `json:"reply_rate"`        // replies / sent, in percent
	Clicked   int64   `json:"clicked,omitempty"` // people who clicked this step's link
}

// CampaignResults is how a campaign is doing. Reply rate is the measure:
// people who wrote back, out of people who got the first email.
type CampaignResults struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Status     string           `json:"status"`
	Enrolled   int64            `json:"enrolled"`
	Reached    int64            `json:"reached"`     // got the first email
	Replied    int64            `json:"replied"`     // wrote back (not bounces or auto-replies)
	ReplyRate  float64          `json:"reply_rate"`  // replied / reached, percent
	Clicked    int64            `json:"clicked"`     // people who clicked a link (campaigns with track_clicks)
	ClickRate  float64          `json:"click_rate"`  // clicked / reached, percent
	Interested int64            `json:"interested"`  // classified interested, or now a partner
	Partners   int64            `json:"partners"`    // said yes
	Bounced    int64            `json:"bounced"`     //
	BounceRate float64          `json:"bounce_rate"` // bounced / reached, percent
	OptedOut   int64            `json:"opted_out"`   // unsubscribed or complained
	InProgress int64            `json:"in_progress"` // still in the sequence
	NoReply    int64            `json:"finished_no_reply"`
	Waiting    int64            `json:"drafts_waiting"` // first emails not yet approved or sent
	Steps      []StepResult     `json:"steps,omitempty"`
	Replies    map[string]int64 `json:"replies_by_kind,omitempty"`
}

func pct(n, d int64) float64 {
	if d == 0 {
		return 0
	}
	return float64(int64(1000*float64(n)/float64(d)+0.5)) / 10
}

// CampaignResults measures one campaign; steps adds the per-step breakdown.
func (s *Service) CampaignResults(ctx context.Context, c db.OutboundCampaign, steps bool) (CampaignResults, error) {
	r := CampaignResults{ID: c.ID, Name: c.Name, Status: c.Status, Replies: map[string]int64{}}
	enr, err := s.db.OutboundCampaignEnrollmentCounts(ctx, c.ID)
	if err != nil {
		return r, err
	}
	for _, e := range enr {
		r.Enrolled += e.N
		switch e.Status {
		case "active", "paused":
			r.InProgress += e.N
		case "done":
			r.NoReply += e.N
		}
	}
	sent, err := s.db.OutboundCampaignSentByStep(ctx, c.ID)
	if err != nil {
		return r, err
	}
	sentBy := map[int64]int64{}
	for _, x := range sent {
		sentBy[x.Position] = x.N
	}
	r.Reached = sentBy[1]
	r.Waiting = r.Enrolled - r.Reached
	classes, err := s.db.OutboundCampaignRepliesByClass(ctx, c.ID)
	if err != nil {
		return r, err
	}
	for _, x := range classes {
		r.Replies[x.Classification] = x.N
		switch x.Classification {
		case "bounce":
			r.Bounced = x.N
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
	r.Partners, _ = s.db.OutboundCampaignPartners(ctx, c.ID)
	r.Interested = max(r.Interested, r.Partners)
	r.ReplyRate = pct(r.Replied, r.Reached)
	r.Clicked, _ = s.db.OutboundCampaignClickers(ctx, c.ID)
	r.ClickRate = pct(r.Clicked, r.Reached)
	r.BounceRate = pct(r.Bounced, r.Reached)
	if !steps {
		r.Replies = nil
		return r, nil
	}
	byStep, err := s.db.OutboundCampaignRepliesByStep(ctx, c.ID)
	if err != nil {
		return r, err
	}
	repliesBy := map[int64]int64{}
	for _, x := range byStep {
		repliesBy[x.Step] = x.N
	}
	clicks, err := s.db.OutboundCampaignClicksByStep(ctx, c.ID)
	if err != nil {
		return r, err
	}
	clicksBy := map[int64]int64{}
	for _, x := range clicks {
		clicksBy[x.Position] = x.N
	}
	all, err := s.db.ListOutboundSteps(ctx, c.ID)
	if err != nil {
		return r, err
	}
	for _, st := range all {
		r.Steps = append(r.Steps, StepResult{Step: st.Position, Sent: sentBy[st.Position], Replies: repliesBy[st.Position], ReplyRate: pct(repliesBy[st.Position], sentBy[st.Position]), Clicked: clicksBy[st.Position]})
	}
	return r, nil
}
