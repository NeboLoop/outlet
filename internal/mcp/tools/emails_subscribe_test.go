package tools

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/mcp/mcpctx"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// list.subscribe refuses an address at a blocked domain, a contact already
// stored included, the way the hosted form and the SDK do.
func TestListSubscribe_RefusesABlockedDomainForAnExistingContact(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	for _, q := range []string{
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('org1', 'Brand', 'brand', 'k1')`,
		`INSERT INTO email_lists (id, public_id, org_id, name, slug) VALUES (1, 'news', 'org1', 'News', 'news')`,
		// Stored before the domain was blocked
		`INSERT INTO contacts (id, org_id, name, email) VALUES ('old', 'org1', 'Old', 'old@spam.example')`,
		`INSERT INTO blocked_domains (org_id, domain) VALUES ('org1', 'spam.example')`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err, q)
	}
	store := db.NewStore(conn)
	brand, err := store.GetOrganizationByID(context.Background(), "org1")
	require.NoError(t, err)
	ctx := mcpctx.WithCallBrand(context.Background(), brand)
	toolCtx := mcpctx.NewToolContext(&svc.ServiceContext{DB: store, Sequences: email.NewSequenceServiceWithBaseURL(store, nil, "")}, brand, "req", "test")
	subscribe := func(address string) error {
		_, _, err := handleListSubscribe(ctx, toolCtx, EmailInput{Resource: "list", Action: "subscribe", ID: "1", Email: address})
		return err
	}

	err = subscribe("old@spam.example")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "blocked")
	var subs int
	require.NoError(t, conn.QueryRow(`SELECT COUNT(*) FROM list_subscribers`).Scan(&subs))
	assert.Zero(t, subs)
	var attempts int
	require.NoError(t, conn.QueryRow(`SELECT block_attempts FROM blocked_domains`).Scan(&attempts))
	assert.Equal(t, 1, attempts)

	require.NoError(t, subscribe("pat@example.com"))
	require.NoError(t, conn.QueryRow(`SELECT COUNT(*) FROM list_subscribers`).Scan(&subs))
	assert.Equal(t, 1, subs)
}
