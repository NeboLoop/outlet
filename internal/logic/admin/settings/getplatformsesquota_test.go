package settings

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// A platform that sends through SMTP has no SES quota, and that's no error.
func TestPlatformSESQuota_NotConfiguredIsNoError(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn))
	store := db.NewStore(conn)
	ctx := context.Background()
	_, err = store.CreateUser(ctx, db.CreateUserParams{ID: "root", Email: "root@example.com", PasswordHash: "x", Name: "Root", Role: "super_admin", Status: "active", EmailVerified: int64(1)})
	require.NoError(t, err)
	user, err := access.Active(ctx, store, "root")
	require.NoError(t, err)

	got, err := NewGetPlatformSESQuotaLogic(access.WithUser(ctx, user), &svc.ServiceContext{DB: store}).GetPlatformSESQuota()
	require.NoError(t, err)
	assert.False(t, got.Configured)
}
