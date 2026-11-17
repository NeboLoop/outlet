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
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/logic/auth"
	"github.com/outlet-sh/outlet/internal/mcp/mcpauth"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, migrations.Run(conn, nil))
	store := db.NewStore(conn)
	ctx := context.Background()
	_, err = store.CreateUser(ctx, db.CreateUserParams{ID: "u1", Email: "u1@example.com", PasswordHash: "x", Name: "U", Role: "super_admin", Status: "active", EmailVerified: int64(1)})
	require.NoError(t, err)
	_, err = store.CreateMCPOAuthClient(ctx, db.CreateMCPOAuthClientParams{ID: "c1", ClientID: "client", ClientSecretHash: "x", Name: "Client", RedirectUris: "https://client.test/cb"})
	require.NoError(t, err)
	s := &svc.ServiceContext{DB: store, EmailService: email.NewService(store, nil)}
	return &authorizeFixture{t: t, store: store, h: NewHandler(s, "https://outlet.test")}
}

func challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

var requestField = regexp.MustCompile(`name="request" value="([^"]+)"`)

// start opens the sign-in page as an MCP client sends a person to it, and
// returns the page, the handle of the sign-in it started and the browser's
// cookie.
func (f *authorizeFixture) start() (body, handle string, cookie *http.Cookie) {
	f.t.Helper()
	q := url.Values{
		"response_type": {"code"}, "client_id": {"client"}, "redirect_uri": {"https://client.test/cb"},
		"state": {"s"}, "code_challenge": {challenge("verifier")}, "code_challenge_method": {"S256"},
	}
	rr := httptest.NewRecorder()
	f.h.HandleAuthorize(rr, httptest.NewRequest(http.MethodGet, "/authorize?"+q.Encode(), nil))
	require.Equal(f.t, http.StatusOK, rr.Code, rr.Body.String())
	m := requestField.FindStringSubmatch(rr.Body.String())
	require.NotNil(f.t, m, "the page carries the sign-in's handle")
	cookies := rr.Result().Cookies()
	require.Len(f.t, cookies, 1)
	return rr.Body.String(), m[1], cookies[0]
}

// post sends a form to a sign-in page handler, from the browser with cookie
// (none when nil), and returns the status and body.
func (f *authorizeFixture) post(handler http.HandlerFunc, form url.Values, cookie *http.Cookie) (int, string) {
	f.t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()
	handler(rr, req)
	return rr.Code, rr.Body.String()
}

// emailed submits the email page and returns the link token and code the
// email carries. The email itself goes nowhere in a test, so a new sign-in
// is issued, as the email page's resend does; it retires the one just sent.
func (f *authorizeFixture) emailed(handle, address string) (token, code string) {
	f.t.Helper()
	status, body := f.post(f.h.HandleAuthorizeSubmit, url.Values{"request": {handle}, "email": {address}}, nil)
	require.Equal(f.t, http.StatusOK, status)
	require.Contains(f.t, body, auth.SignInEmailSent)
	require.Contains(f.t, body, `action="/mcp/oauth/authorize/code"`)
	token, code, err := auth.IssueSignIn(context.Background(), f.store, "", "u1")
	require.NoError(f.t, err)
	return token, code
}

func (f *authorizeFixture) count(table string) int {
	f.t.Helper()
	var n int
	require.NoError(f.t, f.store.GetDB().QueryRow(`SELECT COUNT(*) FROM `+table).Scan(&n))
	return n
}

func (f *authorizeFixture) codes() int { return f.count("mcp_oauth_codes") }

var authCode = regexp.MustCompile(`https://client\.test/cb\?code=([A-Za-z0-9_-]+)`)

// redeemed takes the authorization code from a success page and exchanges
// it at /token with the PKCE verifier, returning the token response status.
func (f *authorizeFixture) redeemed(body, verifier string) int {
	f.t.Helper()
	m := authCode.FindStringSubmatch(body)
	require.NotNil(f.t, m, "the success page sends the browser back with a code:\n%s", body)
	form := url.Values{"grant_type": {"authorization_code"}, "code": {m[1]}, "client_id": {"client"}, "redirect_uri": {"https://client.test/cb"}, "code_verifier": {verifier}}
	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	f.h.HandleToken(rr, req)
	if rr.Code == http.StatusOK {
		assert.Contains(f.t, rr.Body.String(), `"access_token"`)
	}
	return rr.Code
}

// openLink opens the emailed link in the browser with cookie (none when nil).
func (f *authorizeFixture) openLink(handle, token string, cookie *http.Cookie) string {
	f.t.Helper()
	u, err := url.Parse(f.h.signInLink(handle, token))
	require.NoError(f.t, err)
	req := httptest.NewRequest(http.MethodGet, u.RequestURI(), nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()
	f.h.HandleAuthorizeLink(rr, req)
	require.Equal(f.t, http.StatusOK, rr.Code)
	return rr.Body.String()
}

// The page asks only for an email, and the emailed code finishes the
// sign-in with an authorization code /token accepts with the PKCE verifier.
func TestEmailSignIn_CodeFinishes(t *testing.T) {
	f := newAuthorizeFixture(t)
	page, handle, cookie := f.start()
	assert.Contains(t, page, `type="email"`)
	assert.NotContains(t, page, "password", "the MCP page has no password path")
	assert.NotContains(t, page, "code_challenge", "the OAuth request stays on the server")
	assert.True(t, cookie.HttpOnly)
	assert.True(t, cookie.Secure)

	_, code := f.emailed(handle, "u1@example.com")
	status, body := f.post(f.h.HandleAuthorizeCode, url.Values{"request": {handle}, "code": {code}}, nil)
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, "state=s")
	assert.Equal(t, http.StatusOK, f.redeemed(body, "verifier"))
	assert.Equal(t, 0, f.count("mcp_oauth_requests"), "the sign-in finishes once")

	status, _ = f.post(f.h.HandleAuthorizeCode, url.Values{"request": {handle}, "code": {code}}, nil)
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, 1, f.codes())
}

// An address with no account gets the same page as one with an account.
func TestEmailSignIn_SameAnswerForUnknownEmail(t *testing.T) {
	f := newAuthorizeFixture(t)
	_, handle, _ := f.start()
	status, body := f.post(f.h.HandleAuthorizeSubmit, url.Values{"request": {handle}, "email": {"nobody@example.com"}}, nil)
	assert.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, auth.SignInEmailSent)
	assert.Contains(t, body, `action="/mcp/oauth/authorize/code"`)
	assert.Equal(t, 0, f.count("sign_in_codes"))
}

func TestEmailSignIn_WrongCodeRefused(t *testing.T) {
	f := newAuthorizeFixture(t)
	_, handle, _ := f.start()
	_, code := f.emailed(handle, "u1@example.com")
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	status, body := f.post(f.h.HandleAuthorizeCode, url.Values{"request": {handle}, "code": {wrong}}, nil)
	assert.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, "wrong or has expired")
	assert.Equal(t, 0, f.codes())
}

func TestEmailSignIn_ExpiredRefused(t *testing.T) {
	f := newAuthorizeFixture(t)
	_, handle, _ := f.start()
	_, code := f.emailed(handle, "u1@example.com")
	_, err := f.store.GetDB().Exec(`UPDATE sign_in_codes SET expires_at = datetime('now', '-1 minute')`)
	require.NoError(t, err)
	_, body := f.post(f.h.HandleAuthorizeCode, url.Values{"request": {handle}, "code": {code}}, nil)
	assert.Contains(t, body, "wrong or has expired", "an expired code is refused")

	_, handle, _ = f.start()
	_, code = f.emailed(handle, "u1@example.com")
	_, err = f.store.GetDB().Exec(`UPDATE mcp_oauth_requests SET expires_at = datetime('now', '-1 minute')`)
	require.NoError(t, err)
	status, body := f.post(f.h.HandleAuthorizeCode, url.Values{"request": {handle}, "code": {code}}, nil)
	assert.Equal(t, http.StatusBadRequest, status, "an expired sign-in is refused")
	assert.Contains(t, body, "This sign-in has expired")
	assert.Equal(t, 0, f.codes())
}

// The code page carries only the sign-in's handle: OAuth values posted with
// it are ignored, and the kept request is checked again when it finishes.
func TestEmailSignIn_TamperedRequestRefused(t *testing.T) {
	f := newAuthorizeFixture(t)

	// Values posted with the code don't change the request.
	_, handle, _ := f.start()
	_, code := f.emailed(handle, "u1@example.com")
	status, body := f.post(f.h.HandleAuthorizeCode, url.Values{
		"request": {handle}, "code": {code},
		"client_id": {"other"}, "redirect_uri": {"https://attacker.test/cb"},
		"code_challenge": {challenge("attacker")}, "code_challenge_method": {"plain"},
	}, nil)
	require.Equal(t, http.StatusOK, status)
	assert.NotContains(t, body, "attacker.test")
	assert.Equal(t, http.StatusBadRequest, f.redeemed(body, "attacker"), "the posted challenge is not the code's")

	// A client whose redirect URI changed since the sign-in started gets no
	// code, and the person's emailed code is not used up.
	_, handle, _ = f.start()
	_, code = f.emailed(handle, "u1@example.com")
	_, err := f.store.GetDB().Exec(`UPDATE mcp_oauth_clients SET redirect_uris = 'https://attacker.test/cb'`)
	require.NoError(t, err)
	status, body = f.post(f.h.HandleAuthorizeCode, url.Values{"request": {handle}, "code": {code}}, nil)
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, body, "Invalid redirect_uri")
	assert.Equal(t, 1, f.count("sign_in_codes"))

	// A client turned off since gets none either.
	_, err = f.store.GetDB().Exec(`UPDATE mcp_oauth_clients SET redirect_uris = 'https://client.test/cb', is_active = 0`)
	require.NoError(t, err)
	status, body = f.post(f.h.HandleAuthorizeCode, url.Values{"request": {handle}, "code": {code}}, nil)
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, body, "Unknown client_id")

	// Nor does a handle the server never gave out.
	status, _ = f.post(f.h.HandleAuthorizeCode, url.Values{"request": {"made-up"}, "code": {code}}, nil)
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, 1, f.codes(), "only the untampered sign-in got a code")
}

// The emailed link, opened in the browser the sign-in started in, asks to
// allow access and finishes the sign-in. Opening it uses nothing.
func TestEmailSignIn_LinkFinishesInSameBrowser(t *testing.T) {
	f := newAuthorizeFixture(t)
	_, handle, cookie := f.start()
	token, _ := f.emailed(handle, "u1@example.com")

	page := f.openLink(handle, token, cookie)
	assert.Contains(t, page, "Allow and continue")
	assert.Equal(t, 1, f.count("sign_in_codes"), "opening the link uses nothing")

	status, body := f.post(f.h.HandleAuthorizeLinkSubmit, url.Values{"request": {handle}, "token": {token}}, cookie)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, http.StatusOK, f.redeemed(body, "verifier"))
	assert.Equal(t, 0, f.count("mcp_oauth_requests"))
	assert.Equal(t, 0, f.count("sign_in_codes"), "using the link retires the code")
}

// Opened on another device the link can't finish there (the client's
// redirect is on the first device): the page says to enter the code where
// the sign-in started, and the code still works there.
func TestEmailSignIn_LinkElsewhereSaysEnterTheCode(t *testing.T) {
	f := newAuthorizeFixture(t)
	_, handle, _ := f.start()
	token, code := f.emailed(handle, "u1@example.com")

	page := f.openLink(handle, token, nil)
	assert.Contains(t, page, "enter the 6-digit code from this email")
	assert.NotContains(t, page, "Allow and continue")
	other := &http.Cookie{Name: browserCookie, Value: generateSecureToken(32)}
	_, body := f.post(f.h.HandleAuthorizeLinkSubmit, url.Values{"request": {handle}, "token": {token}}, other)
	assert.Contains(t, body, "Finish where you started")
	assert.Equal(t, 0, f.codes())
	assert.Equal(t, 1, f.count("sign_in_codes"), "the link and the code stay unused")

	status, body := f.post(f.h.HandleAuthorizeCode, url.Values{"request": {handle}, "code": {code}}, nil)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, http.StatusOK, f.redeemed(body, "verifier"))
}

// Five wrong codes retire the emailed sign-in, as in the app: the right
// code then gets no authorization code.
func TestEmailSignIn_LocksAfterWrongCodes(t *testing.T) {
	f := newAuthorizeFixture(t)
	_, handle, _ := f.start()
	_, code := f.emailed(handle, "u1@example.com")
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	for i := 0; i < 5; i++ {
		f.post(f.h.HandleAuthorizeCode, url.Values{"request": {handle}, "code": {wrong}}, nil)
	}
	assert.Equal(t, 0, f.count("sign_in_codes"), "the fifth wrong code retires it")
	_, body := f.post(f.h.HandleAuthorizeCode, url.Values{"request": {handle}, "code": {code}}, nil)
	assert.Contains(t, body, "wrong or has expired")
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
