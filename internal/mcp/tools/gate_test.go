package tools

import (
	"context"
	"database/sql"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/mcp/mcpctx"
	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// Every action a tool offers has a rule in the gate, and every rule is an
// action a tool offers. A new action without a rule would be refused.
func TestPolicyCoversEveryToolAction(t *testing.T) {
	flat := func(actions []string) []string { return actions }
	byResource := func(m map[string][]string) []string {
		var out []string
		for res, actions := range m {
			for _, a := range actions {
				out = append(out, res+"."+a)
			}
		}
		return out
	}
	tools := map[string][]string{
		"brand":         byResource(brandActions),
		"campaign":      flat(campaignActions),
		"contact":       flat(contactActions),
		"email":         byResource(emailActions),
		"webhook":       flat(webhookActions),
		"design":        flat(designActions),
		"transactional": flat(transactionalActions),
		"stats":         byResource(statsActions),
		"blocklist":     byResource(blocklistActions),
		"gdpr":          flat(gdprActions),
	}
	require.Len(t, policy, len(tools))
	for tool, actions := range tools {
		var rules []string
		for k := range policy[tool].rules {
			rules = append(rules, k)
		}
		sort.Strings(actions)
		sort.Strings(rules)
		assert.Equal(t, actions, rules, tool)
	}
}

// The direct-call registry passes the same gate as MCP.
func TestRegistryCallsPassTheGate(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn))
	store := db.NewStore(conn)
	ctx := context.Background()
	_, err = store.CreateUser(ctx, db.CreateUserParams{ID: "mike", Email: "mike@example.com", PasswordHash: "x", Name: "M", Role: "agent", Status: "active", EmailVerified: int64(1)})
	require.NoError(t, err)
	user, err := access.Active(ctx, store, "mike")
	require.NoError(t, err)

	toolCtx := mcpctx.NewUserToolContext(&svc.ServiceContext{DB: store}, user, "req", "test", "session")
	registry := NewRegistryWithTools(toolCtx)

	_, err = registry.Call(ctx, "brand", map[string]interface{}{"resource": "brand", "action": "create", "name": "Mine"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Only a super admin can do this.")

	require.Eventually(t, func() bool {
		var n int
		require.NoError(t, conn.QueryRow(`SELECT COUNT(*) FROM mcp_audit_log WHERE user_id = 'mike' AND action = 'brand.create' AND result LIKE 'refused: %'`).Scan(&n))
		return n == 1
	}, 2*time.Second, 10*time.Millisecond)
	_, err = store.GetOrganizationBySlug(ctx, "mine")
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

// A brand.select that lands between the gate's check and the handler can't
// move the call: the handler acts in the brand the gate checked. Here anne
// is an admin of alpha and only a member of bravo, and her session switches
// to bravo while a webhook.create (admin only) checked for alpha is running.
func TestGateHandsItsBrandToTheHandler(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn))
	store := db.NewStore(conn)
	ctx := context.Background()
	_, err = store.CreateUser(ctx, db.CreateUserParams{ID: "anne", Email: "anne@example.com", PasswordHash: "x", Name: "A", Role: "agent", Status: "active", EmailVerified: int64(1)})
	require.NoError(t, err)
	brands := map[string]db.Organization{}
	for b, role := range map[string]string{"alpha": "admin", "bravo": "member"} {
		org, err := store.CreateOrganization(ctx, db.CreateOrganizationParams{ID: b + "-id", Name: b, Slug: b, ApiKey: "key-" + b})
		require.NoError(t, err)
		brands[b] = org
		require.NoError(t, store.AddUserToOrganization(ctx, db.AddUserToOrganizationParams{OrgID: org.ID, UserID: "anne", Role: sql.NullString{String: role, Valid: true}}))
	}
	user, err := access.Active(ctx, store, "anne")
	require.NoError(t, err)
	toolCtx := mcpctx.NewUserToolContext(&svc.ServiceContext{DB: store}, user, "req", "test", "session")
	toolCtx.SelectBrand(brands["alpha"])
	caller := Caller{UserID: "anne", Email: "anne@example.com", Scopes: []string{"mcp:full"}}

	err = gated(ctx, toolCtx, caller, "webhook", []byte(`{"action":"create","url":"https://hooks.example.com/x","events":"contact.created"}`), func(ctx context.Context) string {
		toolCtx.SelectBrand(brands["bravo"]) // the racing brand.select
		_, _, err := webhookHandler(toolCtx)(ctx, nil, WebhookInput{Action: "create", URL: "https://hooks.example.com/x", Events: "contact.created"})
		require.NoError(t, err)
		return "ok"
	})
	require.NoError(t, err)

	alpha, err := store.ListWebhooks(ctx, "alpha-id")
	require.NoError(t, err)
	bravo, err := store.ListWebhooks(ctx, "bravo-id")
	require.NoError(t, err)
	assert.Len(t, alpha, 1, "the webhook is made in the brand the gate checked")
	assert.Empty(t, bravo)

	// The next call is checked for bravo, where anne is only a member.
	err = gated(ctx, toolCtx, caller, "webhook", []byte(`{"action":"create","url":"https://hooks.example.com/y"}`), func(ctx context.Context) string {
		t.Fatal("the handler must not run")
		return ""
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "This needs the brand's admin role.")
}
