package tools

import (
	"context"
	"strings"

	"github.com/outlet-sh/outlet/internal/mcp/mcpctx"
	"github.com/outlet-sh/outlet/internal/services/outbound"
)

// ---- targeting --------------------------------------------------------------

// TargetingInput is the targeting tool's call: who outreach is for (the
// brand profile and its segments), what it can cite (materials), and who it
// never writes to (do not contact).
type TargetingInput struct {
	Resource    string                     `json:"resource" jsonschema:"required,profile, segment, material or do_not_contact"`
	Action      string                     `json:"action" jsonschema:"required,profile: get, update. segment: list, get, create, update, add. material: list, create, update, delete. do_not_contact: list, add, delete, check, settings"`
	ID          string                     `json:"id,omitempty" jsonschema:"segment get, update and add: the segment's id or name; material update and delete: the material's id"`
	Profile     *outbound.Profile          `json:"profile,omitempty" jsonschema:"profile update: {website, sells (what the business sells), audience (who buys it), proof (customers, results, numbers), voice (how its emails sound)}; fields left out are kept"`
	Segment     *outbound.SegmentInput     `json:"segment,omitempty" jsonschema:"segment create and update: {name, description (who it targets, in plain words), filters: {countries, regions, industries, company_sizes (1-10, 11-50, 51-200, 201-1000, 1000+), title_keywords, fields (custom field values)}}"`
	ProspectIDs []string                   `json:"prospect_ids,omitempty" jsonschema:"segment add: up to 100 prospects already on another segment, to put in this one too"`
	Material    *outbound.MaterialInput    `json:"material,omitempty" jsonschema:"material create and update: {kind (case_study, proof_point, offer, link or asset), title, summary (a sentence or two), details, url, segment (the one segment it is for, by name; leave out for brand-wide; any makes it brand-wide again)}"`
	Kind        string                     `json:"kind,omitempty" jsonschema:"material list: one kind only"`
	List        string                     `json:"list,omitempty" jsonschema:"material list: a segment's own and the brand-wide materials (by segment name or id)"`
	DNCList     string                     `json:"dnc_list,omitempty" jsonschema:"do_not_contact add and delete: the do-not-contact list's name, such as Customers or Open deals"`
	Entries     []string                   `json:"entries,omitempty" jsonschema:"do_not_contact add: up to 10000 email addresses or domains (a website counts as its domain)"`
	Email       string                     `json:"email,omitempty" jsonschema:"do_not_contact check: the address to check against every source"`
	Settings    *outbound.DNCSettingsInput `json:"settings,omitempty" jsonschema:"do_not_contact settings: {customer_domains (also skip everyone at a customer's company domain), check_url (an outside yes/no check; empty turns it off), check_secret (signs its requests; write-only)}. Leave settings out to read them"`
}

func targetingRun(ctx context.Context, t *mcpctx.ToolContext, in TargetingInput) (any, error) {
	o, brand := t.Svc().Outbound, t.Brand(ctx)
	need := func(ok bool, field, msg string) error {
		if ok {
			return nil
		}
		return mcpctx.NewValidationError(msg, field)
	}
	switch in.Resource + "." + in.Action {
	case "profile.get":
		return o.Profile(ctx, brand)
	case "profile.update":
		if err := need(in.Profile != nil, "profile", "profile is required: {website, sells, audience, proof, voice}"); err != nil {
			return nil, err
		}
		p, err := o.UpdateProfile(ctx, brand, *in.Profile)
		if err != nil {
			return nil, err
		}
		return map[string]any{"profile": p, "next": "Propose segments from it: targeting(resource: segment, action: create, segment: {name, description, filters})."}, nil

	case "segment.list":
		segs, err := o.Segments(ctx, brand)
		return map[string]any{"segments": segs}, err
	case "segment.get":
		if err := need(in.ID != "", "id", "id is required: the segment's id or name"); err != nil {
			return nil, err
		}
		segs, err := o.Segments(ctx, brand)
		if err != nil {
			return nil, err
		}
		for _, sg := range segs {
			if sg.ID == in.ID || strings.EqualFold(sg.Name, in.ID) {
				return sg, nil
			}
		}
		return nil, mcpctx.NewNotFoundError("no such segment")
	case "segment.create", "segment.update":
		if err := need(in.Segment != nil, "segment", "segment is required: {name, description, filters}"); err != nil {
			return nil, err
		}
		id := ""
		if in.Action == "update" {
			if err := need(in.ID != "", "id", "id is required: the segment's id or name"); err != nil {
				return nil, err
			}
			id = in.ID
		}
		sg, err := o.SaveSegment(ctx, brand, id, *in.Segment)
		if err != nil {
			return nil, err
		}
		return map[string]any{"segment": sg, "next": "Add its prospects with add_prospects(action: add, list: \"" + sg.Name + "\", prospects)."}, nil
	case "segment.add":
		if err := need(in.ID != "", "id", "id is required: the segment's id or name"); err != nil {
			return nil, err
		}
		res, err := o.AddToSegment(ctx, brand, in.ID, in.ProspectIDs)
		return map[string]any{"prospects": res}, err

	case "material.list":
		m, err := o.Materials(ctx, brand, in.Kind, in.List)
		return map[string]any{"materials": m}, err
	case "material.create", "material.update":
		if err := need(in.Material != nil, "material", "material is required: {kind, title, summary, details, url, segment}"); err != nil {
			return nil, err
		}
		id := ""
		if in.Action == "update" {
			if err := need(in.ID != "", "id", "id is required: the material's id"); err != nil {
				return nil, err
			}
			id = in.ID
		}
		return o.SaveMaterial(ctx, brand, id, *in.Material)
	case "material.delete":
		if err := need(in.ID != "", "id", "id is required: the material's id"); err != nil {
			return nil, err
		}
		if err := o.DeleteMaterial(ctx, brand, in.ID); err != nil {
			return nil, err
		}
		return map[string]string{"result": "deleted"}, nil

	case "do_not_contact.list":
		lists, err := o.DoNotContactLists(ctx, brand)
		return map[string]any{"lists": lists, "settings": outbound.DNCSettingsOf(brand)}, err
	case "do_not_contact.add":
		return o.AddDoNotContact(ctx, brand, in.DNCList, in.Entries)
	case "do_not_contact.delete":
		if err := need(in.DNCList != "", "dnc_list", "dnc_list is required: the list to delete"); err != nil {
			return nil, err
		}
		n, err := o.DeleteDoNotContactList(ctx, brand, in.DNCList)
		if err != nil {
			return nil, err
		}
		return map[string]any{"removed": n}, nil
	case "do_not_contact.check":
		if err := need(in.Email != "", "email", "email is required"); err != nil {
			return nil, err
		}
		ex, excluded := o.DoNotContact(ctx, brand, in.Email)
		out := map[string]any{"email": in.Email, "excluded": excluded}
		if excluded {
			out["source"], out["reason"] = ex.Source, ex.Reason
		}
		return out, nil
	case "do_not_contact.settings":
		if in.Settings == nil {
			return outbound.DNCSettingsOf(brand), nil
		}
		return o.UpdateDNCSettings(ctx, brand, *in.Settings)
	}
	return nil, mcpctx.NewValidationError("resource and action must be one of: profile get or update; segment list, get, create, update or add; material list, create, update or delete; do_not_contact list, add, delete, check or settings", "action")
}

const targetingDesc = `Who outreach is for, what it can cite, and who it never writes to. Outlet doesn't browse or write emails: you read the business's website and write; Outlet keeps what you learn. explain(action: get, topic: start_from_website) gives the whole flow.
- profile: get, update {website, sells, audience, proof, voice}. What the business sells, to whom, its proof and how it sounds: fill it in from the website.
- segment: who a campaign targets. A segment is an outreach list with a definition: name, description (plain words) and filters {countries, regions, industries, company_sizes, title_keywords, fields}. list (with prospects by status and results), get (id), create (segment), update (id, segment), add (id, prospect_ids: put prospects already on another segment in this one). Add new prospects into a segment with add_prospects(list: <segment name>); draft_outreach(list: <segment name>) with no prospect_ids enrolls them.
- material: case studies, proof points, offers, links and assets, each with a short summary, brand-wide or for one segment. list (kind, list), create (material), update (id, material), delete (id). draft_outreach(action: brief) hands them over to write from; pin the ones a campaign uses with material_ids.
- do_not_contact: people outreach never writes to. Before any outreach is queued, everyone is checked against: suppression, the brand's own domains, its customers (anyone on a newsletter list or sent transactional mail; with customer_domains, anyone at their company's domain), uploaded do-not-contact lists, and the optional outside check (check_url: Outlet POSTs {email, domain, brand, timestamp} signed with X-Outlet-Signature, HMAC-SHA256 of the body with check_secret, and reads {"do_not_contact": true|false, "reason"}; while it doesn't answer nobody new is enrolled and emails already in a campaign wait, nothing is sent). list, add (dnc_list, entries), delete (dnc_list), check (email: which source excludes it, if any), settings (settings; brand admins only to change).

Examples: targeting(resource: profile, action: update, profile: {website: "https://acme.com", sells: "...", audience: "..."}); targeting(resource: segment, action: create, segment: {name: "Dental clinics in Utah", description: "...", filters: {countries: ["US"], regions: ["UT"], industries: ["dental"]}}); targeting(resource: do_not_contact, action: add, dnc_list: "Customers", entries: ["ann@client.com", "client2.com"])`
