package users

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

// With no super admin left, the first-run setup would make the next visitor
// one. The last active super admin can't be deleted.
func TestDeleteUser_KeepsTheLastSuperAdmin(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn))
	store := db.NewStore(conn)
	ctx := context.Background()
	for _, id := range []string{"root", "second"} {
		_, err := store.CreateUser(ctx, db.CreateUserParams{ID: id, Email: id + "@example.com", PasswordHash: "x", Name: id, Role: "super_admin", Status: "active", EmailVerified: int64(1)})
		require.NoError(t, err)
	}
	user, err := access.Active(ctx, store, "root")
	require.NoError(t, err)
	asRoot := access.WithUser(ctx, user)
	s := &svc.ServiceContext{DB: store}

	_, err = NewDeleteUserLogic(asRoot, s).DeleteUser(&types.DeleteUserRequest{Id: "second"})
	require.NoError(t, err)

	_, err = NewDeleteUserLogic(asRoot, s).DeleteUser(&types.DeleteUserRequest{Id: "root"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "last super admin")
	_, err = store.GetUserByID(ctx, "root")
	assert.NoError(t, err, "root is still there")

	_, err = NewDeleteUserLogic(asRoot, s).DeleteUser(&types.DeleteUserRequest{Id: "nobody"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
