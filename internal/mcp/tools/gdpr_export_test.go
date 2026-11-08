package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	admingdpr "github.com/outlet-sh/outlet/internal/logic/admin/gdpr"
	"github.com/outlet-sh/outlet/internal/mcp/mcpctx"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// The GDPR export is one export: the admin screen's file and the MCP gdpr
// tool's export carry the same data, activity included. MCP can't export
// another brand's contact.
func TestGDPRExport_AdminAndMCPGiveTheSameExport(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	for _, q := range []string{
		`INSERT INTO users (id, email, password_hash, name, role, status, email_verified) VALUES ('mike', 'mike@example.com', 'x', 'M', 'agent', 'active', 1)`,
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('org1', 'Brand', 'brand', 'k1'), ('org2', 'Other', 'other', 'k2')`,
		`INSERT INTO org_users (org_id, user_id, role) VALUES ('org1', 'mike', 'member')`,
		`INSERT INTO contacts (id, org_id, name, email, gdpr_consent) VALUES ('ann', 'org1', 'Ann', 'ann@example.com', 1), ('bea', 'org2', 'Bea', 'bea@example.com', 0)`,
		`INSERT INTO contact_tags (contact_id, tag) VALUES ('ann', 'vip')`,
		`INSERT INTO email_campaigns (id, org_id, name, subject, html_body) VALUES ('camp1', 'org1', 'Launch', 'Hi', '<p>hi</p>')`,
		`INSERT INTO campaign_sends (id, campaign_id, contact_id, status, sent_at) VALUES ('cs1', 'camp1', 'ann', 'sent', '2026-01-02 03:04:05')`,
		`INSERT INTO transactional_emails (id, org_id, name, slug, subject, html_body) VALUES ('te1', 'org1', 'Receipt', 'receipt', 'Receipt', '<p>r</p>')`,
		`INSERT INTO transactional_sends (id, template_id, org_id, to_email, contact_id, status) VALUES ('ts1', 'te1', 'org1', 'ann@example.com', 'ann', 'sent')`,
		`INSERT INTO email_clicks (id, contact_id, link_url, clicked_at) VALUES ('k1', 'ann', 'https://example.com/a', '2026-01-02 03:05:00')`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err, q)
	}
	store := db.NewStore(conn)
	s := &svc.ServiceContext{DB: store}

	user, err := access.Active(context.Background(), store, "mike")
	require.NoError(t, err)
	file, err := admingdpr.NewExportContactDataLogic(access.WithUser(context.Background(), user), s).ExportContactData(&types.GDPRExportRequest{ContactId: "ann"})
	require.NoError(t, err)
	var fromAdmin map[string]any
	require.NoError(t, json.Unmarshal([]byte(file.Data), &fromAdmin))

	brand, err := store.GetOrganizationByID(context.Background(), "org1")
	require.NoError(t, err)
	ctx := mcpctx.WithCallBrand(context.Background(), brand)
	toolCtx := mcpctx.NewToolContext(s, brand, "req", "test")
	_, out, err := handleGDPRExport(ctx, toolCtx, GDPRInput{Action: "export", ContactID: "ann"})
	require.NoError(t, err)
	b, err := json.Marshal(out)
	require.NoError(t, err)
	var fromMCP map[string]any
	require.NoError(t, json.Unmarshal(b, &fromMCP))

	delete(fromAdmin, "exported_at")
	delete(fromMCP, "exported_at")
	assert.Equal(t, fromAdmin, fromMCP)
	assert.Len(t, fromMCP["campaign_sends"], 1)
	assert.Len(t, fromMCP["transactional_sends"], 1)
	assert.Len(t, fromMCP["clicks"], 1)
	assert.Equal(t, []any{"vip"}, fromMCP["tags"])

	_, _, err = handleGDPRExport(ctx, toolCtx, GDPRInput{Action: "export", ContactID: "bea"})
	assert.Error(t, err)
}
