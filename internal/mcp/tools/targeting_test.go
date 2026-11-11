package tools

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/mcp/mcpctx"
	"github.com/outlet-sh/outlet/internal/services/crypto"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/services/outbound"
	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// The start-from-a-website calls work on the first try with plain JSON
// objects, through the same gate as MCP: the profile, a segment with
// filters, a material for it, a do-not-contact list, and the brief that
// hands them all back.
func TestTargetingFirstCalls(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	store := db.NewStore(conn)
	ctx := context.Background()
	_, err = store.CreateUser(ctx, db.CreateUserParams{ID: "nebo", Email: "nebo@example.com", PasswordHash: "x", Name: "N", Role: "agent", Status: "active", EmailVerified: int64(1)})
	require.NoError(t, err)
	org, err := store.CreateOrganization(ctx, db.CreateOrganizationParams{ID: "o-id", Name: "o", Slug: "o", ApiKey: "key-o"})
	require.NoError(t, err)
	require.NoError(t, store.AddUserToOrganization(ctx, db.AddUserToOrganizationParams{OrgID: org.ID, UserID: "nebo", Role: sql.NullString{String: "admin", Valid: true}}))
	user, err := access.Active(ctx, store, "nebo")
	require.NoError(t, err)
	cs, err := crypto.NewService("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	require.NoError(t, err)
	mailer := email.NewService(store, cs)
	seq := email.NewSequenceServiceWithBaseURL(store, mailer, "https://outlet.example.com")
	sc := &svc.ServiceContext{DB: store, Outbound: outbound.New(store, cs, outbound.Config{Mailer: mailer, Sequences: seq})}
	toolCtx := mcpctx.NewUserToolContext(sc, user, "req", "test", "session")
	toolCtx.SelectBrand(org)
	registry := NewRegistryWithTools(toolCtx)
	call := func(tool string, args map[string]any) any {
		out, err := registry.Call(ctx, tool, args)
		require.NoError(t, err, tool, args)
		return out
	}

	call("targeting", map[string]any{"resource": "profile", "action": "update", "profile": map[string]any{"website": "https://acme.com", "sells": "Booking software", "audience": "Small clinics"}})
	out := call("targeting", map[string]any{"resource": "segment", "action": "create", "segment": map[string]any{
		"name": "Utah dental", "description": "Dental clinics in Utah", "filters": map[string]any{"countries": []string{"US"}, "company_sizes": []string{"11-50"}}}})
	sg := out.(map[string]any)["segment"].(outbound.Segment)
	assert.Equal(t, "Utah dental", sg.Name)
	mat := call("targeting", map[string]any{"resource": "material", "action": "create", "material": map[string]any{"kind": "case_study", "title": "Clinic Z", "summary": "Bookings doubled.", "segment": "Utah dental"}}).(outbound.Material)
	call("targeting", map[string]any{"resource": "do_not_contact", "action": "add", "dnc_list": "Customers", "entries": []string{"ann@client.com"}})
	chk := call("targeting", map[string]any{"resource": "do_not_contact", "action": "check", "email": "ann@client.com"}).(map[string]any)
	assert.Equal(t, true, chk["excluded"])
	assert.Equal(t, outbound.SourceList, chk["source"])

	b := call("draft_outreach", map[string]any{"action": "brief", "list": "Utah dental"}).(outbound.Brief)
	assert.Equal(t, "https://acme.com", b.Profile.Website)
	require.Len(t, b.Materials, 1)
	assert.Equal(t, mat.ID, b.Materials[0].ID)
	assert.NotEmpty(t, b.Rules)

	guide := call("explain", map[string]any{"action": "get", "topic": "start_from_website"}).(map[string]string)
	assert.Contains(t, guide["answer"], "targeting(resource: profile, action: update")

	_, err = registry.Call(ctx, "targeting", map[string]any{"resource": "segment", "action": "explode"})
	assert.Error(t, err)
}
