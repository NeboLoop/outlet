package auth

import (
	"context"
	"strings"
	"testing"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestPasswordResetMessage(t *testing.T) {
	m := passwordResetMessage("https://outlet.test/", "ana@example.com", "Ana <b>", "tok123")
	assert.Contains(t, m.HTML, `href="https://outlet.test/auth/reset-password?token=tok123"`, "the link opens the app's reset page")
	assert.Contains(t, m.HTML, "Hi Ana &lt;b&gt;,")
	assert.Equal(t, "ana@example.com", m.To)
	// Platform mail: no brand and no From of its own, so Send uses the
	// platform's default sender.
	assert.Empty(t, m.OrgID)
	assert.Empty(t, m.FromEmail)
}

func TestForgotThenResetPassword(t *testing.T) {
	s := newAuthSvc(t)
	s.EmailService = email.NewService(s.DB, nil)
	ctx := context.Background()
	_, err := s.DB.CreateUser(ctx, db.CreateUserParams{
		ID: "u1", Email: "ana@example.com", PasswordHash: "x", Name: "Ana", Role: "agent", Status: "active", EmailVerified: int64(1),
	})
	require.NoError(t, err)

	_, err = NewForgotPasswordLogic(ctx, s).ForgotPassword(&types.ForgotPasswordRequest{Email: " Ana@Example.com "})
	require.NoError(t, err)
	var token string
	require.NoError(t, s.DB.GetDB().QueryRow(`SELECT token FROM auth_tokens WHERE user_id = 'u1' AND token_type = 'password_reset'`).Scan(&token))

	_, err = NewResetPasswordLogic(ctx, s).ResetPassword(&types.ResetPasswordRequest{Token: token, NewPassword: "new-password", ConfirmPassword: "new-password"})
	require.NoError(t, err)
	user, err := s.DB.GetUserByEmail(ctx, "ana@example.com")
	require.NoError(t, err)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("new-password")))

	_, err = NewResetPasswordLogic(ctx, s).ResetPassword(&types.ResetPasswordRequest{Token: token, NewPassword: "other-password", ConfirmPassword: "other-password"})
	assert.True(t, err != nil && strings.Contains(err.Error(), "invalid or expired"), "a reset link works once: %v", err)
}

// The link `outlet reset-link` prints resets the password once, and issuing
// a new one retires the old.
func TestIssuePasswordReset(t *testing.T) {
	s := newAuthSvc(t)
	ctx := context.Background()
	_, err := s.DB.CreateUser(ctx, db.CreateUserParams{
		ID: "u1", Email: "ana@example.com", PasswordHash: "x", Name: "Ana", Role: "agent", Status: "active", EmailVerified: int64(1),
	})
	require.NoError(t, err)

	old, err := IssuePasswordReset(ctx, s.DB, "u1")
	require.NoError(t, err)
	token, err := IssuePasswordReset(ctx, s.DB, "u1")
	require.NoError(t, err)
	assert.Equal(t, "https://outlet.example.com/auth/reset-password?token="+token, ResetURL("https://outlet.example.com/", token))

	_, err = NewResetPasswordLogic(ctx, s).ResetPassword(&types.ResetPasswordRequest{Token: old, NewPassword: "new-password", ConfirmPassword: "new-password"})
	assert.Error(t, err, "the older link is retired")
	_, err = NewResetPasswordLogic(ctx, s).ResetPassword(&types.ResetPasswordRequest{Token: token, NewPassword: "new-password", ConfirmPassword: "new-password"})
	require.NoError(t, err)
}
