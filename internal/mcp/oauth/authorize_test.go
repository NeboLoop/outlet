package oauth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/mcp/mcpauth"
	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

type authorizeFixture struct {
	t     *testing.T
	store *db.Store
	h     *Handler
}

func newAuthorizeFixture(t *testing.T) *authorizeFixture {
	t.Helper()
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn))
	store := db.NewStore(conn)
	ctx := context.Background()
	hash, err := bcrypt.GenerateFromPassword([]byte("right-password"), bcrypt.MinCost)
	require.NoError(t, err)
	_, err = store.CreateUser(ctx, db.CreateUserParams{ID: "u1", Email: "u1@example.com", PasswordHash: string(hash), Name: "U", Role: "super_admin", Status: "active", EmailVerified: int64(1)})
	require.NoError(t, err)
	_, err = store.CreateMCPOAuthClient(ctx, db.CreateMCPOAuthClientParams{ID: "c1", ClientID: "client", ClientSecretHash: "x", Name: "Client", RedirectUris: "https://client.test/cb"})
	require.NoError(t, err)
	return &authorizeFixture{t: t, store: store, h: NewHandler(&svc.ServiceContext{DB: store}, "https://outlet.test")}
}

func challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// submit posts the sign-in form and returns the status and body.
func (f *authorizeFixture) submit(form url.Values) (int, string) {
	f.t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	f.h.HandleAuthorizeSubmit(rr, req)
	return rr.Code, rr.Body.String()
}

func (f *authorizeFixture) codes() int {
	f.t.Helper()
	var n int
	require.NoError(f.t, f.store.GetDB().QueryRow(`SELECT COUNT(*) FROM mcp_oauth_codes`).Scan(&n))
	return n
}

func goodForm(password string) url.Values {
	return url.Values{
		"response_type": {"code"}, "client_id": {"client"}, "redirect_uri": {"https://client.test/cb"},
		"state": {"s"}, "code_challenge": {challenge("verifier")}, "code_challenge_method": {"S256"},
		"email": {"u1@example.com"}, "password": {password},
	}
}

// The sign-in form is checked like the sign-in page: the client's own
// redirect URI and an S256 PKCE challenge, or no code.
func TestAuthorizeSubmit_ChecksTheRequestAgain(t *testing.T) {
	f := newAuthorizeFixture(t)

	form := goodForm("right-password")
	form.Set("redirect_uri", "https://attacker.test/cb")
	code, _ := f.submit(form)
	assert.Equal(t, http.StatusBadRequest, code)

	form = goodForm("right-password")
	form.Del("code_challenge")
	f.submit(form)

	form = goodForm("right-password")
	form.Set("code_challenge_method", "plain")
	f.submit(form)
	assert.Equal(t, 0, f.codes())

	code, body := f.submit(goodForm("right-password"))
	assert.Equal(t, http.StatusOK, code)
	assert.Contains(t, body, "https://client.test/cb?code=")
	assert.Equal(t, 1, f.codes())
}

// The form uses the admin sign-in's check: five wrong passwords lock the
// account, and while it is locked even the right password gets no code.
func TestAuthorizeSubmit_LocksAfterWrongPasswords(t *testing.T) {
	f := newAuthorizeFixture(t)
	for i := 0; i < 5; i++ {
		_, body := f.submit(goodForm("guess"))
		assert.Contains(t, body, "Wrong email or password.")
	}
	_, body := f.submit(goodForm("right-password"))
	assert.Contains(t, body, "This account is locked")
	assert.Equal(t, 0, f.codes())
}

// An authorization code that expired earlier today is refused.
func TestTokenGrant_RefusesACodeThatExpiredToday(t *testing.T) {
	f := newAuthorizeFixture(t)
	_, err := f.store.CreateMCPOAuthCode(context.Background(), db.CreateMCPOAuthCodeParams{
		ID: "code1", ClientID: "c1", UserID: "u1", CodeHash: mcpauth.HashToken("the-code"), RedirectUri: "https://client.test/cb",
		Scopes: mcpauth.ScopeFull, CodeChallenge: sql.NullString{String: challenge("verifier"), Valid: true},
		CodeChallengeMethod: sql.NullString{String: "S256", Valid: true},
		ExpiresAt:           time.Now().Add(-time.Minute).UTC().Format(time.RFC3339),
	})
	require.NoError(t, err)
	form := url.Values{"grant_type": {"authorization_code"}, "code": {"the-code"}, "client_id": {"client"}, "redirect_uri": {"https://client.test/cb"}, "code_verifier": {"verifier"}}
	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	f.h.HandleToken(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid_grant")
}
