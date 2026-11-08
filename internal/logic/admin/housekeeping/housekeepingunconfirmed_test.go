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

// 0 days means every unconfirmed contact, not 30 days.
func TestHousekeepingUnconfirmed_TakesTheDaysAsGiven(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn))
	store := db.NewStore(conn)
	ctx := context.Background()
	for _, q := range []string{
		`INSERT INTO users (id, email, password_hash, name, role, status) VALUES ('anne', 'anne@example.com', 'x', 'Anne', 'agent', 'active')`,
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('org1', 'One', 'one', 'k1')`,
		`INSERT INTO org_users (org_id, user_id, role) VALUES ('org1', 'anne', 'admin')`,
		`INSERT INTO contacts (id, org_id, name, email, created_at) VALUES
			('new', 'org1', 'N', 'n@example.com', datetime('now', '-1 hour')),
			('old', 'org1', 'O', 'o@example.com', datetime('now', '-40 days'))`,
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
}
