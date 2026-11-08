package sequences

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

// A sequence starts active unless it is created paused, as the create form's
// Active switch says.
func TestCreateSequence_Active(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	for _, q := range []string{
		`INSERT INTO users (id, email, password_hash, name, role, status, email_verified) VALUES ('mike', 'mike@example.com', 'x', 'M', 'agent', 'active', 1)`,
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('alpha-id', 'alpha', 'alpha', 'k-a')`,
		`INSERT INTO org_users (org_id, user_id, role) VALUES ('alpha-id', 'mike', 'admin')`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err, q)
	}
	store := db.NewStore(conn)
	user, err := access.Active(context.Background(), store, "mike")
	require.NoError(t, err)
	ctx := access.WithBrand(access.WithUser(context.Background(), user), "alpha-id")
	create := func(slug string, active *bool) *types.SequenceInfo {
		got, err := NewCreateSequenceLogic(ctx, &svc.ServiceContext{DB: store}).CreateSequence(&types.CreateSequenceRequest{
			Slug: slug, Name: slug, TriggerEvent: "manual", SequenceType: "lifecycle", IsActive: active,
		})
		require.NoError(t, err)
		return got
	}
	off := false
	assert.True(t, create("default", nil).IsActive, "omitted starts it active")
	assert.False(t, create("paused", &off).IsActive)
	assert.Nil(t, create("timing", nil).SendHour, "no fixed send hour: each email goes its delay after joining")
}
