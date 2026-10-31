package mcp

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/mcp/mcpauth"
	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// These tests drive the MCP endpoint over HTTP the way a client does, against
// a real database: keys, people, brands and memberships.

type mcpFixture struct {
	t       *testing.T
	store   *db.Store
	handler *Handler
}

func newMCPFixture(t *testing.T) *mcpFixture {
	t.Helper()
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn))
	store := db.NewStore(conn)
	f := &mcpFixture{t: t, store: store, handler: NewHandler(&svc.ServiceContext{DB: store}, "https://outlet.test")}

	ctx := context.Background()
	for _, u := range []struct{ id, role string }{{"root", "super_admin"}, {"anne", "agent"}, {"mike", "agent"}} {
		_, err := store.CreateUser(ctx, db.CreateUserParams{
			ID: u.id, Email: u.id + "@example.com", PasswordHash: "x", Name: u.id, Role: u.role, Status: "active", EmailVerified: int64(1),
		})
		require.NoError(t, err)
	}
	for _, b := range []string{"alpha", "bravo"} {
		_, err := store.CreateOrganization(ctx, db.CreateOrganizationParams{ID: b + "-id", Name: b, Slug: b, ApiKey: "key-" + b})
		require.NoError(t, err)
	}
	// anne runs alpha; mike is a member of alpha. Neither belongs to bravo.
	f.join("alpha-id", "anne", "admin")
	f.join("alpha-id", "mike", "member")
	for _, k := range []struct{ id, user, scopes string }{
		{"root", "root", "mcp:full"}, {"anne", "anne", "mcp:full"}, {"mike", "mike", "mcp:full"}, {"anne-ro", "anne", "mcp:read"},
	} {
		_, err := store.CreateMCPAPIKey(ctx, db.CreateMCPAPIKeyParams{
			ID: k.id, UserID: k.user, Name: k.id, KeyHash: mcpauth.HashToken("lv_" + k.id), KeyPrefix: "lv_" + k.id,
			Scopes: sql.NullString{String: k.scopes, Valid: true},
		})
		require.NoError(t, err)
	}
	return f
}

func (f *mcpFixture) join(brand, user, role string) {
	require.NoError(f.t, f.store.AddUserToOrganization(context.Background(), db.AddUserToOrganizationParams{
		OrgID: brand, UserID: user, Role: sql.NullString{String: role, Valid: true},
	}))
}

// post sends one JSON-RPC message and returns the HTTP status and the
// decoded JSON-RPC response (nil for a non-200 status).
func (f *mcpFixture) post(key, session string, msg map[string]any) (int, map[string]any) {
	f.t.Helper()
	body, err := json.Marshal(msg)
	require.NoError(f.t, err)
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer lv_"+key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Mcp-Session-Id", session)
	rr := httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		return rr.Code, nil
	}
	raw := rr.Body.Bytes()
	if strings.HasPrefix(rr.Header().Get("Content-Type"), "text/event-stream") {
		sc := bufio.NewScanner(bytes.NewReader(raw))
		sc.Buffer(make([]byte, 1<<20), 1<<20)
		for sc.Scan() {
			if line := sc.Text(); strings.HasPrefix(line, "data: ") {
				raw = []byte(strings.TrimPrefix(line, "data: "))
			}
		}
	}
	var out map[string]any
	require.NoError(f.t, json.Unmarshal(raw, &out), string(raw))
	return rr.Code, out
}

func (f *mcpFixture) initialize(key, session string) {
	f.t.Helper()
	code, out := f.post(key, session, map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{"protocolVersion": "2025-06-18", "capabilities": map[string]any{}, "clientInfo": map[string]any{"name": "test", "version": "1"}},
	})
	require.Equal(f.t, http.StatusOK, code)
	require.NotNil(f.t, out["result"], out)
}

// call runs one tool call in a session and returns the result text and
// whether it was an error.
func (f *mcpFixture) call(key, session, tool string, args map[string]any) (string, bool) {
	f.t.Helper()
	code, out := f.post(key, session, map[string]any{
		"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": map[string]any{"name": tool, "arguments": args},
	})
	require.Equal(f.t, http.StatusOK, code)
	if e, ok := out["error"]; ok {
		b, _ := json.Marshal(e)
		return string(b), true
	}
	res := out["result"].(map[string]any)
	var text strings.Builder
	for _, c := range res["content"].([]any) {
		text.WriteString(c.(map[string]any)["text"].(string))
	}
	isErr, _ := res["isError"].(bool)
	return text.String(), isErr
}

func (f *mcpFixture) audit() []map[string]string {
	f.t.Helper()
	rows, err := f.store.GetDB().Query(`SELECT user_id, user_email, tool, action, COALESCE(org_id, ''), COALESCE(target, ''), result FROM mcp_audit_log ORDER BY created_at, rowid`)
	require.NoError(f.t, err)
	defer rows.Close()
	var out []map[string]string
	for rows.Next() {
		var u, e, tool, action, org, target, result string
		require.NoError(f.t, rows.Scan(&u, &e, &tool, &action, &org, &target, &result))
		out = append(out, map[string]string{"user": u, "email": e, "tool": tool, "action": action, "brand": org, "target": target, "result": result})
	}
	return out
}

func TestMCP_BrandListShowsOnlyTheCallersBrands(t *testing.T) {
	f := newMCPFixture(t)
	f.initialize("mike", "s1")
	text, isErr := f.call("mike", "s1", "brand", map[string]any{"resource": "brand", "action": "list"})
	require.False(t, isErr, text)
	assert.Contains(t, text, `"slug":"alpha"`)
	assert.NotContains(t, text, `"slug":"bravo"`)
	assert.Contains(t, text, `"role":"member"`)

	f.initialize("root", "s2")
	text, _ = f.call("root", "s2", "brand", map[string]any{"resource": "brand", "action": "list"})
	assert.Contains(t, text, `"slug":"alpha"`)
	assert.Contains(t, text, `"slug":"bravo"`)
}

func TestMCP_CallerCantReachAnotherBrand(t *testing.T) {
	f := newMCPFixture(t)
	f.initialize("anne", "s1")
	text, isErr := f.call("anne", "s1", "brand", map[string]any{"resource": "brand", "action": "select", "slug": "bravo"})
	assert.True(t, isErr)
	assert.Contains(t, text, "You don't have access to this brand.")

	// The super admin reaches every brand.
	f.initialize("root", "s2")
	_, isErr = f.call("root", "s2", "brand", map[string]any{"resource": "brand", "action": "select", "slug": "bravo"})
	assert.False(t, isErr)
}

func TestMCP_RolesAreEnforced(t *testing.T) {
	f := newMCPFixture(t)
	f.initialize("mike", "s1")
	_, isErr := f.call("mike", "s1", "brand", map[string]any{"resource": "brand", "action": "select", "slug": "alpha"})
	require.False(t, isErr)

	// A member reads and runs everyday mail...
	_, isErr = f.call("mike", "s1", "campaign", map[string]any{"action": "list"})
	assert.False(t, isErr)
	// ...but brand settings, webhooks and destructive actions need the admin role.
	text, isErr := f.call("mike", "s1", "brand", map[string]any{"resource": "brand", "action": "update", "name": "Renamed"})
	assert.True(t, isErr)
	assert.Contains(t, text, "This needs the brand's admin role.")
	_, isErr = f.call("mike", "s1", "webhook", map[string]any{"action": "list"})
	assert.True(t, isErr)
	_, isErr = f.call("mike", "s1", "blocklist", map[string]any{"resource": "suppression", "action": "clear", "confirm": true})
	assert.True(t, isErr)
	// Only a super admin makes brands.
	text, isErr = f.call("mike", "s1", "brand", map[string]any{"resource": "brand", "action": "create", "name": "Mine"})
	assert.True(t, isErr)
	assert.Contains(t, text, "Only a super admin can do this.")

	// Removing someone from a brand takes effect on their next call.
	require.NoError(t, f.store.RemoveUserFromOrganization(context.Background(), db.RemoveUserFromOrganizationParams{OrgID: "alpha-id", UserID: "mike"}))
	text, isErr = f.call("mike", "s1", "campaign", map[string]any{"action": "list"})
	assert.True(t, isErr)
	assert.Contains(t, text, "You don't have access to this brand.")
}

func TestMCP_UnknownActionIsRefused(t *testing.T) {
	f := newMCPFixture(t)
	f.initialize("root", "s1")
	text, isErr := f.call("root", "s1", "campaign", map[string]any{"action": "explode"})
	assert.True(t, isErr)
	assert.Contains(t, text, "invalid action 'explode' for campaign")
}

func TestMCP_ScopesAreEnforced(t *testing.T) {
	f := newMCPFixture(t)
	f.initialize("anne-ro", "s1")
	text, isErr := f.call("anne-ro", "s1", "brand", map[string]any{"resource": "brand", "action": "list"})
	assert.True(t, isErr)
	assert.Contains(t, text, "mcp:full")
}

func TestMCP_NewSessionChoosesItsOwnBrand(t *testing.T) {
	f := newMCPFixture(t)
	f.initialize("anne", "s1")
	_, isErr := f.call("anne", "s1", "brand", map[string]any{"resource": "brand", "action": "select", "slug": "alpha"})
	require.False(t, isErr)
	_, isErr = f.call("anne", "s1", "campaign", map[string]any{"action": "list"})
	require.False(t, isErr)

	// Wait for the session choice to reach the database, then start a new session.
	require.Eventually(t, func() bool {
		s, err := f.store.GetMCPSession(context.Background(), "s1")
		return err == nil && s.OrgID.Valid
	}, 2*time.Second, 10*time.Millisecond)
	f.initialize("anne", "s2")
	text, isErr := f.call("anne", "s2", "campaign", map[string]any{"action": "list"})
	assert.True(t, isErr)
	assert.Contains(t, text, "No brand selected")

	// Another person can't pick up a session's brand by reusing its ID.
	f.handler.sessionCache.Delete("s1")
	f.handler.brandSelectionCache.Delete("s1")
	f.initialize("mike", "s1")
	text, isErr = f.call("mike", "s1", "campaign", map[string]any{"action": "list"})
	assert.True(t, isErr)
	assert.Contains(t, text, "No brand selected")
}

func TestMCP_RevokedKeyStopsAtOnce(t *testing.T) {
	f := newMCPFixture(t)
	f.initialize("anne", "s1")
	_, isErr := f.call("anne", "s1", "brand", map[string]any{"resource": "brand", "action": "list"})
	require.False(t, isErr)

	n, err := f.store.RevokeMCPAPIKey(context.Background(), db.RevokeMCPAPIKeyParams{ID: "anne", UserID: "anne"})
	require.NoError(t, err)
	require.EqualValues(t, 1, n)

	code, _ := f.post("anne", "s1", map[string]any{
		"jsonrpc": "2.0", "id": 3, "method": "tools/call", "params": map[string]any{"name": "brand", "arguments": map[string]any{"resource": "brand", "action": "list"}},
	})
	assert.Equal(t, http.StatusUnauthorized, code)
}

func TestMCP_DisabledPersonIsRefused(t *testing.T) {
	f := newMCPFixture(t)
	f.initialize("anne", "s1")
	_, err := f.store.GetDB().Exec(`UPDATE users SET status = 'inactive' WHERE id = 'anne'`)
	require.NoError(t, err)
	code, _ := f.post("anne", "s1", map[string]any{
		"jsonrpc": "2.0", "id": 3, "method": "tools/call", "params": map[string]any{"name": "brand", "arguments": map[string]any{"resource": "brand", "action": "list"}},
	})
	assert.Equal(t, http.StatusUnauthorized, code)
}

func TestMCP_BrandDeleteNamesItsBrand(t *testing.T) {
	f := newMCPFixture(t)
	f.initialize("root", "s1")
	_, isErr := f.call("root", "s1", "brand", map[string]any{"resource": "brand", "action": "select", "slug": "alpha"})
	require.False(t, isErr)

	// The session's brand is never the default.
	text, isErr := f.call("root", "s1", "brand", map[string]any{"resource": "brand", "action": "delete", "confirm": true})
	assert.True(t, isErr)
	assert.Contains(t, text, "name the brand")

	text, isErr = f.call("root", "s1", "brand", map[string]any{"resource": "brand", "action": "delete", "slug": "bravo", "confirm": true})
	require.False(t, isErr, text)
	_, err := f.store.GetOrganizationBySlug(context.Background(), "bravo")
	assert.ErrorIs(t, err, sql.ErrNoRows)
	_, err = f.store.GetOrganizationBySlug(context.Background(), "alpha")
	assert.NoError(t, err)
}

func TestMCP_DestructiveActionsWarnWhenTheKeyIsLive(t *testing.T) {
	f := newMCPFixture(t)
	ctx := context.Background()
	// alpha's API key sent mail today (the SDK and SMTP record each send).
	_, err := f.store.CreateTransactionalEmail(ctx, db.CreateTransactionalEmailParams{
		ID: "tpl", OrgID: "alpha-id", Name: "Sign-in", Slug: "sign-in", Subject: "Your code", HtmlBody: "<p>1234</p>",
	})
	require.NoError(t, err)
	_, err = f.store.GetDB().Exec(`INSERT INTO transactional_sends (id, template_id, org_id, to_email) VALUES ('s1', 'tpl', 'alpha-id', 'a@example.com')`)
	require.NoError(t, err)

	f.initialize("anne", "s1")
	_, isErr := f.call("anne", "s1", "brand", map[string]any{"resource": "brand", "action": "select", "slug": "alpha"})
	require.False(t, isErr)

	for _, c := range []struct {
		tool string
		args map[string]any
	}{
		{"blocklist", map[string]any{"resource": "suppression", "action": "clear", "confirm": true}},
		{"gdpr", map[string]any{"action": "delete", "contact_id": "c1", "confirm": true}},
		{"brand", map[string]any{"resource": "brand", "action": "delete", "slug": "alpha", "confirm": true}},
		{"brand", map[string]any{"resource": "domain", "action": "delete", "id": "d1", "confirm": true}},
	} {
		text, isErr := f.call("anne", "s1", c.tool, c.args)
		assert.True(t, isErr, c.tool)
		assert.Contains(t, text, "in live use", c.tool)
		assert.Contains(t, text, "confirm_live", c.tool)
	}

	// Confirming goes ahead.
	text, isErr := f.call("anne", "s1", "blocklist", map[string]any{"resource": "suppression", "action": "clear", "confirm": true, "confirm_live": true})
	assert.False(t, isErr, text)
}

// list makes an email list in a brand and returns its ID.
func (f *mcpFixture) list(brand string) string {
	f.t.Helper()
	l, err := f.store.CreateEmailList(context.Background(), db.CreateEmailListParams{PublicID: "p-" + brand, OrgID: brand + "-id", Name: brand, Slug: brand})
	require.NoError(f.t, err)
	return strconv.FormatInt(l.ID, 10)
}

func TestMCP_SendQueuesThroughTheScheduler(t *testing.T) {
	f := newMCPFixture(t)
	alphaList := f.list("alpha")
	f.initialize("mike", "s1")
	_, isErr := f.call("mike", "s1", "brand", map[string]any{"resource": "brand", "action": "select", "slug": "alpha"})
	require.False(t, isErr)
	text, isErr := f.call("mike", "s1", "campaign", map[string]any{
		"action": "create", "name": "News", "subject": "Hi", "html_body": "<p>Hi</p>", "list_ids": alphaList,
	})
	require.False(t, isErr, text)
	var created struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal([]byte(text), &created))

	text, isErr = f.call("mike", "s1", "campaign", map[string]any{"action": "send", "id": created.ID})
	require.False(t, isErr, text)
	assert.Contains(t, text, `"status":"scheduled"`)
	assert.NotContains(t, text, `"sent":true`)

	// The scheduler, the one sending path, picks it up now.
	due, err := f.store.GetScheduledCampaigns(context.Background())
	require.NoError(t, err)
	require.Len(t, due, 1)
	assert.Equal(t, created.ID, due[0].ID)
}

func TestMCP_WritesAreAudited(t *testing.T) {
	f := newMCPFixture(t)
	f.initialize("mike", "s1")
	_, isErr := f.call("mike", "s1", "brand", map[string]any{"resource": "brand", "action": "select", "slug": "alpha"})
	require.False(t, isErr)
	_, isErr = f.call("mike", "s1", "campaign", map[string]any{"action": "list"})
	require.False(t, isErr)
	_, isErr = f.call("mike", "s1", "contact", map[string]any{"action": "create", "email": "new@example.com", "name": "New"})
	require.False(t, isErr)
	_, isErr = f.call("mike", "s1", "brand", map[string]any{"resource": "brand", "action": "update", "name": "Renamed"})
	require.True(t, isErr)

	var rows []map[string]string
	require.Eventually(t, func() bool {
		rows = f.audit()
		return len(rows) == 2
	}, 2*time.Second, 10*time.Millisecond, "reads are not audited; the two writes are")

	assert.Equal(t, map[string]string{
		"user": "mike", "email": "mike@example.com", "tool": "contact", "action": "create",
		"brand": "alpha-id", "target": "new@example.com", "result": "ok",
	}, rows[0])
	assert.Equal(t, "brand", rows[1]["tool"])
	assert.Equal(t, "brand.update", rows[1]["action"])
	assert.Equal(t, "alpha-id", rows[1]["brand"])
	assert.Contains(t, rows[1]["result"], "refused: ")
}

// Every record ID a call hands a tool must belong to the call's brand: a
// member of alpha can't send to bravo's lists, add emails to bravo's
// sequences, attach a sequence to bravo's list, cancel bravo's mail or read
// bravo's sequence stats by naming their IDs.
func TestMCP_RecordsOfAnotherBrandAreRefused(t *testing.T) {
	f := newMCPFixture(t)
	ctx := context.Background()
	alphaList, bravoList := f.list("alpha"), f.list("bravo")
	bravoListID, _ := strconv.ParseInt(bravoList, 10, 64)
	_, err := f.store.CreateSequence(ctx, db.CreateSequenceParams{
		ID: "bravo-seq", OrgID: sql.NullString{String: "bravo-id", Valid: true}, ListID: sql.NullInt64{Int64: bravoListID, Valid: true},
		Slug: "welcome", Name: "Welcome", TriggerEvent: "on_subscribe", IsActive: sql.NullInt64{Int64: 1, Valid: true}, SequenceType: "lifecycle",
	})
	require.NoError(t, err)
	_, err = f.store.CreateTemplate(ctx, db.CreateTemplateParams{ID: "bravo-tpl", SequenceID: sql.NullString{String: "bravo-seq", Valid: true}, Position: 1, Subject: "Hi", HtmlBody: "<p>Hi</p>"})
	require.NoError(t, err)
	_, err = f.store.CreateContact(ctx, db.CreateContactParams{ID: "bravo-contact", OrgID: sql.NullString{String: "bravo-id", Valid: true}, Name: "B", Email: "b@example.com", Status: "active"})
	require.NoError(t, err)
	_, err = f.store.QueueEmail(ctx, db.QueueEmailParams{
		ID: "bravo-queued", ContactID: sql.NullString{String: "bravo-contact", Valid: true}, TemplateID: sql.NullString{String: "bravo-tpl", Valid: true}, ScheduledFor: "2030-01-01 00:00:00",
	})
	require.NoError(t, err)

	f.initialize("mike", "s1")
	_, isErr := f.call("mike", "s1", "brand", map[string]any{"resource": "brand", "action": "select", "slug": "alpha"})
	require.False(t, isErr)

	for _, c := range []struct {
		tool string
		args map[string]any
	}{
		{"campaign", map[string]any{"action": "create", "name": "x", "subject": "x", "html_body": "x", "list_ids": bravoList}},
		{"campaign", map[string]any{"action": "create", "name": "x", "subject": "x", "html_body": "x", "list_ids": alphaList + "," + bravoList}},
		{"campaign", map[string]any{"action": "create", "name": "x", "subject": "x", "html_body": "x", "list_ids": alphaList, "exclude_list_ids": bravoList}},
		{"email", map[string]any{"resource": "template", "action": "create", "sequence_id": "bravo-seq", "subject": "x", "html_body": "x"}},
		{"email", map[string]any{"resource": "sequence", "action": "create", "name": "x", "list_id": bravoList}},
		{"email", map[string]any{"resource": "queue", "action": "cancel", "id": "bravo-queued"}},
		{"stats", map[string]any{"resource": "email", "action": "get", "sequence_id": "bravo-seq"}},
	} {
		text, isErr := f.call("mike", "s1", c.tool, c.args)
		assert.True(t, isErr, "%s %v: %s", c.tool, c.args, text)
		assert.Contains(t, text, "not found", "%s %v", c.tool, c.args)
	}

	// Changing a campaign to bravo's list is refused too.
	text, isErr := f.call("mike", "s1", "campaign", map[string]any{"action": "create", "name": "x", "subject": "x", "html_body": "x", "list_ids": alphaList})
	require.False(t, isErr, text)
	var created struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal([]byte(text), &created))
	text, isErr = f.call("mike", "s1", "campaign", map[string]any{"action": "update", "id": created.ID, "list_ids": bravoList})
	assert.True(t, isErr, text)

	// Nothing of bravo's changed.
	templates, err := f.store.ListTemplatesBySequence(ctx, sql.NullString{String: "bravo-seq", Valid: true})
	require.NoError(t, err)
	assert.Len(t, templates, 1)
	var status string
	require.NoError(t, f.store.GetDB().QueryRow(`SELECT status FROM email_queue WHERE id = 'bravo-queued'`).Scan(&status))
	assert.Equal(t, "pending", status)
	var seqs int
	require.NoError(t, f.store.GetDB().QueryRow(`SELECT COUNT(*) FROM email_sequences WHERE list_id = ?`, bravoListID).Scan(&seqs))
	assert.Equal(t, 1, seqs)
}

// A domain belongs to one brand: alpha can't claim bravo's domain or send
// from it.
func TestMCP_AnotherBrandsDomainIsRefused(t *testing.T) {
	f := newMCPFixture(t)
	alphaList := f.list("alpha")
	_, err := f.store.CreateDomainIdentity(context.Background(), db.CreateDomainIdentityParams{ID: "d1", OrgID: "bravo-id", Domain: "bravo.example"})
	require.NoError(t, err)

	f.initialize("anne", "s1")
	_, isErr := f.call("anne", "s1", "brand", map[string]any{"resource": "brand", "action": "select", "slug": "alpha"})
	require.False(t, isErr)

	text, isErr := f.call("anne", "s1", "brand", map[string]any{"resource": "domain", "action": "create", "domain": "Bravo.example"})
	assert.True(t, isErr)
	assert.Contains(t, text, "Another brand already uses")

	text, isErr = f.call("anne", "s1", "campaign", map[string]any{
		"action": "create", "name": "x", "subject": "x", "html_body": "x", "list_ids": alphaList, "from_email": "security@bravo.example",
	})
	assert.True(t, isErr)
	assert.Contains(t, text, "can't send from security@bravo.example")

	// Both refused writes are audited in the background; wait for them so
	// nothing is still writing when the test's database is removed.
	require.Eventually(t, func() bool {
		seen := map[string]bool{}
		for _, r := range f.audit() {
			seen[r["tool"]+" "+r["action"]] = true
		}
		return seen["brand domain.create"] && seen["campaign create"]
	}, 2*time.Second, 10*time.Millisecond)
}
