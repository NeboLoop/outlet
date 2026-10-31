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
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
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
