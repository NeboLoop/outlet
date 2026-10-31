package middleware

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

const testAccessSecret = "test-access-secret"

// authFixture is a database with one active and one disabled person.
type authFixture struct {
	store    *db.Store
	mw       *AuthMiddleware
	active   string
	disabled string
}

func newAuthFixture(t *testing.T) authFixture {
	t.Helper()
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db"))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn))
	store := db.NewStore(conn)

	f := authFixture{store: store, mw: NewAuthMiddleware(testAccessSecret, store), active: uuid.NewString(), disabled: uuid.NewString()}
	for id, status := range map[string]string{f.active: "active", f.disabled: "inactive"} {
		_, err := store.CreateUser(context.Background(), db.CreateUserParams{
			ID: id, Email: id + "@example.com", PasswordHash: "x", Name: "Test", Role: "agent", Status: status, EmailVerified: int64(1),
		})
		require.NoError(t, err)
	}
	return f
}

func signedToken(t *testing.T, userID, secret string, ttl time.Duration) string {
	t.Helper()
	token, err := utils.GenerateToken(userID, userID+"@example.com", "agent", secret, ttl)
	require.NoError(t, err)
	return token
}

// serve runs the middleware once and reports the status, the error message,
// and the person the handler saw.
func (f authFixture) serve(t *testing.T, req *http.Request) (int, string, *db.User, context.Context) {
	t.Helper()
	var seen *db.User
	var seenCtx context.Context
	handler := f.mw.Handle(func(w http.ResponseWriter, r *http.Request) {
		if u, ok := access.User(r.Context()); ok {
			seen = &u
		}
		seenCtx = r.Context()
		w.WriteHeader(http.StatusOK)
	})
	rr := httptest.NewRecorder()
	handler(rr, req)
	var body map[string]string
	_ = json.NewDecoder(rr.Body).Decode(&body)
	return rr.Code, body["error"], seen, seenCtx
}

func bearer(token string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/admin/organizations", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func TestAuthMiddleware_ValidTokenSignsTheUserIn(t *testing.T) {
	f := newAuthFixture(t)
	code, _, user, _ := f.serve(t, bearer(signedToken(t, f.active, testAccessSecret, time.Hour)))
	assert.Equal(t, http.StatusOK, code)
	require.NotNil(t, user)
	assert.Equal(t, f.active, user.ID)
	assert.Equal(t, "active", user.Status)
}

func TestAuthMiddleware_ValidTokenFromCookie(t *testing.T) {
	f := newAuthFixture(t)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: signedToken(t, f.active, testAccessSecret, time.Hour)})
	code, _, user, _ := f.serve(t, req)
	assert.Equal(t, http.StatusOK, code)
	require.NotNil(t, user)
	assert.Equal(t, f.active, user.ID)
}

// The middleware used to read tokens without checking their signature, so
// anyone could sign one with any secret and act as any user.
func TestAuthMiddleware_RefusesForgedToken(t *testing.T) {
	f := newAuthFixture(t)
	code, msg, user, _ := f.serve(t, bearer(signedToken(t, f.active, "some-other-secret", time.Hour)))
	assert.Equal(t, http.StatusUnauthorized, code)
	assert.Nil(t, user)
	assert.NotEmpty(t, msg)

	// An unsigned token ("alg: none") is refused too.
	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"user_id": f.active, "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)
	code, _, _, _ = f.serve(t, bearer(unsigned))
	assert.Equal(t, http.StatusUnauthorized, code)
}

func TestAuthMiddleware_RefusesExpiredToken(t *testing.T) {
	f := newAuthFixture(t)
	code, _, _, _ := f.serve(t, bearer(signedToken(t, f.active, testAccessSecret, -time.Minute)))
	assert.Equal(t, http.StatusUnauthorized, code)
}

func TestAuthMiddleware_RefusesDisabledOrDeletedAccount(t *testing.T) {
	f := newAuthFixture(t)
	code, msg, _, _ := f.serve(t, bearer(signedToken(t, f.disabled, testAccessSecret, time.Hour)))
	assert.Equal(t, http.StatusUnauthorized, code)
	assert.Equal(t, "This account is not active.", msg)

	code, msg, _, _ = f.serve(t, bearer(signedToken(t, uuid.NewString(), testAccessSecret, time.Hour)))
	assert.Equal(t, http.StatusUnauthorized, code)
	assert.Equal(t, "This account no longer exists.", msg)

	// Disabling a signed-in person takes effect on their next request.
	token := signedToken(t, f.active, testAccessSecret, time.Hour)
	code, _, _, _ = f.serve(t, bearer(token))
	require.Equal(t, http.StatusOK, code)
	_, err := f.store.GetDB().Exec(`UPDATE users SET status = 'inactive' WHERE id = ?`, f.active)
	require.NoError(t, err)
	code, _, _, _ = f.serve(t, bearer(token))
	assert.Equal(t, http.StatusUnauthorized, code)
}

func TestAuthMiddleware_RefusesWithoutSecret(t *testing.T) {
	f := newAuthFixture(t)
	f.mw = NewAuthMiddleware("", f.store)
	code, _, _, _ := f.serve(t, bearer(signedToken(t, f.active, testAccessSecret, time.Hour)))
	assert.Equal(t, http.StatusUnauthorized, code)
}

func TestAuthMiddleware_MissingOrMalformedHeader(t *testing.T) {
	f := newAuthFixture(t)

	code, msg, _, _ := f.serve(t, httptest.NewRequest(http.MethodGet, "/test", nil))
	assert.Equal(t, http.StatusUnauthorized, code)
	assert.Equal(t, "No authentication token provided", msg)

	for _, header := range []string{"Basic abc", "Bearer", "Bearer a b", "bearer token"} {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", header)
		code, msg, _, _ := f.serve(t, req)
		assert.Equal(t, http.StatusUnauthorized, code, header)
		assert.Equal(t, "Invalid authorization header format", msg, header)
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer not-a-jwt")
	code, _, _, _ = f.serve(t, req)
	assert.Equal(t, http.StatusUnauthorized, code)
}

// X-Org-Id names the brand; access.RequireBrand decides whether the person
// may act in it.
func TestAuthMiddleware_OrgHeaderNamesTheBrand(t *testing.T) {
	f := newAuthFixture(t)
	brandID := uuid.NewString()
	_, err := f.store.CreateOrganization(context.Background(), db.CreateOrganizationParams{ID: brandID, Name: "B", Slug: "b", ApiKey: "k"})
	require.NoError(t, err)
	require.NoError(t, f.store.AddUserToOrganization(context.Background(), db.AddUserToOrganizationParams{
		OrgID: brandID, UserID: f.active, Role: sql.NullString{String: "member", Valid: true},
	}))
	token := signedToken(t, f.active, testAccessSecret, time.Hour)

	req := bearer(token)
	req.Header.Set("X-Org-Id", brandID)
	code, _, _, ctx := f.serve(t, req)
	require.Equal(t, http.StatusOK, code)
	got, err := access.RequireBrand(ctx, f.store, access.Member)
	require.NoError(t, err)
	assert.Equal(t, brandID, got)

	// A brand the person doesn't belong to is refused by the check.
	req = bearer(token)
	req.Header.Set("X-Org-Id", uuid.NewString())
	_, _, _, ctx = f.serve(t, req)
	_, err = access.RequireBrand(ctx, f.store, access.Member)
	assert.Error(t, err)

	// A header that isn't an ID names no brand.
	req = bearer(token)
	req.Header.Set("X-Org-Id", "not-a-uuid")
	_, _, _, ctx = f.serve(t, req)
	_, err = access.RequireBrand(ctx, f.store, access.Member)
	assert.Error(t, err)
}
