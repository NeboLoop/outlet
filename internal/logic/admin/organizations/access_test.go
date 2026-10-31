package organizations

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	adminauth "github.com/outlet-sh/outlet/internal/logic/admin/auth"
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

	// A new person gets a one-time link for the brand and role, by email
	// only: the inviter never sees it.
	resp, err := NewCreateInvitationLogic(f.as(t, "anne"), f.svc).CreateInvitation(&types.CreateInvitationRequest{Id: "alpha-id", Email: " New@Example.com ", Role: "member"})
	require.NoError(t, err)
	assert.Equal(t, "new@example.com", resp.Email)
	assert.False(t, resp.EmailSent, "no mail provider in tests; the response says so")
	assert.NotEmpty(t, resp.ExpiresAt)
	token := pendingToken(t, f, "alpha-id", "new@example.com")
	inv, err := f.svc.DB.GetOrgInvitationByToken(ctx, token)
	require.NoError(t, err)
	assert.Equal(t, "alpha-id", inv.OrgID)
	assert.Equal(t, "member", inv.Role)

	// A second invitation replaces the first.
	_, err = NewCreateInvitationLogic(f.as(t, "anne"), f.svc).CreateInvitation(&types.CreateInvitationRequest{Id: "alpha-id", Email: "new@example.com"})
	require.NoError(t, err)
	assert.NotEqual(t, token, pendingToken(t, f, "alpha-id", "new@example.com"))
	_, err = f.svc.DB.GetOrgInvitationByToken(ctx, token)
	assert.ErrorIs(t, err, sql.ErrNoRows)

	// Someone with an account gets an invitation too, and the answer looks
	// the same, so it doesn't say who has an account. They join only when
	// they accept it, signed in as the invited address.
	resp, err = NewCreateInvitationLogic(f.as(t, "root"), f.svc).CreateInvitation(&types.CreateInvitationRequest{Id: "bravo-id", Email: "mike@example.com", Role: "admin"})
	require.NoError(t, err)
	assert.Equal(t, "mike@example.com", resp.Email)
	_, err = access.Check(ctx, f.svc.DB, "mike", "bravo-id", access.Member)
	assert.Error(t, err, "not a member until he accepts")

	token = pendingToken(t, f, "bravo-id", "mike@example.com")
	_, err = adminauth.NewAcceptInvitationLogic(f.as(t, "anne"), f.svc).AcceptInvitation(&types.AcceptInvitationRequest{Token: token})
	assert.Equal(t, 403, status(err), "someone else can't use mike's invitation")
	_, err = adminauth.NewAcceptInvitationLogic(f.as(t, "mike"), f.svc).AcceptInvitation(&types.AcceptInvitationRequest{Token: token})
	require.NoError(t, err)
	_, err = access.Check(ctx, f.svc.DB, "mike", "bravo-id", access.Admin)
	assert.NoError(t, err)
	_, err = adminauth.NewAcceptInvitationLogic(f.as(t, "mike"), f.svc).AcceptInvitation(&types.AcceptInvitationRequest{Token: token})
	assert.Equal(t, 400, status(err), "an invitation works once")
}

// pendingToken reads the token of the open invitation for an address. In
// production it reaches the person only by email.
func pendingToken(t *testing.T, f restFixture, brandID, email string) string {
	t.Helper()
	var token string
	require.NoError(t, f.svc.DB.GetDB().QueryRow(`SELECT token FROM org_invitations WHERE org_id = ? AND email = ? AND accepted_at IS NULL`, brandID, email).Scan(&token))
	return token
}
