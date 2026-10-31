package organizations

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
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// The admin REST logic runs the same access check as MCP.

type restFixture struct {
	svc *svc.ServiceContext
}

func newRESTFixture(t *testing.T) restFixture {
	t.Helper()
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn))
	store := db.NewStore(conn)
	s := &svc.ServiceContext{DB: store, EmailService: email.NewService(store, nil)}
	s.Config.App.BaseURL = "https://outlet.test"

	ctx := context.Background()
	for _, u := range []struct{ id, role string }{{"root", "super_admin"}, {"anne", "agent"}, {"mike", "agent"}} {
		_, err := store.CreateUser(ctx, db.CreateUserParams{
			ID: u.id, Email: u.id + "@example.com", PasswordHash: "x", Name: u.id, Role: u.role, Status: "active", EmailVerified: int64(1),
		})
		require.NoError(t, err)
	}
	for _, b := range []string{"alpha", "bravo"} {
		_, err := store.CreateOrganization(ctx, db.CreateOrganizationParams{ID: b + "-id", Name: b, Slug: b, ApiKey: "key-" + b})
		require.NoError(t, err)
	}
	for _, m := range []struct{ user, role string }{{"anne", "admin"}, {"mike", "member"}} {
		require.NoError(t, store.AddUserToOrganization(ctx, db.AddUserToOrganizationParams{
			OrgID: "alpha-id", UserID: m.user, Role: sql.NullString{String: m.role, Valid: true},
		}))
	}
	return restFixture{svc: s}
}

// as returns a request context signed in as a person, the way the auth
// middleware leaves it.
func (f restFixture) as(t *testing.T, userID string) context.Context {
	t.Helper()
	user, err := access.Active(context.Background(), f.svc.DB, userID)
	require.NoError(t, err)
	return access.WithUser(context.Background(), user)
}

func status(err error) int {
	var ce *errorx.CodeError
	if errors.As(err, &ce) {
		return ce.Code
	}
	return 0
}

func TestListOrganizations_OnlyTheCallersBrands(t *testing.T) {
	f := newRESTFixture(t)

	resp, err := NewListOrganizationsLogic(f.as(t, "mike"), f.svc).ListOrganizations()
	require.NoError(t, err)
	require.Len(t, resp.Organizations, 1)
	assert.Equal(t, "alpha", resp.Organizations[0].Slug)
	assert.Empty(t, resp.Organizations[0].ApiKey, "a member doesn't see the brand's API key")

	resp, err = NewListOrganizationsLogic(f.as(t, "anne"), f.svc).ListOrganizations()
	require.NoError(t, err)
	require.Len(t, resp.Organizations, 1)
	assert.Equal(t, "key-alpha", resp.Organizations[0].ApiKey)

	resp, err = NewListOrganizationsLogic(f.as(t, "root"), f.svc).ListOrganizations()
	require.NoError(t, err)
	assert.Len(t, resp.Organizations, 2)
}

func TestOrganizationActions_CheckTheNamedBrand(t *testing.T) {
	f := newRESTFixture(t)

	// The request names bravo; anne doesn't belong to it.
	_, err := NewGetOrganizationLogic(f.as(t, "anne"), f.svc).GetOrganization(&types.GetOrgRequest{Id: "bravo-id"})
	assert.Equal(t, 403, status(err))
	_, err = NewGetOrganizationBySlugLogic(f.as(t, "anne"), f.svc).GetOrganizationBySlug(&types.GetOrgBySlugRequest{Slug: "bravo"})
	assert.Equal(t, 403, status(err))
	_, err = NewDeleteOrganizationLogic(f.as(t, "anne"), f.svc).DeleteOrganization(&types.DeleteOrgRequest{Id: "bravo-id"})
	assert.Equal(t, 403, status(err))

	// A member can't delete, rename or re-key its brand.
	_, err = NewDeleteOrganizationLogic(f.as(t, "mike"), f.svc).DeleteOrganization(&types.DeleteOrgRequest{Id: "alpha-id"})
	assert.Equal(t, 403, status(err))
	_, err = NewRegenerateOrgApiKeyLogic(f.as(t, "mike"), f.svc).RegenerateOrgApiKey(&types.RegenerateApiKeyRequest{Id: "alpha-id"})
	assert.Equal(t, 403, status(err))

	// Only a super admin makes brands.
	_, err = NewCreateOrganizationLogic(f.as(t, "anne"), f.svc).CreateOrganization(&types.CreateOrgRequest{Name: "Mine", Slug: "mine"})
	assert.Equal(t, 403, status(err))

	// The brand's admin and the super admin go ahead.
	_, err = NewGetOrganizationLogic(f.as(t, "mike"), f.svc).GetOrganization(&types.GetOrgRequest{Id: "alpha-id"})
	assert.NoError(t, err)
	_, err = NewDeleteOrganizationLogic(f.as(t, "root"), f.svc).DeleteOrganization(&types.DeleteOrgRequest{Id: "bravo-id"})
	assert.NoError(t, err)
}

func TestAllOrganizationsStats_OnlyTheCallersBrands(t *testing.T) {
	f := newRESTFixture(t)
	for _, b := range []string{"alpha-id", "bravo-id"} {
		_, err := f.svc.DB.CreateEmailList(context.Background(), db.CreateEmailListParams{PublicID: "p-" + b, OrgID: b, Name: "L", Slug: "l"})
		require.NoError(t, err)
	}
	resp, err := NewGetAllOrganizationsStatsLogic(f.as(t, "mike"), f.svc).GetAllOrganizationsStats()
	require.NoError(t, err)
	assert.Contains(t, resp.Stats, "alpha-id")
	assert.NotContains(t, resp.Stats, "bravo-id")
}

func TestCreateInvitation(t *testing.T) {
	f := newRESTFixture(t)
	ctx := context.Background()

	// Members and outsiders can't invite.
	_, err := NewCreateInvitationLogic(f.as(t, "mike"), f.svc).CreateInvitation(&types.CreateInvitationRequest{Id: "alpha-id", Email: "x@example.com"})
	assert.Equal(t, 403, status(err))
	_, err = NewCreateInvitationLogic(f.as(t, "anne"), f.svc).CreateInvitation(&types.CreateInvitationRequest{Id: "bravo-id", Email: "x@example.com"})
	assert.Equal(t, 403, status(err))
	_, err = NewCreateInvitationLogic(f.as(t, "anne"), f.svc).CreateInvitation(&types.CreateInvitationRequest{Id: "alpha-id", Email: "x@example.com", Role: "owner"})
	assert.Equal(t, 400, status(err))

	// A new person gets a one-time link for the brand and role.
	resp, err := NewCreateInvitationLogic(f.as(t, "anne"), f.svc).CreateInvitation(&types.CreateInvitationRequest{Id: "alpha-id", Email: " New@Example.com ", Role: "member"})
	require.NoError(t, err)
	assert.False(t, resp.Added)
	assert.Equal(t, "new@example.com", resp.Email)
	assert.False(t, resp.EmailSent, "no mail provider in tests; the response says so")
	require.Contains(t, resp.InviteUrl, "https://outlet.test/auth/register?token=")
	token := resp.InviteUrl[len("https://outlet.test/auth/register?token="):]
	inv, err := f.svc.DB.GetOrgInvitationByToken(ctx, token)
	require.NoError(t, err)
	assert.Equal(t, "alpha-id", inv.OrgID)
	assert.Equal(t, "member", inv.Role)

	// A second invitation replaces the first.
	resp2, err := NewCreateInvitationLogic(f.as(t, "anne"), f.svc).CreateInvitation(&types.CreateInvitationRequest{Id: "alpha-id", Email: "new@example.com"})
	require.NoError(t, err)
	assert.NotEqual(t, resp.InviteUrl, resp2.InviteUrl)
	_, err = f.svc.DB.GetOrgInvitationByToken(ctx, token)
	assert.ErrorIs(t, err, sql.ErrNoRows)

	// Someone with an account joins at once.
	resp, err = NewCreateInvitationLogic(f.as(t, "root"), f.svc).CreateInvitation(&types.CreateInvitationRequest{Id: "bravo-id", Email: "mike@example.com", Role: "admin"})
	require.NoError(t, err)
	assert.True(t, resp.Added)
	_, err = access.Check(ctx, f.svc.DB, "mike", "bravo-id", access.Admin)
	assert.NoError(t, err)
}
