package outbound

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/outlet-sh/outlet/internal/db"

	"github.com/google/uuid"
)

// Targeting is who outreach is for and what it says: the brand profile, its
// segments and its materials. Outlet writes no email and reads no website;
// the AI calling it does both, and keeps what it learns here.

// ---- the brand profile ------------------------------------------------------

// Profile is what the business sells, to whom, its proof and how its emails
// sound: what an AI writing the brand's outreach starts from.
type Profile struct {
	Website   string `json:"website"`
	Sells     string `json:"sells"`    // what the business sells
	Audience  string `json:"audience"` // who buys it
	Proof     string `json:"proof"`    // the short proof: customers, results, numbers
	Voice     string `json:"voice"`    // how its emails sound
	UpdatedAt string `json:"updated_at,omitempty"`
}

// Empty says nothing is known about the business yet.
func (p Profile) Empty() bool { return p.Website == "" && p.Sells == "" && p.Audience == "" }

// Profile reads the brand's profile; empty when none was saved.
func (s *Service) Profile(ctx context.Context, org db.Organization) (Profile, error) {
	p, err := s.db.GetBrandProfile(ctx, org.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return Profile{}, nil
	}
	if err != nil {
		return Profile{}, err
	}
	return Profile{Website: p.Website, Sells: p.Sells, Audience: p.Audience, Proof: p.Proof, Voice: p.Voice, UpdatedAt: p.UpdatedAt}, nil
}

// UpdateProfile changes the fields given; an empty one keeps what is saved.
func (s *Service) UpdateProfile(ctx context.Context, org db.Organization, in Profile) (Profile, error) {
	cur, err := s.Profile(ctx, org)
	if err != nil {
		return cur, err
	}
	keep := func(v, old string) string {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
		return old
	}
	p, err := s.db.UpsertBrandProfile(ctx, db.UpsertBrandProfileParams{
		OrgID: org.ID, Website: keep(in.Website, cur.Website), Sells: keep(in.Sells, cur.Sells),
		Audience: keep(in.Audience, cur.Audience), Proof: keep(in.Proof, cur.Proof), Voice: keep(in.Voice, cur.Voice),
	})
	if err != nil {
		return cur, err
	}
	return Profile{Website: p.Website, Sells: p.Sells, Audience: p.Audience, Proof: p.Proof, Voice: p.Voice, UpdatedAt: p.UpdatedAt}, nil
}

// ---- segments ---------------------------------------------------------------

// SegmentFilters say who a segment targets. They guide whoever finds the
// prospects and are checked when prospects are added (a mismatch is noted,
// never refused). Values match without regard to case.
type SegmentFilters struct {
	Countries     []string          `json:"countries,omitempty"`
	Regions       []string          `json:"regions,omitempty"`
	Industries    []string          `json:"industries,omitempty"`
	CompanySizes  []string          `json:"company_sizes,omitempty"`  // employee ranges: 1-10, 11-50, 51-200, 201-1000, 1000+
	TitleKeywords []string          `json:"title_keywords,omitempty"` // a title matches when it holds any of them
	Fields        map[string]string `json:"fields,omitempty"`         // custom field values, key to value
}

// SegmentInput creates or changes a segment. On a change, an empty name or
// description keeps the saved one, and filters left out keep theirs.
type SegmentInput struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Filters     *SegmentFilters `json:"filters,omitempty"`
}

// SegmentResults is how a segment's campaigns are doing, summed.
type SegmentResults struct {
	Campaigns  int64   `json:"campaigns"`
	Reached    int64   `json:"reached"`
	Replied    int64   `json:"replied"`
	ReplyRate  float64 `json:"reply_rate"`
	Interested int64   `json:"interested"`
}

// Segment is one segment: an outreach list with its definition, its people
// by status and its campaigns' results.
type Segment struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Filters     SegmentFilters   `json:"filters"`
	Prospects   map[string]int64 `json:"prospects"` // by status: prospect, verifying, held, ...
	Results     SegmentResults   `json:"results"`
}

const maxFilterValues = 50

func (in *SegmentInput) check() error {
	if utf8.RuneCountInString(in.Description) > 2000 {
		return refuse("A segment's description is a few sentences (at most 2000 characters).")
	}
	if f := in.Filters; f != nil {
		for name, vals := range map[string][]string{"countries": f.Countries, "regions": f.Regions, "industries": f.Industries, "company_sizes": f.CompanySizes, "title_keywords": f.TitleKeywords} {
			if len(vals) > maxFilterValues {
				return refuse("At most %d %s per segment.", maxFilterValues, name)
			}
		}
		for _, size := range f.CompanySizes {
			if _, _, ok := sizeRange(size); !ok {
				return refuse("Company size %q isn't a range: use 1-10, 11-50, 51-200, 201-1000 or 1000+.", size)
			}
		}
		for k := range f.Fields {
			if !fieldKeyRe.MatchString(k) {
				return refuse("Custom field %q isn't a field key (lowercase letters, digits and _).", k)
			}
		}
	}
	return nil
}

// SaveSegment creates a segment (id empty) or changes one. A new segment is
// a new outreach list; naming an outreach list that already exists gives it
// the definition.
func (s *Service) SaveSegment(ctx context.Context, org db.Organization, id string, in SegmentInput) (Segment, error) {
	if err := in.check(); err != nil {
		return Segment{}, err
	}
	var list db.EmailList
	var err error
	if id == "" {
		if strings.TrimSpace(in.Name) == "" {
			return Segment{}, refuse("A segment needs a name, such as \"Dental clinics in Utah\".")
		}
		list, err = s.outreachList(ctx, org, in.Name)
	} else {
		list, err = s.segment(ctx, org, id)
	}
	if err != nil {
		return Segment{}, err
	}
	name, desc, filters := list.Name, list.SegmentDescription, list.SegmentFilters
	if v := strings.TrimSpace(in.Name); v != "" && id != "" && !strings.EqualFold(v, list.Name) {
		if other, err := s.segment(ctx, org, v); err == nil && other.ID != list.ID {
			return Segment{}, refuse("Another segment is already called %q.", v)
		}
		name = v
	}
	if v := strings.TrimSpace(in.Description); v != "" {
		desc = v
	}
	if in.Filters != nil {
		raw, err := json.Marshal(cleanFilters(*in.Filters))
		if err != nil {
			return Segment{}, err
		}
		filters = string(raw)
	}
	if err := s.db.SetSegmentDefinition(ctx, db.SetSegmentDefinitionParams{Name: name, SegmentDescription: desc, SegmentFilters: filters, ID: list.ID, OrgID: org.ID}); err != nil {
		return Segment{}, err
	}
	all, err := s.Segments(ctx, org)
	if err != nil {
		return Segment{}, err
	}
	for _, sg := range all {
		if sg.ID == strconv.FormatInt(list.ID, 10) {
			return sg, nil
		}
	}
	return Segment{}, refuse("no such segment")
}

func cleanFilters(f SegmentFilters) SegmentFilters {
	clean := func(vals []string) []string {
		var out []string
		for _, v := range vals {
			if v = strings.TrimSpace(v); v != "" && !slices.ContainsFunc(out, func(o string) bool { return strings.EqualFold(o, v) }) {
				out = append(out, v)
			}
		}
		return out
	}
	out := SegmentFilters{Countries: clean(f.Countries), Regions: clean(f.Regions), Industries: clean(f.Industries),
		CompanySizes: clean(f.CompanySizes), TitleKeywords: clean(f.TitleKeywords)}
	for k, v := range f.Fields {
		if v = strings.TrimSpace(v); v != "" {
			if out.Fields == nil {
				out.Fields = map[string]string{}
			}
			out.Fields[k] = v
		}
	}
	return out
}

// segment finds one of the brand's outreach lists by id or by name.
func (s *Service) segment(ctx context.Context, org db.Organization, ref string) (db.EmailList, error) {
	ref = strings.TrimSpace(ref)
	lists, err := s.db.ListOutreachLists(ctx, org.ID)
	if err != nil {
		return db.EmailList{}, err
	}
	for _, l := range lists {
		if strconv.FormatInt(l.ID, 10) == ref || strings.EqualFold(l.Name, ref) {
			return l, nil
		}
	}
	return db.EmailList{}, refuse("No segment %q. targeting(resource: segment, action: list) shows the brand's segments.", ref)
}

func filtersOf(l db.EmailList) SegmentFilters {
	var f SegmentFilters
	_ = json.Unmarshal([]byte(l.SegmentFilters), &f)
	return f
}

// Segments lists the brand's segments (every outreach list), with their
// people by status and their campaigns' results.
func (s *Service) Segments(ctx context.Context, org db.Organization) ([]Segment, error) {
	seqs, err := s.db.ListOutreachSequences(ctx, sql.NullString{String: org.ID, Valid: true})
	if err != nil {
		return nil, err
	}
	camps := make([]CampaignResults, 0, len(seqs))
	for _, c := range seqs {
		r, err := s.CampaignResults(ctx, c, false)
		if err != nil {
			return nil, err
		}
		camps = append(camps, r)
	}
	return s.segmentsOf(ctx, org, camps)
}

// segmentsOf lists the segments with the results of the campaigns given,
// summed by the segment each runs on.
func (s *Service) segmentsOf(ctx context.Context, org db.Organization, camps []CampaignResults) ([]Segment, error) {
	lists, err := s.db.ListOutreachLists(ctx, org.ID)
	if err != nil {
		return nil, err
	}
	counts, err := s.db.CountSegmentProspects(ctx, org.ID)
	if err != nil {
		return nil, err
	}
	results := map[string]SegmentResults{}
	for _, r := range camps {
		sr := results[r.SegmentID]
		sr.Campaigns++
		sr.Reached += r.Reached
		sr.Replied += r.Replied
		sr.Interested += r.Interested
		sr.ReplyRate = pct(sr.Replied, sr.Reached)
		results[r.SegmentID] = sr
	}
	out := make([]Segment, 0, len(lists))
	for _, l := range lists {
		id := strconv.FormatInt(l.ID, 10)
		sg := Segment{ID: id, Name: l.Name, Description: l.SegmentDescription, Filters: filtersOf(l),
			Prospects: map[string]int64{}, Results: results[id]}
		for _, c := range counts {
			if c.ListID == l.ID {
				sg.Prospects[c.Status.String] = c.N
			}
		}
		out = append(out, sg)
	}
	return out, nil
}

// AddToSegment puts prospects already on another outreach list into a
// segment too, with their details. Their status there carries over: a
// verified prospect is a prospect, a held address stays held.
func (s *Service) AddToSegment(ctx context.Context, org db.Organization, segmentRef string, prospectIDs []string) ([]ProspectResult, error) {
	if len(prospectIDs) == 0 || len(prospectIDs) > maxBatch {
		return nil, refuse("Pick 1 to %d prospects.", maxBatch)
	}
	list, err := s.segment(ctx, org, segmentRef)
	if err != nil {
		return nil, err
	}
	out := make([]ProspectResult, 0, len(prospectIDs))
	for _, id := range prospectIDs {
		res := ProspectResult{ID: id}
		c, err := s.db.GetContactByID(ctx, id)
		if err != nil || c.OrgID.String != org.ID {
			res.Status, res.Reason = "refused", "no such prospect in this brand"
			out = append(out, res)
			continue
		}
		res.Email = c.Email
		from, err := s.db.GetProspectOutreachSubscription(ctx, db.GetProspectOutreachSubscriptionParams{OrgID: org.ID, ContactID: c.ID})
		if err != nil {
			res.Status, res.Reason = "refused", "not a prospect: add them with add_prospects"
			out = append(out, res)
			continue
		}
		sub, err := s.db.CreateListSubscriber(ctx, db.CreateListSubscriberParams{ID: uuid.NewString(), ListID: list.ID, ContactID: c.ID, Status: from.Status})
		if errors.Is(err, sql.ErrNoRows) {
			res.Status, res.Reason = "duplicate", "already in "+list.Name
			out = append(out, res)
			continue
		}
		if err != nil {
			res.Status, res.Reason = "refused", err.Error()
			out = append(out, res)
			continue
		}
		fields := s.seq.GetCustomFieldsForContact(ctx, c.ID, from.ListID)
		for k, v := range fields {
			if err := s.setField(ctx, list.ID, sub.ID, k, v); err != nil {
				res.Status, res.Reason = "refused", err.Error()
				break
			}
		}
		if res.Status == "" {
			res.Status, res.Reason = from.Status.String, "added to "+list.Name
			if miss := filtersOf(list).mismatch(fields); miss != "" {
				res.Reason += "; " + miss
			}
		}
		out = append(out, res)
	}
	return out, nil
}

// mismatch says how a prospect's details fall outside the filters, or "".
// A filter the prospect has no value for doesn't count against them.
func (f SegmentFilters) mismatch(fields map[string]string) string {
	var out []string
	oneOf := func(label, key string, allowed []string) {
		v := strings.TrimSpace(fields[key])
		if v == "" || len(allowed) == 0 || slices.ContainsFunc(allowed, func(a string) bool { return strings.EqualFold(a, v) }) {
			return
		}
		out = append(out, fmt.Sprintf("%s %s is not one of %s", label, v, strings.Join(allowed, ", ")))
	}
	oneOf("country", "country", f.Countries)
	oneOf("region", "region", f.Regions)
	oneOf("industry", "industry", f.Industries)
	if v := strings.TrimSpace(fields["company_size"]); v != "" && len(f.CompanySizes) > 0 && !sizeFits(v, f.CompanySizes) {
		out = append(out, fmt.Sprintf("company size %s is outside %s", v, strings.Join(f.CompanySizes, ", ")))
	}
	if v := strings.ToLower(fields["title"]); v != "" && len(f.TitleKeywords) > 0 &&
		!slices.ContainsFunc(f.TitleKeywords, func(k string) bool { return strings.Contains(v, strings.ToLower(k)) }) {
		out = append(out, fmt.Sprintf("title %q has none of %s", fields["title"], strings.Join(f.TitleKeywords, ", ")))
	}
	for k, want := range f.Fields {
		if v := strings.TrimSpace(fields[k]); v != "" && !strings.EqualFold(v, want) {
			out = append(out, fmt.Sprintf("%s is %s, not %s", k, v, want))
		}
	}
	if len(out) == 0 {
		return ""
	}
	slices.Sort(out)
	return "outside the segment's filters: " + strings.Join(out, "; ")
}

// sizeRange reads an employee range: "11-50" or "1000+".
func sizeRange(r string) (lo, hi int64, ok bool) {
	r = strings.ReplaceAll(strings.TrimSpace(r), ",", "")
	if v, found := strings.CutSuffix(r, "+"); found {
		n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return n, 1 << 62, err == nil
	}
	a, b, found := strings.Cut(r, "-")
	if !found {
		return 0, 0, false
	}
	lo, err1 := strconv.ParseInt(strings.TrimSpace(a), 10, 64)
	hi, err2 := strconv.ParseInt(strings.TrimSpace(b), 10, 64)
	return lo, hi, err1 == nil && err2 == nil && lo <= hi
}

// sizeFits says whether a company size (a head count, or a range as the
// filters write them) falls in one of the ranges.
func sizeFits(size string, ranges []string) bool {
	for _, r := range ranges {
		if strings.EqualFold(strings.TrimSpace(size), strings.TrimSpace(r)) {
			return true
		}
		lo, hi, ok := sizeRange(r)
		if n, err := strconv.ParseInt(strings.ReplaceAll(strings.TrimSpace(size), ",", ""), 10, 64); ok && err == nil && n >= lo && n <= hi {
			return true
		}
	}
	return false
}

// ---- materials --------------------------------------------------------------

// MaterialKinds are the kinds of material a brand keeps for outreach.
var MaterialKinds = []string{"case_study", "proof_point", "offer", "link", "asset"}

// MaterialInput creates or changes a material. On a change, an empty field
// keeps the saved one. Segment names the one segment it is for (id or
// name); "any" makes it brand-wide again.
type MaterialInput struct {
	Kind    string `json:"kind"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Details string `json:"details,omitempty"`
	URL     string `json:"url,omitempty"`
	Segment string `json:"segment,omitempty"`
}

// Material is one case study, proof point, offer, link or asset.
type Material struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	Summary   string `json:"summary"`
	Details   string `json:"details,omitempty"`
	URL       string `json:"url,omitempty"`
	SegmentID string `json:"segment_id,omitempty"` // empty: for any segment
	Segment   string `json:"segment,omitempty"`
	UpdatedAt string `json:"updated_at"`
}

const maxSummary = 600

func (s *Service) materialOf(ctx context.Context, m db.BrandMaterial) Material {
	out := Material{ID: m.ID, Kind: m.Kind, Title: m.Title, Summary: m.Summary, Details: m.Details, URL: m.Url, UpdatedAt: m.UpdatedAt}
	if m.ListID.Valid {
		out.SegmentID = strconv.FormatInt(m.ListID.Int64, 10)
		if l, err := s.db.GetEmailList(ctx, m.ListID.Int64); err == nil {
			out.Segment = l.Name
		}
	}
	return out
}

// SaveMaterial creates a material (id empty) or changes one.
func (s *Service) SaveMaterial(ctx context.Context, org db.Organization, id string, in MaterialInput) (Material, error) {
	cur := db.BrandMaterial{}
	if id != "" {
		m, err := s.db.GetBrandMaterial(ctx, db.GetBrandMaterialParams{ID: id, OrgID: org.ID})
		if err != nil {
			return Material{}, refuse("no such material")
		}
		cur = m
	}
	pick := func(v, old string) string {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
		return old
	}
	kind, title, summary := pick(in.Kind, cur.Kind), pick(in.Title, cur.Title), pick(in.Summary, cur.Summary)
	details, url := pick(in.Details, cur.Details), pick(in.URL, cur.Url)
	switch {
	case !slices.Contains(MaterialKinds, kind):
		return Material{}, refuse("A material's kind is case_study, proof_point, offer, link or asset.")
	case title == "":
		return Material{}, refuse("A material needs a title.")
	case summary == "":
		return Material{}, refuse("A material needs a short summary: what it shows, in a sentence or two.")
	case utf8.RuneCountInString(summary) > maxSummary:
		return Material{}, refuse("The summary is the short version (at most %d characters); put the rest in details.", maxSummary)
	case url != "" && !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://"):
		return Material{}, refuse("url must start with https://.")
	}
	listID := cur.ListID
	switch seg := strings.TrimSpace(in.Segment); {
	case strings.EqualFold(seg, "any"):
		listID = sql.NullInt64{}
	case seg != "":
		l, err := s.segment(ctx, org, seg)
		if err != nil {
			return Material{}, err
		}
		listID = sql.NullInt64{Int64: l.ID, Valid: true}
	}
	var m db.BrandMaterial
	var err error
	if id == "" {
		m, err = s.db.CreateBrandMaterial(ctx, db.CreateBrandMaterialParams{ID: uuid.NewString(), OrgID: org.ID, Kind: kind, Title: title, Summary: summary, Details: details, Url: url, ListID: listID})
	} else {
		if err = s.db.UpdateBrandMaterial(ctx, db.UpdateBrandMaterialParams{Kind: kind, Title: title, Summary: summary, Details: details, Url: url, ListID: listID, ID: id, OrgID: org.ID}); err == nil {
			m, err = s.db.GetBrandMaterial(ctx, db.GetBrandMaterialParams{ID: id, OrgID: org.ID})
		}
	}
	if err != nil {
		return Material{}, err
	}
	return s.materialOf(ctx, m), nil
}

// DeleteMaterial removes a material; campaigns that pinned it lose the pin.
func (s *Service) DeleteMaterial(ctx context.Context, org db.Organization, id string) error {
	n, err := s.db.DeleteBrandMaterial(ctx, db.DeleteBrandMaterialParams{ID: id, OrgID: org.ID})
	if err == nil && n == 0 {
		return refuse("no such material")
	}
	return err
}

// Materials lists the brand's materials of one kind (empty: any), for one
// segment (empty: all of them): that segment's own and the brand-wide ones.
func (s *Service) Materials(ctx context.Context, org db.Organization, kind, segmentRef string) ([]Material, error) {
	var listID int64
	if segmentRef != "" {
		l, err := s.segment(ctx, org, segmentRef)
		if err != nil {
			return nil, err
		}
		listID = l.ID
	}
	rows, err := s.db.ListBrandMaterials(ctx, db.ListBrandMaterialsParams{OrgID: org.ID, Kind: strings.TrimSpace(kind), ListID: listID})
	if err != nil {
		return nil, err
	}
	out := make([]Material, 0, len(rows))
	for _, m := range rows {
		out = append(out, s.materialOf(ctx, m))
	}
	return out, nil
}

// campaignMaterials are the materials a campaign was written from.
func (s *Service) campaignMaterials(ctx context.Context, campaignID string) []Material {
	rows, err := s.db.ListCampaignMaterials(ctx, campaignID)
	if err != nil {
		return nil
	}
	out := make([]Material, 0, len(rows))
	for _, m := range rows {
		out = append(out, s.materialOf(ctx, m))
	}
	return out
}
