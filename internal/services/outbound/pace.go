package outbound

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/events"
	"github.com/outlet-sh/outlet/internal/services/email"
)

// The pace dial is the mailbox pool's daily_new_prospects limit: how many
// people a day get their first email, across the pool. Follow-ups and
// answers aren't counted; they finish what was started. At the limit, first
// emails wait for the next day's window: nothing is dropped, and the owner
// hears about it (the outreach.paused webhook, once a day, and the overview).

// PacePreset is one setting of the dial.
type PacePreset struct {
	Name              string `json:"name"`
	DailyNewProspects int64  `json:"daily_new_prospects"` // 0: no limit
}

// PacePresets are the dial's settings, slowest first.
var PacePresets = []PacePreset{{"light", 25}, {"typical", 50}, {"growth", 100}, {"unlimited", 0}}

// The send_counts senders the dial counts under, on the pool's pathway: the
// day's first emails, and whether the owner was told the day's limit was
// reached.
const (
	newProspectsSender = "pace:new-prospects"
	paceNoticeSender   = "pace:notice"
)

// PaceEstimate is what one setting means for the brand.
type PaceEstimate struct {
	Pace              string `json:"pace"` // light, typical, growth, unlimited, or custom
	DailyNewProspects int64  `json:"daily_new_prospects"`
	// SendsPerDay is the most emails a day once follow-ups catch up: new
	// prospects times the steps in a campaign (replies stop some sooner).
	SendsPerDay int64 `json:"sends_per_day"`
	// MailboxesNeeded is how many mailboxes carry that at the pool's daily
	// limit per mailbox; 0 when either has no limit.
	MailboxesNeeded int64 `json:"mailboxes_needed"`
}

// Pace is the dial as it stands, and what each preset would mean.
type Pace struct {
	PoolID            string         `json:"pool_id"`
	Current           PaceEstimate   `json:"current"`
	NewToday          int64          `json:"new_today"` // first emails sent today
	StepsPerCampaign  float64        `json:"steps_per_campaign"`
	MailboxDailyLimit int64          `json:"mailbox_daily_limit"` // the pool's, per mailbox; 0: none
	MailboxesActive   int64          `json:"mailboxes_active"`
	Capacity          int64          `json:"capacity"` // emails a day the active mailboxes may send at their limit now; 0: no limit
	Presets           []PaceEstimate `json:"presets"`
	// Cost says what is known about the price. Outlet has no price per
	// email: mailbox services bill per mailbox or domain, priced by a quote.
	Cost string `json:"cost"`
}

func paceName(n int64) string {
	for _, p := range PacePresets {
		if p.DailyNewProspects == n {
			return p.Name
		}
	}
	return "custom"
}

// outreachPool is the mailbox pool carrying the brand's outreach.
func (s *Service) outreachPool(ctx context.Context, org db.Organization) (db.SendingPathway, error) {
	p, err := s.db.GetPathwayForKind(ctx, db.GetPathwayForKindParams{OrgID: org.ID, Kind: email.KindOutreach.Name()})
	if err != nil {
		return p, refuse("No mailbox pool carries this brand's outreach. Connect mailboxes in Outbound, Setup.")
	}
	if p.Provider != email.ProviderMailboxes {
		return p, refuse("The brand's outreach rides %q, not a mailbox pool; the pace dial is a mailbox pool's.", p.Name)
	}
	return p, nil
}

// Pace reads the dial.
func (s *Service) Pace(ctx context.Context, org db.Organization) (Pace, error) {
	p, err := s.outreachPool(ctx, org)
	if err != nil {
		return Pace{}, err
	}
	steps, _ := s.db.AverageOutreachSteps(ctx, sql.NullString{String: org.ID, Valid: true})
	if steps == 0 {
		steps = 3 // what a new campaign is advised to have
	}
	out := Pace{PoolID: p.ID, StepsPerCampaign: float64(int64(steps*10+0.5)) / 10, MailboxDailyLimit: p.DailyLimit,
		Cost: "Not known to Outlet: mailbox services bill per mailbox or domain, priced by a quote (setup_sending quote). Sending itself costs nothing more."}
	out.NewToday, _ = s.mailer.SentSince(ctx, org, p.ID, newProspectsSender, s.now())
	if mbs, err := s.db.ListPoolMailboxes(ctx, sql.NullString{String: p.ID, Valid: true}); err == nil {
		unlimited := false
		for _, mb := range mbs {
			if mb.Status != "active" {
				continue
			}
			out.MailboxesActive++
			h := s.mailboxHealth(ctx, mb, org)
			unlimited = unlimited || !h.Limited
			out.Capacity += h.AllowToday
		}
		if unlimited {
			out.Capacity = 0
		}
	}
	estimate := func(n int64) PaceEstimate {
		e := PaceEstimate{Pace: paceName(n), DailyNewProspects: n}
		if n > 0 {
			e.SendsPerDay = int64(float64(n)*steps + 0.5)
			if p.DailyLimit > 0 {
				e.MailboxesNeeded = (e.SendsPerDay + p.DailyLimit - 1) / p.DailyLimit
			}
		}
		return e
	}
	out.Current = estimate(p.DailyNewProspects)
	for _, pr := range PacePresets {
		out.Presets = append(out.Presets, estimate(pr.DailyNewProspects))
	}
	return out, nil
}

// SetPace sets the dial to a preset by name, or to a number of new prospects
// a day (dailyNew, when pace is empty or "custom"). 0 is no limit.
func (s *Service) SetPace(ctx context.Context, org db.Organization, pace string, dailyNew *int64) (Pace, error) {
	p, err := s.outreachPool(ctx, org)
	if err != nil {
		return Pace{}, err
	}
	var n int64
	switch name := strings.ToLower(strings.TrimSpace(pace)); name {
	case "", "custom":
		if dailyNew == nil || *dailyNew < 0 {
			return Pace{}, refuse("Set pace to light (25 new prospects a day), typical (50), growth (100) or unlimited, or give daily_new_prospects.")
		}
		n = *dailyNew
	default:
		found := false
		for _, pr := range PacePresets {
			if pr.Name == name {
				n, found = pr.DailyNewProspects, true
			}
		}
		if !found {
			return Pace{}, refuse("pace is light, typical, growth or unlimited.")
		}
	}
	if err := s.db.SetPathwayDailyNewProspects(ctx, db.SetPathwayDailyNewProspectsParams{DailyNewProspects: n, ID: p.ID, OrgID: org.ID}); err != nil {
		return Pace{}, err
	}
	return s.Pace(ctx, org)
}

// reserveNewProspect counts one first email against the dial, or says the
// day's limit is reached. release gives the place back when the send fails.
func (s *Service) reserveNewProspect(ctx context.Context, org db.Organization, p db.SendingPathway) (release func(), ok bool, err error) {
	if p.DailyNewProspects <= 0 {
		return func() {}, true, nil
	}
	day := email.Day(s.now(), org)
	if err := s.db.EnsureSendCount(ctx, db.EnsureSendCountParams{PathwayID: p.ID, SenderID: newProspectsSender, Day: day}); err != nil {
		return nil, false, err
	}
	n, err := s.db.IncrementSendCount(ctx, db.IncrementSendCountParams{PathwayID: p.ID, SenderID: newProspectsSender, Day: day, Allowance: p.DailyNewProspects})
	if err != nil || n == 0 {
		return nil, false, err
	}
	return func() {
		_ = s.db.RefundSendCount(context.WithoutCancel(ctx), db.RefundSendCountParams{PathwayID: p.ID, SenderID: newProspectsSender, Day: day})
	}, true, nil
}

// paced holds a first email for the next day's window, and tells the owner
// once a day that the day's new prospects were reached.
func (s *Service) paced(ctx context.Context, org db.Organization, p db.SendingPathway, seq db.EmailSequence, m db.NextOutreachForMailboxRow) error {
	now := s.now()
	loc := email.LoadLocation(s.contactTimezone(ctx, seq, m.ContactID.String), seq.SendTimezone.String, org.OutboundTimezone)
	l := now.In(email.LoadLocation(org.OutboundTimezone))
	tomorrow := time.Date(l.Year(), l.Month(), l.Day()+1, 0, 0, 0, 0, l.Location())
	start, end := int(seq.WindowStartHour), int(seq.WindowEndHour)
	if start == 0 && end == 0 {
		end = 24
	}
	next := email.NextWindow(tomorrow, loc, start, end)
	reason := fmt.Sprintf("today's %d new prospects are reached (the pace dial)", p.DailyNewProspects)
	if err := s.db.ReleaseOutreachEmail(ctx, db.ReleaseOutreachEmailParams{ErrorMessage: nullStr(reason), ScheduledFor: ts(next), ID: m.ID}); err != nil {
		return err
	}
	day := email.Day(now, org)
	_ = s.db.EnsureSendCount(ctx, db.EnsureSendCountParams{PathwayID: p.ID, SenderID: paceNoticeSender, Day: day})
	if n, err := s.db.IncrementSendCount(ctx, db.IncrementSendCountParams{PathwayID: p.ID, SenderID: paceNoticeSender, Day: day, Allowance: 1}); err == nil && n == 1 {
		s.db.PublishOutreachPaused(events.OutreachPausedEvent{OrgID: org.ID, Scope: "pace",
			Reason: reason + "; first emails wait for the next day's window, follow-ups go on", ResumesAt: ts(next), Timestamp: now.UTC()})
	}
	return nil
}

// pacedToday says whether the day's new prospects are reached, for the
// overview.
func (s *Service) pacedToday(ctx context.Context, org db.Organization) (limit int64, reached bool) {
	p, err := s.outreachPool(ctx, org)
	if err != nil || p.DailyNewProspects <= 0 {
		return 0, false
	}
	n, err := s.mailer.SentSince(ctx, org, p.ID, newProspectsSender, s.now())
	return p.DailyNewProspects, err == nil && n >= p.DailyNewProspects
}
