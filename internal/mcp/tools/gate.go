package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/mcp/mcpauth"
	"github.com/outlet-sh/outlet/internal/mcp/mcpctx"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zeromicro/go-zero/core/logx"
)

// Every tool call passes this gate before its handler runs. The gate checks
// the key's scopes, finds the brand the call acts on, runs access.Check (the
// same check the admin REST API uses), warns before wiping data a live API
// key depends on, and records every change in the MCP audit log. It reads
// the brand once and hands that brand to the handler in the call's context
// (mcpctx.WithCallBrand), so a brand.select running at the same time can't
// move the call to a brand it wasn't checked for.

// brandFrom says where a tool action finds its brand.
type brandFrom int

const (
	sessionBrand brandFrom = iota // the brand this session chose with brand.select
	namedBrand                    // the brand the call names: id, brand_id or slug
	noBrand                       // none: only the caller, or the platform
)

// rule says what one tool action needs.
type rule struct {
	need  access.Role // role needed; empty means any active, signed-in person
	write bool        // a change, recorded in the audit log
	brand brandFrom
	live  bool // warn first when the brand's API key is in live use
}

func read(need access.Role) rule  { return rule{need: need} }
func write(need access.Role) rule { return rule{need: need, write: true} }

// toolPolicy holds one tool's rules, keyed by action, or by resource.action
// for tools with resources.
type toolPolicy struct {
	resources bool
	rules     map[string]rule
}

var (
	member = access.Member
	admin  = access.Admin
)

// policy lists every tool action. An action that isn't listed is refused.
var policy = map[string]toolPolicy{
	"brand": {resources: true, rules: map[string]rule{
		"brand.list":            {brand: noBrand},
		"brand.select":          {need: member, brand: namedBrand},
		"brand.get":             read(member),
		"brand.update":          write(admin),
		"brand.create":          {need: access.SuperAdmin, write: true, brand: noBrand},
		"brand.delete":          {need: admin, write: true, brand: namedBrand, live: true},
		"brand.dashboard_stats": read(member),
		"domain.list":           read(member),
		"domain.get":            read(member),
		"domain.create":         write(admin),
		"domain.refresh":        write(admin),
		"domain.delete":         {need: admin, write: true, live: true},
	}},
	"campaign": {rules: map[string]rule{
		"create": write(member), "list": read(member), "get": read(member), "update": write(member),
		"delete": write(member), "schedule": write(member), "send": write(member), "stats": read(member),
	}},
	"contact": {rules: map[string]rule{
		"create": write(member), "list": read(member), "get": read(member), "update": write(member),
		"add_tags": write(member), "remove_tags": write(member), "unsubscribe": write(member),
		"block": write(member), "unblock": write(member), "activity": read(member),
	}},
	"email": {resources: true, rules: map[string]rule{
		"list.create": write(member), "list.list": read(member), "list.get": read(member),
		"list.update": write(member), "list.delete": write(member), "list.stats": read(member),
		"list.subscribers": read(member), "list.subscribe": write(member), "list.unsubscribe": write(member),
		"sequence.create": write(member), "sequence.list": read(member), "sequence.get": read(member),
		"sequence.update": write(member), "sequence.delete": write(member), "sequence.stats": read(member),
		"template.create": write(member), "template.list": read(member), "template.get": read(member),
		"template.update": write(member), "template.delete": write(member),
		"enrollment.enroll": write(member), "enrollment.unenroll": write(member), "enrollment.pause": write(member),
		"enrollment.resume": write(member), "enrollment.list": read(member),
		"entry_rule.create": write(member), "entry_rule.list": read(member), "entry_rule.update": write(member),
		"entry_rule.delete": write(member),
		"queue.list":        read(member), "queue.cancel": write(member),
	}},
	"webhook": {rules: map[string]rule{
		"create": write(admin), "list": read(admin), "get": read(admin), "update": write(admin),
		"delete": write(admin), "test": write(admin), "logs": read(admin),
	}},
	"design": {rules: map[string]rule{
		"create": write(member), "list": read(member), "get": read(member), "update": write(member),
		"delete": write(member),
	}},
	"transactional": {rules: map[string]rule{
		"create": write(member), "list": read(member), "get": read(member), "update": write(member),
		"delete": write(member), "stats": read(member),
	}},
	"stats": {resources: true, rules: map[string]rule{
		"overview.get": read(member), "email.get": read(member), "contact.get": read(member),
	}},
	"blocklist": {resources: true, rules: map[string]rule{
		"suppression.list": read(member), "suppression.add": write(member), "suppression.delete": write(member),
		"suppression.bulk_add": write(member), "suppression.clear": {need: admin, write: true, live: true},
		"domain.list": read(member), "domain.add": write(member), "domain.delete": write(member),
		"domain.bulk_add": write(member),
	}},
	"gdpr": {rules: map[string]rule{
		"lookup": read(member), "export": read(member), "get_consent": read(member),
		"update_consent": write(member), "delete": {need: admin, write: true, live: true},
	}},
}

// Caller is the person behind one tool call.
type Caller struct {
	UserID string
	Email  string
	Scopes []string
}

// callerFromToken reads the caller from the token this request carried. The
// MCP handler verifies that token against the database on every request.
func callerFromToken(ctx context.Context) (Caller, bool) {
	info := mcpauth.TokenInfoFromContext(ctx)
	if info == nil {
		return Caller{}, false
	}
	userInfo, ok := info.Extra["user_info"].(*mcpauth.UserInfo)
	if !ok || userInfo.UserID == "" {
		return Caller{}, false
	}
	return Caller{UserID: userInfo.UserID, Email: userInfo.Email, Scopes: info.Scopes}, true
}

// gateArgs are the arguments the gate reads from any tool call.
type gateArgs struct {
	Resource    string          `json:"resource"`
	Action      string          `json:"action"`
	ID          json.RawMessage `json:"id"`
	BrandID     string          `json:"brand_id"`
	Slug        string          `json:"slug"`
	ContactID   string          `json:"contact_id"`
	Email       string          `json:"email"`
	Domain      string          `json:"domain"`
	Name        string          `json:"name"`
	Confirm     bool            `json:"confirm"`
	ConfirmLive bool            `json:"confirm_live"`
}

func (a gateArgs) id() string {
	return strings.Trim(string(a.ID), `"`)
}

// target names what the call acts on, for the audit log.
func (a gateArgs) target() string {
	for _, s := range []string{a.id(), a.ContactID, a.Email, a.Domain, a.BrandID, a.Slug, a.Name} {
		if s != "" && s != "null" && s != "0" {
			return s
		}
	}
	return ""
}

var errNoScope = &mcpctx.ToolError{Code: "forbidden", Message: "This key or token can't call tools. It needs the " + mcpauth.ScopeFull + " scope."}

// gateResult is what the gate decided for one call.
type gateResult struct {
	audit   bool
	caller  Caller
	tool    string
	action  string
	brand   db.Organization // the brand the call acts in, if any
	brandID string
	target  string
}

// authorize runs the gate for one tool call. When it returns an error the
// call must not run.
func authorize(ctx context.Context, toolCtx *mcpctx.ToolContext, caller Caller, tool string, raw json.RawMessage) (gateResult, error) {
	var args gateArgs
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &args); err != nil {
			return gateResult{}, mcpctx.NewValidationError("arguments are not valid JSON", "")
		}
	}

	tp, ok := policy[tool]
	if !ok {
		return gateResult{}, mcpctx.NewNotFoundError(fmt.Sprintf("unknown tool '%s'", tool))
	}
	key := args.Action
	if tp.resources {
		key = args.Resource + "." + args.Action
	}
	r, ok := tp.rules[key]
	if !ok {
		return gateResult{}, unknownAction(tool, tp, args)
	}

	res := gateResult{audit: r.write, caller: caller, tool: tool, action: key, target: args.target()}

	if !slices.Contains(caller.Scopes, mcpauth.ScopeFull) {
		return res, errNoScope
	}

	switch r.brand {
	case sessionBrand:
		res.brand = toolCtx.SessionBrand()
		if res.brand.ID == "" {
			return res, mcpctx.ErrNoBrandSelected
		}
	case namedBrand:
		brand, err := lookupNamedBrand(ctx, toolCtx.DB(), args.BrandID, args.id(), args.Slug)
		if err != nil {
			return res, err
		}
		res.brand = brand
	}
	res.brandID = res.brand.ID

	if r.need == "" {
		_, err := access.Active(ctx, toolCtx.DB(), caller.UserID)
		return res, err
	}
	if _, err := access.Check(ctx, toolCtx.DB(), caller.UserID, res.brandID, r.need); err != nil {
		return res, err
	}

	if r.live && args.Confirm && !args.ConfirmLive {
		if err := liveKeyWarning(ctx, toolCtx.DB(), res.brandID); err != nil {
			return res, err
		}
	}
	return res, nil
}

func unknownAction(tool string, tp toolPolicy, args gateArgs) error {
	if tp.resources {
		resources := map[string]bool{}
		var actions []string
		for k := range tp.rules {
			res, act, _ := strings.Cut(k, ".")
			resources[res] = true
			if res == args.Resource {
				actions = append(actions, act)
			}
		}
		if !resources[args.Resource] {
			names := make([]string, 0, len(resources))
			for n := range resources {
				names = append(names, n)
			}
			sort.Strings(names)
			return mcpctx.NewValidationError(fmt.Sprintf("invalid resource '%s' for %s, must be: %s", args.Resource, tool, strings.Join(names, ", ")), "resource")
		}
		sort.Strings(actions)
		return mcpctx.NewValidationError(fmt.Sprintf("invalid action '%s' for resource '%s', must be: %s", args.Action, args.Resource, strings.Join(actions, ", ")), "action")
	}
	actions := make([]string, 0, len(tp.rules))
	for a := range tp.rules {
		actions = append(actions, a)
	}
	sort.Strings(actions)
	return mcpctx.NewValidationError(fmt.Sprintf("invalid action '%s' for %s, must be: %s", args.Action, tool, strings.Join(actions, ", ")), "action")
}

// lookupNamedBrand finds the brand a call names by ID or slug.
func lookupNamedBrand(ctx context.Context, q db.Querier, brandID, id, slug string) (db.Organization, error) {
	if brandID == "" {
		brandID = id
	}
	var brand db.Organization
	var err error
	switch {
	case brandID != "":
		brand, err = q.GetOrganizationByID(ctx, brandID)
	case slug != "":
		brand, err = q.GetOrganizationBySlug(ctx, slug)
	default:
		return db.Organization{}, mcpctx.NewValidationError("name the brand: pass id, brand_id or slug", "id")
	}
	if errors.Is(err, sql.ErrNoRows) {
		return db.Organization{}, mcpctx.NewNotFoundError("brand not found")
	}
	return brand, err
}

// liveKeyWarning refuses a destructive call while the brand's API key is
// sending mail, unless the caller confirms with confirm_live.
func liveKeyWarning(ctx context.Context, q db.Querier, brandID string) error {
	recent, err := q.GetRecentAPIKeySends(ctx, brandID)
	if err != nil {
		return err
	}
	if recent.Sends == 0 {
		return nil
	}
	return &mcpctx.ToolError{
		Code: "live_key",
		Message: fmt.Sprintf("This brand's API key is in live use: it sent %d emails in the last 7 days, the last at %s UTC. "+
			"Something depends on this brand (NeboAI's sign-in mail is one). To go ahead anyway, call again with confirm_live: true.",
			recent.Sends, recent.LastSentAt),
		Field: "confirm_live",
	}
}

// recordAudit writes one audit row. It runs off the request path.
func recordAudit(store db.Querier, g gateResult, outcome string) {
	if len(outcome) > 500 {
		outcome = outcome[:500]
	}
	go func() {
		err := store.CreateMCPAuditLog(context.Background(), db.CreateMCPAuditLogParams{
			ID:        uuid.NewString(),
			UserID:    g.caller.UserID,
			UserEmail: g.caller.Email,
			Tool:      g.tool,
			Action:    g.action,
			OrgID:     sql.NullString{String: g.brandID, Valid: g.brandID != ""},
			Target:    sql.NullString{String: g.target, Valid: g.target != ""},
			Result:    outcome,
		})
		if err != nil {
			logx.Errorf("MCP audit: could not record %s %s by %s: %v", g.tool, g.action, g.caller.UserID, err)
		}
	}()
}

// gated runs one tool call through the gate: authorize it, run it in a
// context that carries the brand the gate checked, and audit it when it
// changes something. It returns the gate's refusal, if any; run returns the
// outcome to record.
func gated(ctx context.Context, toolCtx *mcpctx.ToolContext, caller Caller, tool string, args json.RawMessage, run func(ctx context.Context) string) error {
	g, err := authorize(ctx, toolCtx, caller, tool, args)
	if err != nil {
		if g.audit {
			recordAudit(toolCtx.DB(), g, "refused: "+err.Error())
		}
		return err
	}
	if g.brand.ID != "" {
		ctx = mcpctx.WithCallBrand(ctx, g.brand)
	}
	result := run(ctx)
	if g.audit {
		recordAudit(toolCtx.DB(), g, result)
	}
	return nil
}

// Gate is the MCP receiving middleware that runs every tool call through the
// gate. The caller comes from the token the request carried.
func Gate(toolCtx *mcpctx.ToolContext) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			call, ok := req.(*mcp.CallToolRequest)
			if method != "tools/call" || !ok {
				return next(ctx, method, req)
			}
			caller, ok := callerFromToken(ctx)
			if !ok {
				return errorResult(&mcpctx.ToolError{Code: "unauthorized", Message: "Sign in first."}), nil
			}
			var result mcp.Result
			var callErr error
			if err := gated(ctx, toolCtx, caller, call.Params.Name, call.Params.Arguments, func(ctx context.Context) string {
				result, callErr = next(ctx, method, req)
				return outcome(result, callErr)
			}); err != nil {
				return errorResult(err), nil
			}
			return result, callErr
		}
	}
}

func errorResult(err error) *mcp.CallToolResult {
	res := &mcp.CallToolResult{}
	res.SetError(err)
	return res
}

func outcome(result mcp.Result, err error) string {
	if err != nil {
		return "error: " + err.Error()
	}
	if r, ok := result.(*mcp.CallToolResult); ok && r.IsError {
		if e := r.GetError(); e != nil {
			return "error: " + e.Error()
		}
		for _, c := range r.Content {
			if t, ok := c.(*mcp.TextContent); ok {
				return "error: " + t.Text
			}
		}
		return "error"
	}
	return "ok"
}
