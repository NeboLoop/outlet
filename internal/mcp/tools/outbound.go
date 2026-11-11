package tools

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/mcp/mcpctx"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/services/outbound"
	"github.com/outlet-sh/outlet/internal/services/outbound/providers"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The outbound tools are shaped around the outreach job: nine tools, each a
// whole step in one call. Read, write and send are separate (scopes
// outbound:read, outbound:write and outbound:send), so a client can look
// freely and ask a person before anything goes out. The limits live in the
// outbound service; no argument here can raise them.

const outboundPrereq = "PREREQUISITE: select the brand first with brand(resource: brand, action: select)."

// outboundActions lists each outreach tool's actions; the gate has a rule for each.
var outboundActions = map[string][]string{
	"outreach_overview": {"get"},
	"setup_sending":     {"settings", "dns_token", "add_domain", "check_domain", "add_mailboxes", "list", "set_limits", "mailbox_limit", "pace", "set_pace", "connections", "add_connection", "update_connection", "remove_connection", "move_domain", "service_inventory", "connect_domain", "quote", "buy", "sync", "delete_mailbox"},
	"add_prospects":     {"add"},
	"draft_outreach":    {"brief", "create"},
	"review_drafts":     {"list", "approve", "edit", "skip"},
	"conversations":     {"list", "reply", "classify", "mark_partner", "export"},
	"targeting": {
		"profile.get", "profile.update",
		"segment.list", "segment.get", "segment.create", "segment.update", "segment.add", "segment.delete",
		"material.list", "material.create", "material.update", "material.delete",
		"do_not_contact.list", "do_not_contact.add", "do_not_contact.delete", "do_not_contact.check", "do_not_contact.settings",
	},
	"pause":   {"pause", "resume"},
	"explain": {"get"},
}

// outboundErr turns the service's refusals into tool errors the caller can act on.
func outboundErr(err error) error {
	var r *outbound.Refusal
	if errors.As(err, &r) {
		return &mcpctx.ToolError{Code: "refused", Message: r.Msg}
	}
	return err
}

type outboundFunc[In any] func(ctx context.Context, toolCtx *mcpctx.ToolContext, in In) (any, error)

func outboundHandler[In any](toolCtx *mcpctx.ToolContext, fn outboundFunc[In]) mcp.ToolHandlerFor[In, any] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in In) (*mcp.CallToolResult, any, error) {
		if err := toolCtx.RequireBrand(ctx); err != nil {
			return nil, nil, err
		}
		out, err := fn(ctx, toolCtx, in)
		if err != nil {
			return nil, nil, outboundErr(err)
		}
		return nil, out, nil
	}
}

func readOnly() *mcp.ToolAnnotations    { return &mcp.ToolAnnotations{ReadOnlyHint: true} }
func sends() *mcp.ToolAnnotations       { t := true; return &mcp.ToolAnnotations{DestructiveHint: &t} }
func changesOnly() *mcp.ToolAnnotations { f := false; return &mcp.ToolAnnotations{DestructiveHint: &f} }

// ---- outreach_overview ----------------------------------------------------

type OverviewInput struct {
	Action     string `json:"action" jsonschema:"required,get"`
	CampaignID string `json:"campaign_id,omitempty" jsonschema:"one campaign's results, step by step: sent, replies, reply rate and clicks per step"`
}

func overviewRun(ctx context.Context, t *mcpctx.ToolContext, in OverviewInput) (any, error) {
	if in.CampaignID != "" {
		brand := t.Brand(ctx)
		c, err := t.DB().GetOutreachSequence(ctx, db.GetOutreachSequenceParams{ID: in.CampaignID, OrgID: sql.NullString{String: brand.ID, Valid: true}})
		if err != nil {
			return nil, mcpctx.NewNotFoundError("no such campaign")
		}
		return t.Svc().Outbound.CampaignResults(ctx, c, true)
	}
	return t.Svc().Outbound.Overview(ctx, t.Brand(ctx))
}

// ---- setup_sending ----------------------------------------------------------

type SetupSendingInput struct {
	Action         string                  `json:"action" jsonschema:"required,One of: settings, dns_token, add_domain, check_domain, add_mailboxes, list, set_limits, mailbox_limit, pace, set_pace, connections, add_connection, update_connection, remove_connection, move_domain, service_inventory, connect_domain, quote, buy, sync, delete_mailbox"`
	MailingAddress string                  `json:"mailing_address,omitempty" jsonschema:"settings: the postal address every first email carries"`
	OptOutLine     string                  `json:"opt_out_line,omitempty" jsonschema:"settings: the line that tells people how to stop the emails"`
	Timezone       string                  `json:"timezone,omitempty" jsonschema:"settings: IANA time zone used when a prospect has none"`
	Token          string                  `json:"token,omitempty" jsonschema:"dns_token: a DigitalOcean API token limited to domain read and write"`
	Domain         string                  `json:"domain,omitempty" jsonschema:"add_domain: the outreach domain, e.g. getexample.com"`
	DNSHost        string                  `json:"dns_host,omitempty" jsonschema:"add_domain: digitalocean (writes the records) or manual (returns them to paste; default)"`
	MailProvider   string                  `json:"mail_provider,omitempty" jsonschema:"add_domain: google (default) or other"`
	MXHosts        []string                `json:"mx_hosts,omitempty" jsonschema:"add_domain: MX hosts, for mail_provider other"`
	SPFInclude     string                  `json:"spf_include,omitempty" jsonschema:"add_domain: SPF include, for mail_provider other"`
	DKIMSelector   string                  `json:"dkim_selector,omitempty" jsonschema:"add_domain: DKIM selector (default google)"`
	DKIMValue      string                  `json:"dkim_value,omitempty" jsonschema:"add_domain: the DKIM TXT value from the mail provider's admin console"`
	DMARCReport    string                  `json:"dmarc_report,omitempty" jsonschema:"add_domain: address for DMARC reports (optional)"`
	ClickTracking  bool                    `json:"click_tracking,omitempty" jsonschema:"add_domain: set up go.<domain> (a CNAME to this server) so campaigns can track clicks"`
	DomainID       string                  `json:"domain_id,omitempty" jsonschema:"check_domain: which domain to recheck; move_domain: the outreach domain to move"`
	Mailboxes      []outbound.MailboxInput `json:"mailboxes,omitempty" jsonschema:"add_mailboxes: up to 100 mailboxes; each must log in to SMTP and IMAP"`
	PoolID         string                  `json:"pool_id,omitempty" jsonschema:"set_limits: the mailbox pool to change; connections and add_connection: the pool (default: the pool carrying outreach)"`
	Limits         *email.Limits           `json:"limits,omitempty" jsonschema:"set_limits: the pool's limits, all of them: daily_limit (per mailbox per day), warmup_start, warmup_step, min_gap_seconds, max_gap_seconds, pause_bounce_rate (0.03 is 3%), pause_min_bounces, pause_spam_blocks, pause_complaints. 0 turns one off"`
	MailboxID      string                  `json:"mailbox_id,omitempty" jsonschema:"mailbox_limit: which mailbox"`
	Pace           string                  `json:"pace,omitempty" jsonschema:"set_pace: light (25 new prospects a day), typical (50), growth (100) or unlimited"`
	NewPerDay      *int64                  `json:"daily_new_prospects,omitempty" jsonschema:"set_pace: new prospects a day instead of a preset (0: no limit)"`
	DailyLimit     *int64                  `json:"daily_limit,omitempty" jsonschema:"mailbox_limit: the mailbox's own daily limit, leave it out to use its connection's or pool's; add_connection and update_connection: the daily limit of the connection's mailboxes unless one has its own (leave it out for the pool's)"`
	WarmupStart    *int64                  `json:"warmup_start,omitempty" jsonschema:"add_connection and update_connection: the warm-up's first-day allowance for the connection's mailboxes (leave it out for the pool's)"`
	WarmupStep     *int64                  `json:"warmup_step,omitempty" jsonschema:"add_connection and update_connection: added to the allowance each day until the daily limit (leave it out for the pool's)"`
	ConnectionID   string                  `json:"connection_id,omitempty" jsonschema:"the mailbox service connection: update_connection, remove_connection, move_domain (where the domain goes), service_inventory, connect_domain, quote, buy, sync, delete_mailbox; add_domain: a manual connection (default: the pool's first manual one)"`
	Service        string                  `json:"service,omitempty" jsonschema:"add_connection: manual, maildoso, cheapinboxes or infraforge"`
	Name           string                  `json:"name,omitempty" jsonschema:"add_connection and update_connection: the connection's name, e.g. Maildoso main (default: the service's name)"`
	APIKey         string                  `json:"api_key,omitempty" jsonschema:"add_connection and update_connection: the service's API key (write-only, tested before it is saved; update: leave it out to keep the stored one)"`
	Kind           string                  `json:"kind,omitempty" jsonschema:"quote: domain (register a new one), connect (a domain the brand owns, where the service bills for it) or mailboxes"`
	People         []providers.Person      `json:"people,omitempty" jsonschema:"quote with kind mailboxes: up to 100 mailboxes, each first_name, last_name and optional local (before the @; default first.last)"`
	QuoteID        string                  `json:"quote_id,omitempty" jsonschema:"buy: the quote to buy"`
	Confirm        bool                    `json:"confirm,omitempty" jsonschema:"buy and delete_mailbox: must be true; buy spends money"`
	Import         bool                    `json:"import,omitempty" jsonschema:"sync: also import the service's mailboxes the pool lacks"`
}

func setupSendingRun(ctx context.Context, t *mcpctx.ToolContext, in SetupSendingInput) (any, error) {
	o, brand := t.Svc().Outbound, t.Brand(ctx)
	switch in.Action {
	case "settings":
		st, err := o.UpdateSettings(ctx, brand, outbound.SettingsInput{MailingAddress: in.MailingAddress, OptOutLine: in.OptOutLine, Timezone: in.Timezone})
		if err != nil {
			return nil, err
		}
		return map[string]any{"mailing_address": st.MailingAddress, "opt_out_line": st.OutboundOptOutLine, "timezone": st.OutboundTimezone, "dns_token_set": len(st.DnsTokenEnc) > 0}, nil
	case "dns_token":
		if in.Token == "" {
			return nil, mcpctx.NewValidationError("token is required", "token")
		}
		if err := o.SetDNSToken(ctx, brand.ID, in.Token); err != nil {
			return nil, err
		}
		return map[string]string{"result": "DigitalOcean token saved, encrypted."}, nil
	case "add_domain":
		if in.Domain == "" {
			return nil, mcpctx.NewValidationError("domain is required", "domain")
		}
		res, err := o.AddDomain(ctx, brand, outbound.DomainInput{
			Domain: in.Domain, DNSHost: in.DNSHost, MailProvider: in.MailProvider, MXHosts: in.MXHosts,
			SPFInclude: in.SPFInclude, DKIMSelector: in.DKIMSelector, DKIMValue: in.DKIMValue, DMARCReport: in.DMARCReport, ClickTracking: in.ClickTracking,
			ConnectionID: in.ConnectionID,
		})
		if err != nil {
			return nil, err
		}
		next := "DNS is complete. Connect mailboxes with setup_sending(action: add_mailboxes)."
		if !res.Check.OK() {
			next = "Missing in public DNS: " + fmt.Sprint(res.Check.Missing) + ". Add records_to_add (DKIM comes from the mail provider's admin console), then check_domain. New records can take minutes to appear."
		}
		out := map[string]any{"domain_id": res.Domain.ID, "status": res.Domain.VerificationStatus.String, "tracking_host": res.Domain.TrackingHost, "check": res.Check, "records_to_add": res.RecordsToAdd, "next": next}
		if res.DNSError != "" {
			out["dns_error"] = res.DNSError
		}
		return out, nil
	case "check_domain":
		d, err := t.DB().GetOutreachDomain(ctx, db.GetOutreachDomainParams{ID: in.DomainID, OrgID: brand.ID})
		if err != nil {
			return nil, mcpctx.NewNotFoundError("no such domain")
		}
		if _, err := o.CheckDomain(ctx, d); err != nil {
			return nil, err
		}
		if d, err = t.DB().GetOutreachDomain(ctx, db.GetOutreachDomainParams{ID: in.DomainID, OrgID: brand.ID}); err != nil {
			return nil, err
		}
		return outbound.DomainInfoOf(d), nil
	case "add_mailboxes":
		res, err := o.AddMailboxes(ctx, brand, in.Mailboxes)
		if err != nil {
			return nil, err
		}
		return map[string]any{"mailboxes": res, "next": "New mailboxes warm up under their pool's limits (setup_sending(action: list) shows them). Use each one normally for about two weeks before outreach."}, nil
	case "list":
		return listPools(ctx, t)
	case "set_limits":
		if in.Limits == nil {
			return nil, mcpctx.NewValidationError("limits is required", "limits")
		}
		pool, err := mailboxPool(ctx, t, in.PoolID)
		if err != nil {
			return nil, err
		}
		if _, err := t.Svc().EmailService.UpdatePathway(ctx, brand.ID, pool.ID, email.PathwayInput{Kinds: email.KindsOf(pool), Limits: in.Limits}); err != nil {
			return nil, err
		}
		return listPools(ctx, t)
	case "pace":
		return o.Pace(ctx, brand)
	case "set_pace":
		p, err := o.SetPace(ctx, brand, in.Pace, in.NewPerDay)
		if err != nil {
			return nil, err
		}
		return map[string]any{"pace": p, "next": "First emails beyond the day's number wait for the next day's window; follow-ups go on. When the day's number is reached, webhooks get outreach.paused once."}, nil
	case "mailbox_limit":
		if in.MailboxID == "" {
			return nil, mcpctx.NewValidationError("mailbox_id is required", "mailbox_id")
		}
		if err := t.Svc().EmailService.SetMailboxDailyLimit(ctx, brand.ID, in.MailboxID, in.DailyLimit); err != nil {
			return nil, err
		}
		return listPools(ctx, t)
	}
	if out, handled, err := mailboxServiceRun(ctx, t, in); handled {
		return out, err
	}
	return nil, mcpctx.NewValidationError("action must be settings, dns_token, add_domain, check_domain, add_mailboxes, list, set_limits, mailbox_limit, pace, set_pace, connections, add_connection, update_connection, remove_connection, move_domain, service_inventory, connect_domain, quote, buy, sync or delete_mailbox", "action")
}

// connectionLimits are a connection's limits from the input.
func (in SetupSendingInput) connectionLimits() outbound.ConnectionLimits {
	return outbound.ConnectionLimits{DailyLimit: in.DailyLimit, WarmupStart: in.WarmupStart, WarmupStep: in.WarmupStep}
}

// mailboxServiceRun runs the actions on a pool's mailbox service
// connections.
func mailboxServiceRun(ctx context.Context, t *mcpctx.ToolContext, in SetupSendingInput) (out any, handled bool, err error) {
	o, brand := t.Svc().Outbound, t.Brand(ctx)
	switch in.Action {
	case "connections":
		poolID := ""
		if in.PoolID != "" {
			pool, err := mailboxPool(ctx, t, in.PoolID)
			if err != nil {
				return nil, true, err
			}
			poolID = pool.ID
		}
		list, err := o.ListConnections(ctx, brand, poolID)
		if err != nil {
			return nil, true, err
		}
		return map[string]any{"connections": list, "next": "Each connection's domains and mailboxes send together from the pool. Add one with add_connection; act on one with its connection_id."}, true, nil
	case "add_connection":
		if in.Service == "" {
			return nil, true, mcpctx.NewValidationError("service is required: manual, maildoso, cheapinboxes or infraforge", "service")
		}
		c, err := o.AddConnection(ctx, brand, outbound.ConnectionInput{PoolID: in.PoolID, Service: in.Service, Name: in.Name, APIKey: in.APIKey, ConnectionLimits: in.connectionLimits()})
		if err != nil {
			return nil, true, err
		}
		next := "Connect a domain with connect_domain, or quote one with quote(kind: domain), passing this connection_id."
		if c.Service == providers.Manual {
			next = "Add a domain with add_domain(connection_id), then its mailboxes with add_mailboxes."
		}
		return map[string]any{"connection": c, "next": next}, true, nil
	case "update_connection":
		c, err := o.UpdateConnection(ctx, brand, in.ConnectionID, outbound.ConnectionInput{Name: in.Name, APIKey: in.APIKey, ConnectionLimits: in.connectionLimits()})
		return c, true, err
	case "remove_connection":
		if err := o.RemoveConnection(ctx, brand, in.ConnectionID); err != nil {
			return nil, true, err
		}
		return map[string]string{"result": "Connection removed. Nothing changed at the service."}, true, nil
	case "move_domain":
		if in.DomainID == "" {
			return nil, true, mcpctx.NewValidationError("domain_id is required", "domain_id")
		}
		c, err := o.MoveDomainToConnection(ctx, brand, in.DomainID, in.ConnectionID)
		return c, true, err
	case "service_inventory":
		out, err = o.MailboxServiceInventory(ctx, brand, in.ConnectionID)
	case "connect_domain":
		if in.Domain == "" {
			return nil, true, mcpctx.NewValidationError("domain is required", "domain")
		}
		out, err = o.ConnectServiceDomain(ctx, brand, in.ConnectionID, in.Domain)
	case "quote":
		q, err := o.QuoteService(ctx, brand, in.ConnectionID, outbound.QuoteInput{Kind: in.Kind, Domain: in.Domain, People: in.People})
		if err != nil {
			return nil, true, err
		}
		return map[string]any{"quote": q, "next": "Show the person the summary and price. Buying spends money: only on their yes, call setup_sending(action: buy, connection_id, quote_id, confirm: true) before it expires."}, true, nil
	case "buy":
		if in.QuoteID == "" {
			return nil, true, mcpctx.NewValidationError("quote_id is required: get a quote first", "quote_id")
		}
		if !in.Confirm {
			return nil, true, mcpctx.NewValidationError("confirm=true is required: buying spends money. Show the person the quote first.", "confirm")
		}
		out, err = o.BuyService(ctx, brand, in.ConnectionID, in.QuoteID, in.Confirm)
	case "sync":
		out, err = o.SyncService(ctx, brand, in.ConnectionID, in.Import)
	case "delete_mailbox":
		if in.MailboxID == "" {
			return nil, true, mcpctx.NewValidationError("mailbox_id is required", "mailbox_id")
		}
		if !in.Confirm {
			return nil, true, mcpctx.NewValidationError("confirm=true is required: the service deletes the mailbox (Maildoso and Infraforge with its messages) or cancels it", "confirm")
		}
		if err := o.DeleteServiceMailbox(ctx, brand, in.ConnectionID, in.MailboxID); err != nil {
			return nil, true, err
		}
		return map[string]string{"result": "Deleted at the mailbox service; it stays in the pool, paused."}, true, nil
	default:
		return nil, false, nil
	}
	return out, true, err
}

// mailboxPool is the brand's mailbox pool named by id, or the one carrying
// its outreach, or its only one.
func mailboxPool(ctx context.Context, t *mcpctx.ToolContext, id string) (db.SendingPathway, error) {
	brand := t.Brand(ctx)
	if id != "" {
		p, err := t.Svc().EmailService.GetPathway(ctx, brand.ID, id)
		if err == nil && p.Provider != email.ProviderMailboxes {
			return p, mcpctx.NewValidationError("pool_id is not a mailbox pool", "pool_id")
		}
		return p, err
	}
	all, err := t.DB().ListSendingPathways(ctx, brand.ID)
	if err != nil {
		return db.SendingPathway{}, err
	}
	var pools []db.SendingPathway
	for _, p := range all {
		if p.Provider == email.ProviderMailboxes {
			if p.CarriesOutreach == 1 {
				return p, nil
			}
			pools = append(pools, p)
		}
	}
	if len(pools) == 1 {
		return pools[0], nil
	}
	return db.SendingPathway{}, mcpctx.NewValidationError("pass pool_id: the brand has no single mailbox pool", "pool_id")
}

// listPools is the brand's mailbox pools with their limits, and each
// mailbox with its own limit and today's sends.
func listPools(ctx context.Context, t *mcpctx.ToolContext) (any, error) {
	brand := t.Brand(ctx)
	all, err := t.DB().ListSendingPathways(ctx, brand.ID)
	if err != nil {
		return nil, err
	}
	overview, err := t.Svc().Outbound.Overview(ctx, brand)
	if err != nil {
		return nil, err
	}
	pools := []map[string]any{}
	for _, p := range all {
		if p.Provider != email.ProviderMailboxes {
			continue
		}
		mbs := []outbound.MailboxHealth{}
		for _, mb := range overview.Mailboxes {
			if mb.PoolID == p.ID {
				mbs = append(mbs, mb)
			}
		}
		pools = append(pools, map[string]any{"id": p.ID, "name": p.Name, "kinds": email.KindsOf(p), "limits": email.LimitsOf(p), "mailboxes": mbs})
	}
	return map[string]any{"pools": pools, "next": "Change a pool's limits with setup_sending(action: set_limits, limits), one mailbox's with setup_sending(action: mailbox_limit, mailbox_id, daily_limit)."}, nil
}

// ---- add_prospects ----------------------------------------------------------

type AddProspectsInput struct {
	Action    string                   `json:"action" jsonschema:"required,add"`
	List      string                   `json:"list,omitempty" jsonschema:"The segment (an outreach list) to add them to, by name (made if missing; default Outreach)"`
	Prospects []outbound.ProspectInput `json:"prospects" jsonschema:"required,Up to 100 people. email_source (where the address was published for business) is required for each"`
}

func addProspectsRun(ctx context.Context, t *mcpctx.ToolContext, in AddProspectsInput) (any, error) {
	res, err := t.Svc().Outbound.AddProspects(ctx, t.Brand(ctx), in.List, in.Prospects)
	if err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for _, r := range res {
		counts[r.Status]++
	}
	return map[string]any{"counts": counts, "prospects": res}, nil
}

// ---- draft_outreach ---------------------------------------------------------

type DraftOutreachInput struct {
	Action      string                 `json:"action" jsonschema:"required,brief or create"`
	List        string                 `json:"list,omitempty" jsonschema:"brief: the segment to write for, by name or id (default Outreach)"`
	MaterialIDs []string               `json:"material_ids,omitempty" jsonschema:"brief: only these materials (default: the segment's and the brand-wide ones)"`
	Campaign    outbound.CampaignInput `json:"campaign,omitempty" jsonschema:"create: name, list (the segment), steps [{subject, body, delay_days}] with {{first_name}}-style tags and {a|b} variants, prospect_ids (leave out to enroll the segment's prospects, up to 100), material_ids (the materials it was written from), approval_mode, window_start_hour, window_end_hour, track_opens, track_clicks, idempotency_key, dry_run"`
}

func draftOutreachRun(ctx context.Context, t *mcpctx.ToolContext, in DraftOutreachInput) (any, error) {
	switch in.Action {
	case "brief":
		return t.Svc().Outbound.Brief(ctx, t.Brand(ctx), in.List, in.MaterialIDs)
	case "create":
		in.Campaign.AuthoredBy = outbound.AuthorAI // whatever calls MCP is an AI
		return t.Svc().Outbound.DraftOutreach(ctx, t.Brand(ctx), in.Campaign)
	}
	return nil, mcpctx.NewValidationError("action must be brief or create", "action")
}

// ---- review_drafts ----------------------------------------------------------

type ReviewDraftsInput struct {
	Action  string   `json:"action" jsonschema:"required,One of: list, approve, edit, skip"`
	IDs     []string `json:"ids,omitempty" jsonschema:"approve or skip: up to 100 draft ids"`
	ID      string   `json:"id,omitempty" jsonschema:"edit: one draft id"`
	Subject string   `json:"subject,omitempty" jsonschema:"edit: new subject"`
	Body    string   `json:"body,omitempty" jsonschema:"edit: new body"`
	Limit   int      `json:"limit,omitempty" jsonschema:"list: how many (default 25, max 100)"`
}

func reviewDraftsRun(ctx context.Context, t *mcpctx.ToolContext, in ReviewDraftsInput) (any, error) {
	o, brand := t.Svc().Outbound, t.Brand(ctx)
	switch in.Action {
	case "list":
		d, err := o.ListDrafts(ctx, brand, in.Limit)
		return map[string]any{"drafts": d}, err
	case "approve", "skip":
		return o.Review(ctx, brand, in.Action, in.IDs)
	case "edit":
		if err := o.EditDraft(ctx, brand, in.ID, in.Subject, in.Body, outbound.AuthorAI); err != nil {
			return nil, err
		}
		return map[string]string{"result": "edited"}, nil
	}
	return nil, mcpctx.NewValidationError("action must be list, approve, edit or skip", "action")
}

// ---- conversations ----------------------------------------------------------

type ConversationsInput struct {
	Action         string `json:"action" jsonschema:"required,One of: list, reply, classify, mark_partner, export"`
	Handled        bool   `json:"handled,omitempty" jsonschema:"list: true lists replies already dealt with"`
	Limit          int    `json:"limit,omitempty" jsonschema:"list: how many (default 25); export: rows (default 500, max 5000)"`
	CampaignID     string `json:"campaign_id,omitempty" jsonschema:"export: one campaign's leads (default: every campaign)"`
	Cursor         string `json:"cursor,omitempty" jsonschema:"export: the next_cursor of the last export, to continue it"`
	ReplyID        string `json:"reply_id,omitempty" jsonschema:"reply or classify: which reply"`
	Body           string `json:"body,omitempty" jsonschema:"reply: what to write back (same mailbox, same thread)"`
	Classification string `json:"classification,omitempty" jsonschema:"classify: interested, not_interested, question, out_of_office, wrong_person, unsubscribe or complaint; list and export: only replies of this kind (interested is a hot lead)"`
	ProspectID     string `json:"prospect_id,omitempty" jsonschema:"mark_partner: the creator who said yes"`
}

func conversationsRun(ctx context.Context, t *mcpctx.ToolContext, in ConversationsInput) (any, error) {
	o, brand := t.Svc().Outbound, t.Brand(ctx)
	switch in.Action {
	case "list":
		r, err := o.ListReplies(ctx, brand, in.Handled, in.Classification, in.Limit)
		return map[string]any{"replies": r}, err
	case "export":
		limit := in.Limit
		if limit <= 0 {
			limit = 500
		}
		limit = min(limit, 5000)
		var buf bytes.Buffer
		rows, next, err := o.WriteLeads(ctx, brand, &buf, outbound.LeadQuery{CampaignID: in.CampaignID, Classification: in.Classification, Limit: limit, After: in.Cursor})
		if err != nil {
			return nil, err
		}
		out := map[string]any{"rows": rows, "csv": buf.String()}
		if next != "" {
			out["next_cursor"] = next
		}
		return out, nil
	case "reply":
		id, err := o.Answer(ctx, brand, in.ReplyID, in.Body, outbound.AuthorAI)
		if err != nil {
			return nil, err
		}
		return map[string]string{"message_id": id, "result": "queued; it goes out from the same mailbox within its daily limit"}, nil
	case "classify":
		if err := o.Classify(ctx, brand, in.ReplyID, in.Classification); err != nil {
			return nil, err
		}
		return map[string]string{"result": "classified"}, nil
	case "mark_partner":
		if err := o.MarkPartner(ctx, brand, in.ProspectID); err != nil {
			return nil, err
		}
		return map[string]string{"result": "marked as partner"}, nil
	}
	return nil, mcpctx.NewValidationError("action must be list, reply, classify, mark_partner or export", "action")
}

// ---- pause ------------------------------------------------------------------

type PauseInput struct {
	Action string `json:"action" jsonschema:"required,pause or resume"`
	Scope  string `json:"scope" jsonschema:"required,all, mailbox or campaign"`
	ID     string `json:"id,omitempty" jsonschema:"the mailbox or campaign id"`
}

func pauseRun(ctx context.Context, t *mcpctx.ToolContext, in PauseInput) (any, error) {
	if in.Action != "pause" && in.Action != "resume" {
		return nil, mcpctx.NewValidationError("action must be pause or resume", "action")
	}
	if err := t.Svc().Outbound.Pause(ctx, t.Brand(ctx), in.Scope, in.ID, in.Action == "resume"); err != nil {
		return nil, err
	}
	return map[string]string{"result": in.Action + "d " + in.Scope}, nil
}

// ---- explain ----------------------------------------------------------------

type ExplainInput struct {
	Action     string `json:"action" jsonschema:"required,get"`
	MessageID  string `json:"message_id,omitempty" jsonschema:"why an email did or didn't send"`
	MailboxID  string `json:"mailbox_id,omitempty" jsonschema:"why a mailbox is paused"`
	ProspectID string `json:"prospect_id,omitempty" jsonschema:"why a prospect can't be enrolled"`
	Topic      string `json:"topic,omitempty" jsonschema:"start_from_website: the calls that go from a business's website to drafted outreach, and what this brand still lacks"`
}

func explainRun(ctx context.Context, t *mcpctx.ToolContext, in ExplainInput) (any, error) {
	s, err := t.Svc().Outbound.Explain(ctx, t.Brand(ctx), outbound.ExplainQuery{MessageID: in.MessageID, MailboxID: in.MailboxID, ProspectID: in.ProspectID, Topic: in.Topic})
	if err != nil {
		return nil, err
	}
	return map[string]string{"answer": s}, nil
}

// ---- registration -----------------------------------------------------------

type outboundTool struct {
	name, title, desc string
	ann               *mcp.ToolAnnotations
	add               func(server *mcp.Server, tool *mcp.Tool, toolCtx *mcpctx.ToolContext)
	reg               func(registry *ToolRegistry, toolCtx *mcpctx.ToolContext)
}

func defineOutbound[In any](name, title, desc string, ann *mcp.ToolAnnotations, fn outboundFunc[In]) outboundTool {
	return outboundTool{
		name: name, title: title, desc: desc + "\n\n" + outboundPrereq, ann: ann,
		add: func(server *mcp.Server, tool *mcp.Tool, toolCtx *mcpctx.ToolContext) {
			mcp.AddTool(server, tool, outboundHandler(toolCtx, fn))
		},
		reg: func(registry *ToolRegistry, toolCtx *mcpctx.ToolContext) {
			registry.Register(name, func(ctx context.Context, args json.RawMessage) (interface{}, error) {
				var in In
				if err := json.Unmarshal(args, &in); err != nil {
					return nil, fmt.Errorf("invalid input: %w", err)
				}
				_, out, err := outboundHandler(toolCtx, fn)(ctx, nil, in)
				return out, err
			})
		},
	}
}

var outboundTools = []outboundTool{
	defineOutbound("outreach_overview", "Outreach overview",
		"Outreach at a glance: whether outbound is on, domains and their DNS, each mailbox's sends today against its limit, bounces and complaints, drafts and replies waiting, anything paused and why, the pace dial and whether today's new prospects are reached, what to do next, every campaign's results (reached, replied, reply rate, interested, partners, clicks, bounces, opt-outs, its segment, who wrote it) and each segment's (its people by status and its campaigns' results). With campaign_id: that campaign step by step, to see which email gets the replies, why people stopped early (stopped_by_reason) and the materials it was written from.\n\nExamples: outreach_overview(action: get); outreach_overview(action: get, campaign_id: \"...\")",
		readOnly(), overviewRun),
	defineOutbound("setup_sending", "Set up sending",
		"Set up where outreach sends from. Outreach uses separate outreach domains, never the brand's own or a protected company domain, and never SES.\n"+
			"- settings: mailing_address (required before any campaign), opt_out_line, timezone\n"+
			"- dns_token: store a DigitalOcean token (domain read/write) so add_domain can write DNS\n"+
			"- add_domain: domain, dns_host (digitalocean|manual), mail_provider (google|other), mx_hosts, spf_include, dkim_selector, dkim_value, dmarc_report, click_tracking. Writes MX, SPF, DMARC and DKIM when on DigitalOcean, otherwise returns records_to_add. When DigitalOcean fails, dns_error says why and records_to_add holds what to add by hand\n"+
			"- check_domain: domain_id. Rechecks public DNS (also done daily; failing DNS pauses the domain's mailboxes) and returns the records the domain needs, each marked present or not\n"+
			"- add_mailboxes: mailboxes [{email, from_name, signature, provider: google, password (an app password), daily_limit (optional, its own)} or custom smtp_/imap_ hosts]. Each must log in\n"+
			"- list: the brand's mailbox pools with their limits (daily limit, warm-up, spacing, health pauses), and each mailbox with its own limit and today's sends\n"+
			"- set_limits: limits (all of them), pool_id (default: the pool carrying outreach). Brand admins only\n"+
			"- mailbox_limit: mailbox_id, daily_limit (leave it out to use its connection's or the pool's). Brand admins only\n"+
			"- pace: the daily pace dial, how many new prospects get their first email each day across the pool, today's count, and what each preset means (sends a day once follow-ups catch up, mailboxes needed at the pool's daily limit, cost where known)\n"+
			"- set_pace: pace (light 25, typical 50, growth 100, unlimited) or daily_new_prospects (0: no limit). At the day's number first emails wait for the next day's window (nothing is lost), follow-ups go on, and webhooks get outreach.paused once that day. Brand admins only\n"+
			"Connections: a pool holds any number of mailbox services side by side (several of one service too), each with its own API key, plus manual connections for mailboxes added by hand (Google Workspace, Microsoft 365). Each outreach domain belongs to one connection; the pool sends from all their mailboxes under its limits, one mailbox per prospect for the whole thread:\n"+
			"- connections: pool_id (optional). Each connection with its service, key status, domains, mailbox count, daily limit and warm-up\n"+
			"- add_connection: service (manual|maildoso|cheapinboxes|infraforge), api_key (required for a service, tested first), name, pool_id, daily_limit, warmup_start, warmup_step (all left out: the service's terms, Maildoso 15 a day warming up from 5 by 1; else the pool's). Brand admins only\n"+
			"- update_connection: connection_id, name, api_key (leave it out to keep it), daily_limit, warmup_start, warmup_step (each set as given; left out: the pool's). Brand admins only\n"+
			"- remove_connection: connection_id. Refused while it holds domains: move them first. Brand admins only\n"+
			"- move_domain: domain_id, connection_id. Moves an outreach domain and its mailboxes to another connection of the same pool. Brand admins only\n"+
			"Each of these takes connection_id, a mailbox service connection:\n"+
			"- service_inventory: the domains and mailboxes the brand has at the service\n"+
			"- connect_domain: domain. Connects a domain the brand owns (free where the service allows) and returns the nameservers to set at its registrar. Brand admins only\n"+
			"- quote: kind (domain|connect|mailboxes), domain, people [{first_name, last_name, local}]. Prices it, or says the service's API gives no price. Buys nothing\n"+
			"- buy: quote_id, confirm: true. Spends money: only after the person has seen the quote and said yes, and only at the connection that gave the quote. Bought mailboxes are imported after their logins are tested. Brand admins only\n"+
			"- sync: import (optional). Lists mailboxes the service has that the pool doesn't, and the connection's mailboxes the service lacks; with import, imports the ready ones. Brand admins only\n"+
			"- delete_mailbox: mailbox_id, confirm: true. Deletes it at the service (CheapInboxes: cancels at the period's end) and pauses it in the pool. Brand admins only",
		changesOnly(), setupSendingRun),
	defineOutbound("add_prospects", "Add prospects",
		"Import up to 100 people to reach into a segment (an outreach list, by name; made if missing). They become contacts that never signed up: campaigns and newsletters never reach them. Company fields (company, title, country, region, industry, company_size), creator fields (platform, channel_url, audience_size, niche, recent_video, personal_line), custom fields, and email_source, where the address was published for business. Every field is a merge tag. Removes duplicates, checks suppression and do-not-contact (anyone the brand already knows: its customers, its own domains, its do-not-contact lists) and says why each was excluded; notes anyone outside the segment's filters. Each address added comes back 'verifying' and is checked within a minute, then becomes a prospect, or held (never sent) when risky. Only prospects can be enrolled.\n\nExample: add_prospects(action: add, list: \"Dental clinics in Utah\", prospects: [{email, first_name, company, title, country: \"US\", region: \"UT\", industry: \"dental\", personal_line, email_source: \"clinic website, contact page\"}])",
		changesOnly(), addProspectsRun),
	defineOutbound("draft_outreach", "Draft outreach",
		"brief: what to write a campaign from, for a segment (list): the brand profile, the segment's definition and how many prospects it can take, the merge tags its prospects have, the materials (the segment's and the brand-wide ones, or material_ids) and the rules. Call it before writing.\n"+
			"create: a campaign of 1 to 5 steps, drafting every first email for its prospects: prospect_ids, or none to enroll the segment's (list) prospects free to enroll, up to 100. Pin the materials it was written from with material_ids. Everyone is checked against do-not-contact before anything is queued, and again before each follow-up; skipped says why for each. Templates use {{first_name}}, {{name}}, {{company}}, {{title}}, {{personal_line}}, {{sender_first}} and any other field the prospects have, and {a|b} variants. A missing value skips that prospect rather than sending a blank. Each prospect gets one mailbox for the whole sequence, and follow-ups reply in the same thread. The opt-out line and mailing address are added. track_clicks and track_opens send links and an open pixel through each domain's go. subdomain (both off by default: reply rate is the main measure, pixels hurt cold delivery, and Apple Mail inflates opens). dry_run returns previews of the first emails without saving anything. Nothing sends until approved (approval_mode every by default). Emails written here are marked as written by an AI, so the reviewer knows what to check.\n\nExamples: draft_outreach(action: brief, list: \"Dental clinics in Utah\"); draft_outreach(action: create, campaign: {name, list: \"Dental clinics in Utah\", steps: [{subject, body}, {delay_days: 4, body}], material_ids: [\"...\"], dry_run: true})",
		changesOnly(), draftOutreachRun),
	defineOutbound("review_drafts", "Review drafts",
		"List emails waiting for approval, each with its message_id, campaign, step and authored_by (ai; ai_edited when a person edited what an AI wrote; person), then approve, edit or skip them by message_id (up to 100 at a time). Approving lets them send, paced per mailbox inside the sending window.",
		sends(), reviewDraftsRun),
	defineOutbound("conversations", "Conversations",
		"Replies to outreach. Any reply stops that person's sequence; opt-outs and bounces go on the suppression list. list shows replies needing attention, each with the campaign it answers and its segment (classification: interested lists the hot leads); classify records what one means; reply answers in the same thread from the same mailbox; mark_partner records a creator who said yes; export gives the leads as CSV, one row per person per campaign (email, name, company, title, segment, campaign, status, stopped_reason, reply_classification, replied_at), narrowed by campaign_id and classification, with next_cursor to continue.",
		sends(), conversationsRun),
	defineOutbound("pause", "Pause outreach",
		"Pause or resume all outreach, one mailbox or one campaign. A mailbox paused for DNS or health resumes only once the problem is gone.",
		changesOnly(), pauseRun),
	defineOutbound("explain", "Explain",
		"Plain answers to \"why didn't this send?\" (message_id), \"why is this mailbox paused?\" (mailbox_id), \"why can't this prospect be enrolled?\" (prospect_id, do-not-contact included) and \"how do I start from a business's website?\" (topic: start_from_website: the calls, in order, from reading the site to drafted outreach, and what this brand still lacks).",
		readOnly(), explainRun),
	defineOutbound("targeting", "Targeting", targetingDesc, changesOnly(), targetingRun),
}

// RegisterOutboundTools adds the outreach tools to the MCP server.
func RegisterOutboundTools(server *mcp.Server, toolCtx *mcpctx.ToolContext) {
	for _, t := range outboundTools {
		t.add(server, &mcp.Tool{Name: t.name, Title: t.title, Description: t.desc, Annotations: t.ann}, toolCtx)
	}
}

func registerOutboundToolsToRegistry(registry *ToolRegistry, toolCtx *mcpctx.ToolContext) {
	for _, t := range outboundTools {
		t.reg(registry, toolCtx)
	}
}
