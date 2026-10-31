package access_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/errorx"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func newStore(t *testing.T) *db.Store {
	t.Helper()
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn))
	return db.NewStore(conn)
}

func addUser(t *testing.T, s *db.Store, id, role, status string) {
	t.Helper()
	_, err := s.CreateUser(context.Background(), db.CreateUserParams{
		ID: id, Email: id + "@example.com", PasswordHash: "x", Name: id, Role: role, Status: status, EmailVerified: int64(1),
	})
	require.NoError(t, err)
}

func addBrand(t *testing.T, s *db.Store, id string) {
	t.Helper()
	_, err := s.CreateOrganization(context.Background(), db.CreateOrganizationParams{ID: id, Name: id, Slug: id, ApiKey: "key-" + id})
	require.NoError(t, err)
}

func join(t *testing.T, s *db.Store, brand, user, role string) {
	t.Helper()
	require.NoError(t, s.AddUserToOrganization(context.Background(), db.AddUserToOrganizationParams{
		OrgID: brand, UserID: user, Role: sql.NullString{String: role, Valid: true},
	}))
}

func code(err error) int {
	var ce *errorx.CodeError
	if errors.As(err, &ce) {
		return ce.Code
	}
	return 0
}

func TestCheck(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	addBrand(t, s, "brand-a")
	addBrand(t, s, "brand-b")
	addUser(t, s, "root", "super_admin", "active")
	addUser(t, s, "anne", "agent", "active")   // admin of brand-a
	addUser(t, s, "mike", "admin", "active")   // member of brand-a; the platform "admin" role grants nothing
	addUser(t, s, "olga", "agent", "inactive") // admin of brand-a, but disabled
	join(t, s, "brand-a", "anne", "admin")
	join(t, s, "brand-a", "mike", "member")
	join(t, s, "brand-a", "olga", "admin")

	cases := []struct {
		name  string
		user  string
		brand string
		need  access.Role
		code  int // 0 = allowed
	}{
		{"super admin reaches every brand", "root", "brand-b", access.Admin, 0},
		{"super admin takes platform actions", "root", "", access.SuperAdmin, 0},
		{"brand admin acts as admin in its brand", "anne", "brand-a", access.Admin, 0},
		{"brand admin acts as member in its brand", "anne", "brand-a", access.Member, 0},
		{"brand admin can't reach another brand", "anne", "brand-b", access.Member, 403},
		{"brand admin can't take platform actions", "anne", "", access.SuperAdmin, 403},
		{"no brand means platform", "anne", "", access.Member, 403},
		{"member reads its brand", "mike", "brand-a", access.Member, 0},
		{"member can't take admin actions", "mike", "brand-a", access.Admin, 403},
		{"platform admin role reaches no other brand", "mike", "brand-b", access.Member, 403},
		{"disabled person is refused everywhere", "olga", "brand-a", access.Member, 401},
		{"unknown person is refused", "nobody", "brand-a", access.Member, 401},
		{"no person is refused", "", "brand-a", access.Member, 401},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			caller, err := access.Check(ctx, s, tc.user, tc.brand, tc.need)
			if tc.code == 0 {
				require.NoError(t, err)
				assert.Equal(t, tc.user, caller.User.ID)
				return
			}
			require.Error(t, err)
			assert.Equal(t, tc.code, code(err), err.Error())
		})
	}

	// Status is read on every check: disabling takes effect at once.
	_, err := access.Check(ctx, s, "anne", "brand-a", access.Admin)
	require.NoError(t, err)
	_, err = s.GetDB().Exec(`UPDATE users SET status = 'inactive' WHERE id = 'anne'`)
	require.NoError(t, err)
	_, err = access.Check(ctx, s, "anne", "brand-a", access.Admin)
	assert.Equal(t, 401, code(err))
}

func TestBrands(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	addBrand(t, s, "brand-a")
	addBrand(t, s, "brand-b")
	addUser(t, s, "root", "super_admin", "active")
	addUser(t, s, "mike", "agent", "active")
	join(t, s, "brand-b", "mike", "member")

	all, err := access.Brands(ctx, s, "root")
	require.NoError(t, err)
	assert.Len(t, all, 2)

	mine, err := access.Brands(ctx, s, "mike")
	require.NoError(t, err)
	require.Len(t, mine, 1)
	assert.Equal(t, "brand-b", mine[0].ID)
	assert.Equal(t, access.Member, mine[0].Role)
}

func TestRequireBrand(t *testing.T) {
	s := newStore(t)
	addBrand(t, s, "brand-a")
	addUser(t, s, "mike", "agent", "active")
	join(t, s, "brand-a", "mike", "member")
	user, err := access.Active(context.Background(), s, "mike")
	require.NoError(t, err)

	// No one signed in.
	_, err = access.RequireBrand(context.Background(), s, access.Member)
	assert.Equal(t, 401, code(err))

	ctx := access.WithUser(context.Background(), user)
	// Signed in, but the request names no brand.
	_, err = access.RequireBrand(ctx, s, access.Member)
	assert.Equal(t, 400, code(err))

	brandID, err := access.RequireBrand(access.WithBrand(ctx, "brand-a"), s, access.Member)
	require.NoError(t, err)
	assert.Equal(t, "brand-a", brandID)

	// Naming a brand grants nothing.
	_, err = access.RequireBrand(access.WithBrand(ctx, "brand-b"), s, access.Member)
	assert.Equal(t, 403, code(err))
	_, err = access.RequireBrand(access.WithBrand(ctx, "brand-a"), s, access.Admin)
	assert.Equal(t, 403, code(err))
}
