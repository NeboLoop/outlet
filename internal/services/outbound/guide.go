package outbound

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/outlet-sh/outlet/internal/db"
)

// Brief is what an AI writes a campaign from: the brand, the segment, the
// merge tags its prospects have, the materials to draw on, and the rules
// cold email needs. Outlet writes nothing itself.
type Brief struct {
	Profile    Profile    `json:"profile"`
	Segment    Segment    `json:"segment"`
	Enrollable int64      `json:"enrollable"` // prospects a campaign on the segment can take now
	MergeTags  []string   `json:"merge_tags"` // {{tag}}s every step may use
	Materials  []Material `json:"materials"`  // the pinned ones, else the segment's and the brand-wide ones
	Rules      []string   `json:"rules"`
	Missing    []string   `json:"missing,omitempty"` // what the brand still lacks before it can send
	Next       string     `json:"next"`
}

// draftRules are the rules a written campaign is checked against.
var draftRules = []string{
	"1 to 5 steps; 3 over about 12 days works best. Every step after the first waits at least 1 day (delay_days).",
	"The first email needs a subject and may hold at most one link. A later step with no subject replies in the same thread.",
	"Plain text, short, about the prospect: name something true about them from their fields, then one clear ask.",
	"Use {a|b} variants so no two emails are identical. A merge tag a prospect has no value for skips that prospect rather than sending a blank.",
	"The opt-out line and mailing address are added to every email: don't write them.",
	"Use only claims the materials or the profile support.",
}

// baseTags are the merge tags every prospect has.
var baseTags = []string{"name", "first_name", "email", "sender_name", "sender_first"}

// Brief reads what writing a campaign for a segment starts from. With
// materialIDs, those are the materials; else the segment's and the
// brand-wide ones.
func (s *Service) Brief(ctx context.Context, org db.Organization, segmentRef string, materialIDs []string) (Brief, error) {
	if strings.TrimSpace(segmentRef) == "" {
		segmentRef = DefaultOutreachList
	}
	list, err := s.segment(ctx, org, segmentRef)
	if err != nil {
		return Brief{}, err
	}
	b := Brief{Rules: draftRules, MergeTags: append([]string{}, baseTags...)}
	if b.Profile, err = s.Profile(ctx, org); err != nil {
		return b, err
	}
	segs, err := s.Segments(ctx, org)
	if err != nil {
		return b, err
	}
	for _, sg := range segs {
		if sg.ID == fmt.Sprint(list.ID) {
			b.Segment = sg
		}
	}
	page, err := s.ListProspects(ctx, org, ProspectQuery{ListID: fmt.Sprint(list.ID), Enrollable: true, Limit: 1})
	if err != nil {
		return b, err
	}
	b.Enrollable = page.Total
	if fields, err := s.db.ListCustomFieldsByList(ctx, list.ID); err == nil {
		for _, f := range fields {
			if !slices.Contains(b.MergeTags, f.FieldKey) {
				b.MergeTags = append(b.MergeTags, f.FieldKey)
			}
		}
	}
	if len(materialIDs) > 0 {
		b.Materials, err = s.pinnable(ctx, org, materialIDs)
	} else {
		b.Materials, err = s.Materials(ctx, org, "", fmt.Sprint(list.ID))
	}
	if err != nil {
		return b, err
	}
	b.Missing = s.missing(ctx, org, b.Profile)
	b.Next = fmt.Sprintf("Write the steps from this, then draft_outreach(action: create, campaign: {name, list: %q, steps, material_ids}) with the ids of the materials used. With no prospect_ids it enrolls the segment's prospects (up to 100). Try dry_run first to read the first emails.", list.Name)
	if b.Enrollable == 0 {
		b.Next = fmt.Sprintf("Segment %q has no prospect free to enroll yet: add_prospects(list: %q) first. ", list.Name, list.Name) + b.Next
	}
	return b, nil
}

// missing is what the brand still lacks before outreach can go.
func (s *Service) missing(ctx context.Context, org db.Organization, p Profile) []string {
	var out []string
	if !s.Enabled(org) {
		out = append(out, "outbound is off: the brand's admin turns it on in the Outlet app, Outbound, Setup")
	}
	if p.Empty() {
		out = append(out, "a brand profile: targeting(resource: profile, action: update)")
	}
	if strings.TrimSpace(org.MailingAddress) == "" {
		out = append(out, "a mailing address: setup_sending(action: settings, mailing_address)")
	}
	if mbs, err := s.db.ListMailboxes(ctx, org.ID); err != nil || !slices.ContainsFunc(mbs, func(m db.Mailbox) bool { return m.Status == "active" }) {
		out = append(out, "an active mailbox to send from: setup_sending")
	}
	return out
}

// startFromWebsite is the playbook for starting outreach from a business's
// website, in the order of the calls.
const startFromWebsite = `Starting outreach from a website. Outlet doesn't browse or write: you read the site, and Outlet keeps what you learn.
1. Read the website (home, pricing, customers or case studies, about).
2. targeting(resource: profile, action: update, profile: {website, sells, audience, proof, voice}): what the business sells, to whom, its proof, how it sounds.
3. Propose 2 to 4 segments to the person: who buys, in plain words, with filters (countries, regions, industries, company_sizes, title_keywords). On their yes, targeting(resource: segment, action: create, segment: {name, description, filters}) for each.
4. targeting(resource: material, action: create, material: {kind, title, summary, url, segment}) for each case study, proof point, offer or link worth citing (kind: case_study, proof_point, offer, link or asset; segment: empty for brand-wide).
5. Find prospects outside Outlet, then add_prospects(action: add, list: <segment name>, prospects: [{email, first_name, company, title, country, industry, company_size, personal_line, email_source}]). Anyone the brand already knows (customers, do-not-contact lists, its own domains) is excluded and says why.
6. draft_outreach(action: brief, list: <segment name>): the profile, segment, merge tags, materials and rules to write from.
7. Write the steps, then draft_outreach(action: create, campaign: {name, list, steps, material_ids, dry_run: true}) to read the first emails, and again without dry_run to save them. They wait for approval in review_drafts.`

// explainTopic answers a how-to question, with what this brand still lacks.
func (s *Service) explainTopic(ctx context.Context, org db.Organization, topic string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(topic)) {
	case "start_from_website", "website":
		p, err := s.Profile(ctx, org)
		if err != nil {
			return "", err
		}
		var have []string
		if !p.Empty() {
			have = append(have, "a profile ("+firstNonEmpty(p.Website, p.Sells)+")")
		}
		lists, _ := s.db.ListOutreachLists(ctx, org.ID)
		if defined := definedSegments(lists); defined > 0 {
			have = append(have, fmt.Sprintf("%d defined segments", defined))
		}
		if mats, err := s.db.ListBrandMaterials(ctx, db.ListBrandMaterialsParams{OrgID: org.ID}); err == nil && len(mats) > 0 {
			have = append(have, fmt.Sprintf("%d materials", len(mats)))
		}
		out := startFromWebsite
		if len(have) > 0 {
			out += "\nThis brand already has " + strings.Join(have, ", ") + "."
		}
		if miss := s.missing(ctx, org, p); len(miss) > 0 {
			out += "\nStill missing before anything can send: " + strings.Join(miss, "; ") + "."
		}
		return out, nil
	}
	return "", refuse("Topics: start_from_website.")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
