package auth

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/config"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

func newAuthSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn))
	store := db.NewStore(conn)
	var c config.Config
	c.Auth.AccessSecret = "access-secret"
	c.Auth.RefreshSecret = "refresh-secret"
	c.Auth.AccessExpire = 3600
	c.Auth.RefreshExpire = 7200
	_, err = store.CreateOrganization(context.Background(), db.CreateOrganizationParams{ID: "alpha-id", Name: "Alpha", Slug: "alpha", ApiKey: "k"})
	require.NoError(t, err)
	return &svc.ServiceContext{Config: c, DB: store}
}

func invite(t *testing.T, s *svc.ServiceContext, token, email, role string) {
	t.Helper()
	_, err := s.DB.CreateOrgInvitation(context.Background(), db.CreateOrgInvitationParams{
		ID: "inv-" + token, OrgID: "alpha-id", Email: email, Role: sql.NullString{String: role, Valid: true}, Token: token,
	})
	require.NoError(t, err)
}

func register(s *svc.ServiceContext, token string) (*types.RegisterResponse, error) {
	return NewRegisterLogic(context.Background(), s).Register(&types.RegisterRequest{
		Token: token, Password: "a-long-password", FirstName: "New", LastName: "Person",
	})
}

func TestRegister_RequiresAnInvitation(t *testing.T) {
	s := newAuthSvc(t)
	_, err := register(s, "")
	require.Error(t, err)
	assert.Equal(t, "Outlet is invite-only. Ask a brand admin for an invitation.", err.Error())

	_, err = register(s, "made-up")
	require.Error(t, err)
	assert.Equal(t, "This invitation isn't valid.", err.Error())

	n, err := s.DB.CheckEmailExists(context.Background(), "new@example.com")
	require.NoError(t, err)
	assert.Zero(t, n)
}

func TestRegister_InvitationDecidesBrandAndRoleAndWorksOnce(t *testing.T) {
	s := newAuthSvc(t)
	ctx := context.Background()
	invite(t, s, "tok", "New@Example.com", "admin")

	resp, err := register(s, "tok")
	require.NoError(t, err)
	assert.Equal(t, "new@example.com", resp.User.Email)
	claims, err := utils.ValidateToken(resp.Token, "access-secret")
	require.NoError(t, err)
	assert.Equal(t, resp.User.Id, claims.UserID)

	// Active at once, a member of the invited brand with the invited role,
	// and nothing more.
	caller, err := access.Check(ctx, s.DB, resp.User.Id, "alpha-id", access.Admin)
	require.NoError(t, err)
	assert.Equal(t, access.Admin, caller.Role)
	_, err = access.Check(ctx, s.DB, resp.User.Id, "", access.SuperAdmin)
	assert.Error(t, err)

	// The invitation is spent.
	_, err = register(s, "tok")
	require.Error(t, err)
	assert.Equal(t, "This invitation was already used.", err.Error())
}

func TestRegister_ExpiredInvitation(t *testing.T) {
	s := newAuthSvc(t)
	invite(t, s, "old", "late@example.com", "member")
	_, err := s.DB.GetDB().Exec(`UPDATE org_invitations SET expires_at = datetime('now', '-1 minute') WHERE token = 'old'`)
	require.NoError(t, err)

	_, err = register(s, "old")
	require.Error(t, err)
	assert.Equal(t, "This invitation has expired. Ask for a new one.", err.Error())
}

func TestValidateInvitation(t *testing.T) {
	s := newAuthSvc(t)
	invite(t, s, "tok", "new@example.com", "member")
	l := NewValidateInvitationLogic(context.Background(), s)

	resp, err := l.ValidateInvitation(&types.ValidateInvitationRequest{Token: "tok"})
	require.NoError(t, err)
	assert.True(t, resp.Valid)
	assert.Equal(t, "new@example.com", resp.Email)
	assert.Equal(t, "member", resp.Role)
	assert.Equal(t, "Alpha", resp.BrandName)

	resp, err = l.ValidateInvitation(&types.ValidateInvitationRequest{Token: "nope"})
	require.NoError(t, err)
	assert.False(t, resp.Valid)
	assert.Equal(t, "This invitation isn't valid.", resp.Message)
}

func TestLoginAndRefresh_RefuseDisabledAccount(t *testing.T) {
	s := newAuthSvc(t)
	ctx := context.Background()
	hash, err := bcrypt.GenerateFromPassword([]byte("a-long-password"), bcrypt.MinCost)
	require.NoError(t, err)
	_, err = s.DB.CreateUser(ctx, db.CreateUserParams{
		ID: "u1", Email: "u1@example.com", PasswordHash: string(hash), Name: "U", Role: "agent", Status: "active", EmailVerified: int64(1),
	})
	require.NoError(t, err)

	login, err := NewLoginLogic(ctx, s).Login(&types.LoginRequest{Email: "u1@example.com", Password: "a-long-password"})
	require.NoError(t, err)

	_, err = s.DB.GetDB().Exec(`UPDATE users SET status = 'inactive' WHERE id = 'u1'`)
	require.NoError(t, err)

	_, err = NewLoginLogic(ctx, s).Login(&types.LoginRequest{Email: "u1@example.com", Password: "a-long-password"})
	require.Error(t, err)
	assert.Equal(t, "This account is not active.", err.Error())

	_, err = NewRefreshTokenLogic(ctx, s).RefreshToken(&types.RefreshTokenRequest{RefreshToken: login.RefreshToken})
	require.Error(t, err)
	assert.Equal(t, "This account is not active.", err.Error())

}
