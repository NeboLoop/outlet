package auth

import (
	"context"
	"testing"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSignInSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()
	s := newAuthSvc(t)
	s.EmailService = email.NewService(s.DB, nil)
	_, err := s.DB.CreateUser(context.Background(), db.CreateUserParams{
		ID: "u1", Email: "ana@example.com", PasswordHash: "x", Name: "Ana", Role: "agent", Status: "active", EmailVerified: int64(1),
	})
	require.NoError(t, err)
	return s
}

func signInWithEmail(s *svc.ServiceContext, req types.SignInWithEmailRequest) (*types.LoginResponse, error) {
	return NewSignInWithEmailLogic(context.Background(), s).SignInWithEmail(&req)
}

func signInRows(t *testing.T, s *svc.ServiceContext) int {
	t.Helper()
	var n int
	require.NoError(t, s.DB.GetDB().QueryRow(`SELECT COUNT(*) FROM sign_in_codes`).Scan(&n))
	return n
}

func TestSignInMessage(t *testing.T) {
	m := signInMessage("https://outlet.test/", "ana@example.com", "Ana <b>", "tok123", "042917")
	assert.Contains(t, m.HTML, `href="https://outlet.test/auth/sign-in?token=tok123"`, "the link opens the app's sign-in page")
	assert.Contains(t, m.HTML, "042917")
	assert.Contains(t, m.HTML, "Hi Ana &lt;b&gt;,")
	assert.Equal(t, "ana@example.com", m.To)
	assert.Empty(t, m.OrgID, "platform mail")
	assert.Empty(t, m.FromEmail, "platform mail")
}

// The answer is the same whether or not the address has an account, and
// only an account gets a sign-in.
func TestSendSignInEmail_SameAnswerForUnknownEmail(t *testing.T) {
	s := newSignInSvc(t)
	l := NewSendSignInEmailLogic(context.Background(), s)

	unknown, err := l.SendSignInEmail(&types.SendSignInEmailRequest{Email: "nobody@example.com"})
	require.NoError(t, err)
	assert.Equal(t, 0, signInRows(t, s), "no account, no sign-in")

	known, err := l.SendSignInEmail(&types.SendSignInEmailRequest{Email: " Ana@Example.com "})
	require.NoError(t, err)
	assert.Equal(t, unknown, known)
	assert.Equal(t, 1, signInRows(t, s))

	var tokenHash, codeHash string
	require.NoError(t, s.DB.GetDB().QueryRow(`SELECT token_hash, code_hash FROM sign_in_codes WHERE user_id = 'u1'`).Scan(&tokenHash, &codeHash))
	assert.Len(t, tokenHash, 64, "only a hash of the link token is kept")
	assert.Len(t, codeHash, 64, "only a hash of the code is kept")
}

func TestSendSignInEmail_InactiveAccountGetsNothing(t *testing.T) {
	s := newSignInSvc(t)
	_, err := s.DB.GetDB().Exec(`UPDATE users SET status = 'inactive' WHERE id = 'u1'`)
	require.NoError(t, err)
	_, err = NewSendSignInEmailLogic(context.Background(), s).SendSignInEmail(&types.SendSignInEmailRequest{Email: "ana@example.com"})
	require.NoError(t, err)
	assert.Equal(t, 0, signInRows(t, s))
}

func TestSignInWithEmail_Link(t *testing.T) {
	s := newSignInSvc(t)
	token, code, err := IssueSignIn(context.Background(), s.DB, s.Config.Auth.AccessSecret, "u1")
	require.NoError(t, err)

	resp, err := signInWithEmail(s, types.SignInWithEmailRequest{Token: token})
	require.NoError(t, err)
	assert.Equal(t, "u1", resp.User.Id)
	assert.True(t, resp.User.Active)
	claims, err := utils.ValidateToken(resp.Token, s.Config.Auth.AccessSecret)
	require.NoError(t, err, "the session is a real access token")
	assert.Equal(t, "u1", claims.UserID)
	_, err = utils.ValidateToken(resp.RefreshToken, s.Config.Auth.RefreshSecret)
	require.NoError(t, err, "and a real refresh token")

	_, err = signInWithEmail(s, types.SignInWithEmailRequest{Token: token})
	assert.ErrorIs(t, err, errSignInEmailBad, "the link works once")
	_, err = signInWithEmail(s, types.SignInWithEmailRequest{Email: "ana@example.com", Code: code})
	assert.ErrorIs(t, err, errSignInEmailBad, "using the link retires the code")
}

func TestSignInWithEmail_Code(t *testing.T) {
	s := newSignInSvc(t)
	token, code, err := IssueSignIn(context.Background(), s.DB, s.Config.Auth.AccessSecret, "u1")
	require.NoError(t, err)
	assert.Regexp(t, `^\d{6}$`, code)

	resp, err := signInWithEmail(s, types.SignInWithEmailRequest{Email: " ANA@example.com ", Code: " " + code + " "})
	require.NoError(t, err)
	assert.Equal(t, "u1", resp.User.Id)

	_, err = signInWithEmail(s, types.SignInWithEmailRequest{Email: "ana@example.com", Code: code})
	assert.ErrorIs(t, err, errSignInEmailBad, "the code works once")
	_, err = signInWithEmail(s, types.SignInWithEmailRequest{Token: token})
	assert.ErrorIs(t, err, errSignInEmailBad, "using the code retires the link")
}

// A sign-in also clears a lock from wrong passwords: it proves the inbox.
func TestSignInWithEmail_ClearsPasswordLock(t *testing.T) {
	s := newSignInSvc(t)
	_, err := s.DB.GetDB().Exec(`UPDATE users SET failed_login_attempts = 5, locked_until = datetime('now', '+30 minutes') WHERE id = 'u1'`)
	require.NoError(t, err)
	token, _, err := IssueSignIn(context.Background(), s.DB, s.Config.Auth.AccessSecret, "u1")
	require.NoError(t, err)
	_, err = signInWithEmail(s, types.SignInWithEmailRequest{Token: token})
	require.NoError(t, err)
	var attempts int
	require.NoError(t, s.DB.GetDB().QueryRow(`SELECT failed_login_attempts FROM users WHERE id = 'u1'`).Scan(&attempts))
	assert.Equal(t, 0, attempts)
}

func TestSignInWithEmail_Expired(t *testing.T) {
	s := newSignInSvc(t)
	token, code, err := IssueSignIn(context.Background(), s.DB, s.Config.Auth.AccessSecret, "u1")
	require.NoError(t, err)
	_, err = s.DB.GetDB().Exec(`UPDATE sign_in_codes SET expires_at = datetime('now', '-1 minute')`)
	require.NoError(t, err)

	_, err = signInWithEmail(s, types.SignInWithEmailRequest{Token: token})
	assert.ErrorIs(t, err, errSignInEmailBad, "an expired link is refused")
	_, err = signInWithEmail(s, types.SignInWithEmailRequest{Email: "ana@example.com", Code: code})
	assert.ErrorIs(t, err, errSignInEmailBad, "an expired code is refused")
}

// A new email retires the one before it.
func TestSignInWithEmail_NewIssueRetiresOld(t *testing.T) {
	s := newSignInSvc(t)
	ctx := context.Background()
	oldToken, oldCode, err := IssueSignIn(ctx, s.DB, s.Config.Auth.AccessSecret, "u1")
	require.NoError(t, err)
	token, code, err := IssueSignIn(ctx, s.DB, s.Config.Auth.AccessSecret, "u1")
	require.NoError(t, err)
	assert.Equal(t, 1, signInRows(t, s))

	_, err = signInWithEmail(s, types.SignInWithEmailRequest{Token: oldToken})
	assert.ErrorIs(t, err, errSignInEmailBad)
	if oldCode != code {
		_, err = signInWithEmail(s, types.SignInWithEmailRequest{Email: "ana@example.com", Code: oldCode})
		assert.ErrorIs(t, err, errSignInEmailBad)
	}
	_, err = signInWithEmail(s, types.SignInWithEmailRequest{Token: token})
	require.NoError(t, err)
}

// Five wrong codes retire the email: the right code no longer works.
func TestSignInWithEmail_WrongCodeLimit(t *testing.T) {
	s := newSignInSvc(t)
	token, code, err := IssueSignIn(context.Background(), s.DB, s.Config.Auth.AccessSecret, "u1")
	require.NoError(t, err)
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}

	for i := 0; i < signInCodeTries-1; i++ {
		_, err = signInWithEmail(s, types.SignInWithEmailRequest{Email: "ana@example.com", Code: wrong})
		assert.ErrorIs(t, err, errSignInEmailBad)
	}
	assert.Equal(t, 1, signInRows(t, s), "four wrong codes leave it")
	_, err = signInWithEmail(s, types.SignInWithEmailRequest{Email: "ana@example.com", Code: wrong})
	assert.ErrorIs(t, err, errSignInEmailBad)
	assert.Equal(t, 0, signInRows(t, s), "the fifth retires it")

	_, err = signInWithEmail(s, types.SignInWithEmailRequest{Email: "ana@example.com", Code: code})
	assert.ErrorIs(t, err, errSignInEmailBad, "the right code is refused once it is retired")
	_, err = signInWithEmail(s, types.SignInWithEmailRequest{Token: token})
	assert.ErrorIs(t, err, errSignInEmailBad, "and so is the link")
}

// Four wrong codes, then the right one, still signs in.
func TestSignInWithEmail_RightCodeAfterTypos(t *testing.T) {
	s := newSignInSvc(t)
	_, code, err := IssueSignIn(context.Background(), s.DB, s.Config.Auth.AccessSecret, "u1")
	require.NoError(t, err)
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	for i := 0; i < signInCodeTries-1; i++ {
		_, _ = signInWithEmail(s, types.SignInWithEmailRequest{Email: "ana@example.com", Code: wrong})
	}
	_, err = signInWithEmail(s, types.SignInWithEmailRequest{Email: "ana@example.com", Code: code})
	require.NoError(t, err)
}

func TestSignInWithEmail_RefusalsLookAlike(t *testing.T) {
	s := newSignInSvc(t)
	_, err := signInWithEmail(s, types.SignInWithEmailRequest{Email: "nobody@example.com", Code: "123456"})
	assert.ErrorIs(t, err, errSignInEmailBad, "an address with no account")
	_, err = signInWithEmail(s, types.SignInWithEmailRequest{Email: "ana@example.com", Code: "123456"})
	assert.ErrorIs(t, err, errSignInEmailBad, "an account with no sign-in sent")
	_, err = signInWithEmail(s, types.SignInWithEmailRequest{Token: "not-a-token"})
	assert.ErrorIs(t, err, errSignInEmailBad, "a made-up link")
	_, err = signInWithEmail(s, types.SignInWithEmailRequest{Email: "ana@example.com"})
	assert.ErrorIs(t, err, errSignInEmailInput, "no link and no code")
}

func TestSignInWithEmail_InactiveAccountRefused(t *testing.T) {
	s := newSignInSvc(t)
	token, _, err := IssueSignIn(context.Background(), s.DB, s.Config.Auth.AccessSecret, "u1")
	require.NoError(t, err)
	_, err = s.DB.GetDB().Exec(`UPDATE users SET status = 'inactive' WHERE id = 'u1'`)
	require.NoError(t, err)
	_, err = signInWithEmail(s, types.SignInWithEmailRequest{Token: token})
	assert.Error(t, err)
}
