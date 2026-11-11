package outbound

import (
	"context"
	"database/sql"
	"strings"

	"github.com/outlet-sh/outlet/internal/db"
)

// Setting outbound up takes five steps. Each is done when the data it makes
// exists, and nothing else records it: a step can't say done once its data
// is gone, nor not done while the data is there.
const (
	SetupBusiness     = "business"       // a profile that says what the business sells and to whom
	SetupTargeting    = "targeting"      // a segment that says who it targets
	SetupSending      = "sending"        // outbound on, a mailing address, a domain that passes its DNS check, an active mailbox
	SetupDoNotContact = "do_not_contact" // a do-not-contact list, or an outside check
	SetupCampaign     = "campaign"       // a campaign
)

// SetupStep is one step of setting outbound up: whether it is done and, when
// not, what it still needs.
type SetupStep struct {
	Step  string `json:"step"`
	Done  bool   `json:"done"`
	Needs string `json:"needs,omitempty"`
}

// setupFacts are what the steps are read from.
type setupFacts struct {
	profile         Profile
	definedSegments int
	enabled         bool
	mailingAddress  bool
	verifiedDomain  bool
	anyDomain       bool
	activeMailbox   bool
	dncLists        int
	outsideCheck    bool
	campaigns       int
}

// setupSteps reads the five steps, in order, from the facts.
func setupSteps(f setupFacts) []SetupStep {
	step := func(name string, needs ...string) SetupStep {
		var open []string
		for _, n := range needs {
			if n != "" {
				open = append(open, n)
			}
		}
		return SetupStep{Step: name, Done: len(open) == 0, Needs: strings.Join(open, "; ")}
	}
	when := func(missing bool, need string) string {
		if missing {
			return need
		}
		return ""
	}
	domain := when(!f.anyDomain, "an outreach domain")
	if f.anyDomain && !f.verifiedDomain {
		domain = "a domain that passes its DNS check"
	}
	return []SetupStep{
		step(SetupBusiness, when(!f.profile.Described(), "what the business sells and who buys it")),
		step(SetupTargeting, when(f.definedSegments == 0, "a segment that says who it targets")),
		step(SetupSending,
			when(!f.enabled, "outbound turned on"),
			when(!f.mailingAddress, "a mailing address"),
			domain,
			when(!f.activeMailbox, "an active mailbox")),
		step(SetupDoNotContact, when(f.dncLists == 0 && !f.outsideCheck, "a do-not-contact list, such as your customers")),
		step(SetupCampaign, when(f.campaigns == 0, "a campaign")),
	}
}

// Setup reads where the brand is in setting outbound up.
func (s *Service) Setup(ctx context.Context, org db.Organization) ([]SetupStep, error) {
	f := setupFacts{
		enabled:        s.Enabled(org),
		mailingAddress: strings.TrimSpace(org.MailingAddress) != "",
		outsideCheck:   org.DncCheckUrl != "",
	}
	var err error
	if f.profile, err = s.Profile(ctx, org); err != nil {
		return nil, err
	}
	lists, err := s.db.ListOutreachLists(ctx, org.ID)
	if err != nil {
		return nil, err
	}
	f.definedSegments = definedSegments(lists)
	domains, err := s.db.ListOutreachDomains(ctx, org.ID)
	if err != nil {
		return nil, err
	}
	for _, d := range domains {
		f.anyDomain = true
		f.verifiedDomain = f.verifiedDomain || d.VerificationStatus.String == "success"
	}
	mbs, err := s.db.ListMailboxes(ctx, org.ID)
	if err != nil {
		return nil, err
	}
	for _, mb := range mbs {
		f.activeMailbox = f.activeMailbox || mb.Status == "active"
	}
	dnc, err := s.db.ListDoNotContactLists(ctx, org.ID)
	if err != nil {
		return nil, err
	}
	f.dncLists = len(dnc)
	seqs, err := s.db.ListOutreachSequences(ctx, sql.NullString{String: org.ID, Valid: true})
	if err != nil {
		return nil, err
	}
	f.campaigns = len(seqs)
	return setupSteps(f), nil
}

// definedSegments counts the outreach lists that say who they target.
func definedSegments(lists []db.EmailList) int {
	n := 0
	for _, l := range lists {
		if strings.TrimSpace(l.SegmentDescription) != "" {
			n++
		}
	}
	return n
}
