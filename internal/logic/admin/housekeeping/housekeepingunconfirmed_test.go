package housekeeping

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// Unconfirmed means a double opt-in signup still waiting to confirm, for
// longer than the days given (0 means every one, not 30). Someone added by
// hand, imported, confirmed or found for outreach is never one, though none
// of them is marked verified.
func TestHousekeepingUnconfirmed_TakesTheDaysAsGiven(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	store := db.NewStore(conn)
	ctx := context.Background()
	for _, q := range []string{
		`INSERT INTO users (id, email, password_hash, name, role, status) VALUES ('anne', 'anne@example.com', 'x', 'Anne', 'agent', 'active')`,
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('org1', 'One', 'one', 'k1')`,
		`INSERT INTO org_users (org_id, user_id, role) VALUES ('org1', 'anne', 'admin')`,
		`INSERT INTO email_lists (id, public_id, org_id, name, slug) VALUES (1, 'news', 'org1', 'News', 'news'), (2, 'more', 'org1', 'More', 'more')`,
		`INSERT INTO contacts (id, org_id, name, email, created_at) VALUES
			('new', 'org1', 'N', 'n@example.com', datetime('now', '-1 hour')),
			('old', 'org1', 'O', 'o@example.com', datetime('now', '-40 days')),
			('added', 'org1', 'A', 'a@example.com', datetime('now', '-40 days')),
			('confirmed', 'org1', 'C', 'c@example.com', datetime('now', '-40 days')),
			('prospect', 'org1', 'P', 'p@example.com', datetime('now', '-40 days')),
			('also_active', 'org1', 'B', 'b@example.com', datetime('now', '-40 days')),
			('no_list', 'org1', 'L', 'l@example.com', datetime('now', '-40 days'))`,
		`INSERT INTO list_subscribers (id, list_id, contact_id, status, verification_sent_at, subscribed_at) VALUES
			('s1', 1, 'new', 'pending', datetime('now', '-1 hour'), datetime('now', '-1 hour')),
			('s2', 1, 'old', 'pending', datetime('now', '-40 days'), datetime('now', '-40 days')),
			('s3', 1, 'added', 'active', NULL, datetime('now', '-40 days')),
			('s4', 1, 'confirmed', 'active', datetime('now', '-40 days'), datetime('now', '-40 days')),
			('s5', 1, 'prospect', 'prospect', NULL, datetime('now', '-40 days')),
			('s6', 1, 'also_active', 'pending', datetime('now', '-40 days'), datetime('now', '-40 days')),
			('s7', 2, 'also_active', 'active', NULL, datetime('now', '-40 days'))`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err, q)
	}
	user, err := access.Active(ctx, store, "anne")
	require.NoError(t, err)
	asAdmin := access.WithBrand(access.WithUser(ctx, user), "org1")
	run := func(days int) (*types.HousekeepingResponse, error) {
		return NewHousekeepingUnconfirmedLogic(asAdmin, &svc.ServiceContext{DB: store}).HousekeepingUnconfirmed(&types.HousekeepingUnconfirmedRequest{OlderThanDays: days, DryRun: true})
	}

	got, err := run(0)
	require.NoError(t, err)
	assert.Equal(t, 2, got.AffectedCount)
	assert.Contains(t, got.Message, "older than 0 days")

	got, err = run(30)
	require.NoError(t, err)
	assert.Equal(t, 1, got.AffectedCount)

	_, err = run(-1)
	assert.Error(t, err)

	deleted, err := NewHousekeepingUnconfirmedLogic(asAdmin, &svc.ServiceContext{DB: store}).HousekeepingUnconfirmed(&types.HousekeepingUnconfirmedRequest{OlderThanDays: 30})
	require.NoError(t, err)
	assert.Equal(t, 1, deleted.AffectedCount)
	var left int
	require.NoError(t, conn.QueryRow(`SELECT COUNT(*) FROM contacts`).Scan(&left))
	assert.Equal(t, 6, left, "only the old pending signup went")
}
