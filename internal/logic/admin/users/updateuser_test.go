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

// A super admin adds people with the roles the platform has, saves their
// email and role, and can make someone else a super admin; the last active
// super admin keeps the role.
func TestUsers_RolesAndEdits(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	store := db.NewStore(conn)
	ctx := context.Background()
	_, err = store.CreateUser(ctx, db.CreateUserParams{ID: "root", Email: "root@example.com", PasswordHash: "x", Name: "Root", Role: "super_admin", Status: "active", EmailVerified: int64(1)})
	require.NoError(t, err)
	user, err := access.Active(ctx, store, "root")
	require.NoError(t, err)
	asRoot := access.WithUser(ctx, user)
	s := &svc.ServiceContext{DB: store}

	// Every role the screen offers can be created; others are refused
	for _, role := range []string{access.PlatformUser, "super_admin"} {
		_, err := NewCreateUserLogic(asRoot, s).CreateUser(&types.CreateUserRequest{Email: role + "@example.com", Password: "pw-123456", Name: role, Role: role})
		require.NoError(t, err, role)
	}
	_, err = NewCreateUserLogic(asRoot, s).CreateUser(&types.CreateUserRequest{Email: "v@example.com", Password: "pw-123456", Name: "V", Role: "viewer"})
	assert.Error(t, err)

	created, err := NewCreateUserLogic(asRoot, s).CreateUser(&types.CreateUserRequest{Email: "ann@example.com", Password: "pw-123456", Name: "Ann"})
	require.NoError(t, err)
	assert.Equal(t, access.PlatformUser, created.User.Role)
	id := created.User.Id

	// Edit saves name, email and role, including granting super admin
	got, err := NewUpdateUserLogic(asRoot, s).UpdateUser(&types.UpdateUserRequest{Id: id, Name: "Ann B", Email: "ann.b@example.com", Role: "super_admin"})
	require.NoError(t, err)
	assert.Equal(t, "Ann B", got.Name)
	assert.Equal(t, "ann.b@example.com", got.Email)
	assert.Equal(t, "super_admin", got.Role)

	_, err = NewUpdateUserLogic(asRoot, s).UpdateUser(&types.UpdateUserRequest{Id: id, Email: "root@example.com"})
	assert.ErrorContains(t, err, "Another user")
	_, err = NewUpdateUserLogic(asRoot, s).UpdateUser(&types.UpdateUserRequest{Id: id, Email: "not an email"})
	assert.ErrorContains(t, err, "valid email")
	_, err = NewUpdateUserLogic(asRoot, s).UpdateUser(&types.UpdateUserRequest{Id: id, Role: "manager"})
	assert.Error(t, err)

	// With several super admins one can step down; the last one can't
	list, err := store.ListUsers(ctx, db.ListUsersParams{FilterRole: "super_admin", PageSize: 10})
	require.NoError(t, err)
	for _, u := range list {
		if u.ID == "root" {
			continue
		}
		_, err := NewUpdateUserLogic(asRoot, s).UpdateUser(&types.UpdateUserRequest{Id: u.ID, Role: access.PlatformUser})
		require.NoError(t, err, u.Email)
	}
	_, err = NewUpdateUserLogic(asRoot, s).UpdateUser(&types.UpdateUserRequest{Id: "root", Role: access.PlatformUser})
	assert.ErrorContains(t, err, "last super admin")
	root, err := store.GetUserByID(ctx, "root")
	require.NoError(t, err)
	assert.Equal(t, "super_admin", root.Role)

	// A person who isn't a super admin can't change anyone, themselves included
	ann, err := access.Active(ctx, store, id)
	require.NoError(t, err)
	_, err = NewUpdateUserLogic(access.WithUser(ctx, ann), s).UpdateUser(&types.UpdateUserRequest{Id: id, Role: "super_admin"})
	assert.Error(t, err)
}
