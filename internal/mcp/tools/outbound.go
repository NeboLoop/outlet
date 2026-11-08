package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/mcp/mcpctx"
	"github.com/outlet-sh/outlet/internal/services/outbound"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The outbound tools are shaped around the outreach job: eight tools, each a
// whole step in one call. Read, write and send are separate (scopes
// outbound:read, outbound:write and outbound:send), so a client can look
// freely and ask a person before anything goes out. The limits live in the
// outbound service; no argument here can raise them.

const outboundPrereq = "PREREQUISITE: select the brand first with brand(resource: brand, action: select)."

// outboundActions lists each outreach tool's actions; the gate has a rule for each.
var outboundActions = map[string][]string{
	"outreach_overview": {"get"},
	"setup_sending":     {"settings", "dns_token", "add_domain", "check_domain", "add_mailboxes", "list"},
	"add_prospects":     {"add"},
	"draft_outreach":    {"create"},
	"review_drafts":     {"list", "approve", "edit", "skip"},
	"conversations":     {"list", "reply", "classify", "mark_partner"},
	"pause":             {"pause", "resume"},
	"explain":           {"get"},
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
	Action string `json:"action" jsonschema:"required,get"`
}

func overviewRun(ctx context.Context, t *mcpctx.ToolContext, _ OverviewInput) (any, error) {
	return t.Svc().Outbound.Overview(ctx, t.Brand(ctx))
}

// ---- setup_sending ----------------------------------------------------------

type SetupSendingInput struct {
	Action         string                  `json:"action" jsonschema:"required,One of: settings, dns_token, add_domain, check_domain, add_mailboxes, list"`
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
	DomainID       string                  `json:"domain_id,omitempty" jsonschema:"check_domain: which domain to recheck"`
	Mailboxes      []outbound.MailboxInput `json:"mailboxes,omitempty" jsonschema:"add_mailboxes: up to 100 mailboxes; each must log in to SMTP and IMAP"`
}

func setupSendingRun(ctx context.Context, t *mcpctx.ToolContext, in SetupSendingInput) (any, error) {
	o, brand := t.Svc().Outbound, t.Brand(ctx)
	switch in.Action {
	case "settings":
		st, err := o.UpdateSettings(ctx, brand, outbound.SettingsInput{MailingAddress: in.MailingAddress, OptOutLine: in.OptOutLine, Timezone: in.Timezone})
		if err != nil {
			return nil, err
		}
		return map[string]any{"mailing_address": st.MailingAddress, "opt_out_line": st.OptOutLine, "timezone": st.Timezone, "dns_token_set": len(st.DnsTokenEnc) > 0}, nil
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
		d, pending, check, err := o.AddDomain(ctx, brand, outbound.DomainInput{
			Domain: in.Domain, DNSHost: in.DNSHost, MailProvider: in.MailProvider, MXHosts: in.MXHosts,
			SPFInclude: in.SPFInclude, DKIMSelector: in.DKIMSelector, DKIMValue: in.DKIMValue, DMARCReport: in.DMARCReport,
		})
		if err != nil {
			return map[string]any{"records_to_add": pending}, err
		}
		next := "DNS is complete. Connect mailboxes with setup_sending(action: add_mailboxes)."
		if !check.OK() {
			next = "Missing in public DNS: " + fmt.Sprint(check.Missing) + ". Add records_to_add (DKIM comes from the mail provider's admin console), then check_domain. New records can take minutes to appear."
		}
		return map[string]any{"domain_id": d.ID, "status": d.Status, "check": check, "records_to_add": pending, "next": next}, nil
	case "check_domain":
		d, err := t.DB().GetOutboundDomain(ctx, db.GetOutboundDomainParams{ID: in.DomainID, OrgID: brand.ID})
		if err != nil {
			return nil, mcpctx.NewNotFoundError("no such domain")
		}
		return o.CheckDomain(ctx, d)
	case "add_mailboxes":
		res, err := o.AddMailboxes(ctx, brand, in.Mailboxes)
		if err != nil {
			return nil, err
		}
		return map[string]any{"mailboxes": res, "next": "New mailboxes start at 5 emails a day and add 2 a day up to 30. Use each one normally for about two weeks before outreach."}, nil
	case "list":
		mbs, err := t.DB().ListOutboundMailboxes(ctx, brand.ID)
		if err != nil {
			return nil, err
		}
		out := make([]map[string]any, 0, len(mbs))
		for _, mb := range mbs {
			out = append(out, map[string]any{"id": mb.ID, "email": mb.Email, "from_name": mb.FromName, "status": mb.Status, "paused_reason": mb.PausedReason, "daily_cap": mb.DailyCap, "warmup_started_at": mb.WarmupStartedAt})
		}
		return map[string]any{"mailboxes": out}, nil
	}
	return nil, mcpctx.NewValidationError("action must be settings, dns_token, add_domain, check_domain, add_mailboxes or list", "action")
}

// ---- add_prospects ----------------------------------------------------------

type AddProspectsInput struct {
	Action    string                   `json:"action" jsonschema:"required,add"`
	Prospects []outbound.ProspectInput `json:"prospects" jsonschema:"required,Up to 100 people. email_source (where the address was published for business) is required for each"`
}

func addProspectsRun(ctx context.Context, t *mcpctx.ToolContext, in AddProspectsInput) (any, error) {
	res, err := t.Svc().Outbound.AddProspects(ctx, t.Brand(ctx), in.Prospects)
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
	Action   string                 `json:"action" jsonschema:"required,create"`
	Campaign outbound.CampaignInput `json:"campaign" jsonschema:"required,name, steps (subject and body with {{first_name}}-style tags and {a|b} variants), prospect_ids, optional mailbox_ids, approval_mode, window hours, idempotency_key, dry_run"`
}

func draftOutreachRun(ctx context.Context, t *mcpctx.ToolContext, in DraftOutreachInput) (any, error) {
	return t.Svc().Outbound.DraftOutreach(ctx, t.Brand(ctx), in.Campaign)
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
		if err := o.EditDraft(ctx, brand, in.ID, in.Subject, in.Body); err != nil {
			return nil, err
		}
		return map[string]string{"result": "edited"}, nil
	}
	return nil, mcpctx.NewValidationError("action must be list, approve, edit or skip", "action")
}

// ---- conversations ----------------------------------------------------------

type ConversationsInput struct {
	Action         string `json:"action" jsonschema:"required,One of: list, reply, classify, mark_partner"`
	Handled        bool   `json:"handled,omitempty" jsonschema:"list: true lists replies already dealt with"`
	Limit          int    `json:"limit,omitempty" jsonschema:"list: how many (default 25)"`
	ReplyID        string `json:"reply_id,omitempty" jsonschema:"reply or classify: which reply"`
	Body           string `json:"body,omitempty" jsonschema:"reply: what to write back (same mailbox, same thread)"`
	Classification string `json:"classification,omitempty" jsonschema:"classify: interested, not_interested, question, out_of_office, wrong_person, unsubscribe or complaint"`
	ProspectID     string `json:"prospect_id,omitempty" jsonschema:"mark_partner: the creator who said yes"`
}

func conversationsRun(ctx context.Context, t *mcpctx.ToolContext, in ConversationsInput) (any, error) {
	o, brand := t.Svc().Outbound, t.Brand(ctx)
	switch in.Action {
	case "list":
		r, err := o.ListReplies(ctx, brand, in.Handled, in.Limit)
		return map[string]any{"replies": r}, err
	case "reply":
		id, err := o.Answer(ctx, brand, in.ReplyID, in.Body)
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
	return nil, mcpctx.NewValidationError("action must be list, reply, classify or mark_partner", "action")
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
}

func explainRun(ctx context.Context, t *mcpctx.ToolContext, in ExplainInput) (any, error) {
	s, err := t.Svc().Outbound.Explain(ctx, t.Brand(ctx), in.MessageID, in.MailboxID, in.ProspectID)
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
		"Outreach at a glance: whether outbound is on, domains and their DNS, each mailbox's sends today against its limit, bounces and complaints, drafts and replies waiting, anything paused and why, and what to do next.\n\nExample: outreach_overview(action: get)",
		readOnly(), overviewRun),
	defineOutbound("setup_sending", "Set up sending",
		"Set up where outreach sends from. Outreach uses separate outreach domains, never the brand's own or a protected company domain, and never SES.\n"+
			"- settings: mailing_address (required before any campaign), opt_out_line, timezone\n"+
			"- dns_token: store a DigitalOcean token (domain read/write) so add_domain can write DNS\n"+
			"- add_domain: domain, dns_host (digitalocean|manual), mail_provider (google|other), mx_hosts, spf_include, dkim_selector, dkim_value, dmarc_report. Writes MX, SPF, DMARC and DKIM when on DigitalOcean, otherwise returns records_to_add\n"+
			"- check_domain: domain_id. Rechecks public DNS (also done daily; failing DNS pauses the domain's mailboxes)\n"+
			"- add_mailboxes: mailboxes [{email, from_name, signature, provider: google, password (an app password)} or custom smtp_/imap_ hosts]. Each must log in\n"+
			"- list: the brand's mailboxes",
		changesOnly(), setupSendingRun),
	defineOutbound("add_prospects", "Add prospects",
		"Import up to 100 people to reach, with creator fields (platform, channel_url, audience_size, niche, recent_video, personal_line) and email_source, where the address was published for business. Removes duplicates, checks suppression and verifies every address; risky ones are held and never sent.\n\nExample: add_prospects(action: add, prospects: [{email, first_name, platform: youtube, channel_url, recent_video, personal_line, email_source: \"YouTube About page\"}])",
		changesOnly(), addProspectsRun),
	defineOutbound("draft_outreach", "Draft outreach",
		"Create a campaign of 1 to 5 steps and draft every first email for its prospects. Templates use {{first_name}}, {{name}}, {{channel_url}}, {{recent_video}}, {{personal_line}}, {{sender_first}} and custom fields, and {a|b} variants. A missing value skips that prospect rather than sending a blank. Each prospect gets one mailbox for the whole sequence, and follow-ups reply in the same thread. The opt-out line and mailing address are added. dry_run renders without saving. Nothing sends until approved (approval_mode every by default).",
		changesOnly(), draftOutreachRun),
	defineOutbound("review_drafts", "Review drafts",
		"List emails waiting for approval, then approve, edit or skip them by id (up to 100 at a time). Approving lets them send, paced per mailbox inside the sending window.",
		sends(), reviewDraftsRun),
	defineOutbound("conversations", "Conversations",
		"Replies to outreach. Any reply stops that person's sequence; opt-outs and bounces go on the suppression list. list shows replies needing attention; classify records what one means; reply answers in the same thread from the same mailbox; mark_partner records a creator who said yes.",
		sends(), conversationsRun),
	defineOutbound("pause", "Pause outreach",
		"Pause or resume all outreach, one mailbox or one campaign. A mailbox paused for DNS or health resumes only once the problem is gone.",
		changesOnly(), pauseRun),
	defineOutbound("explain", "Explain",
		"Plain answers to \"why didn't this send?\" (message_id), \"why is this mailbox paused?\" (mailbox_id) and \"why can't this prospect be enrolled?\" (prospect_id).",
		readOnly(), explainRun),
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
