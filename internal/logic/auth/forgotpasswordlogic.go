package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"html"
	"net/url"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	emailsvc "github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

func generateUUID() string {
	return uuid.New().String()
}

type ForgotPasswordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Request password reset
func NewForgotPasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ForgotPasswordLogic {
	return &ForgotPasswordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ForgotPasswordLogic) ForgotPassword(req *types.ForgotPasswordRequest) (resp *types.ForgotPasswordResponse, err error) {
	// Normalize email
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}

	// For security, always return the same message regardless of whether email exists
	successMessage := "If the email exists, a password reset link has been sent"

	// Check if user exists
	user, err := l.svcCtx.DB.GetUserByEmail(l.ctx, email)
	if err == sql.ErrNoRows {
		// Don't reveal that email doesn't exist
		l.Infof("Password reset requested for non-existent email: %s", email)
		return &types.ForgotPasswordResponse{Message: successMessage}, nil
	}
	if err != nil {
		l.Errorf("Database error checking email: %v", err)
		return &types.ForgotPasswordResponse{Message: successMessage}, nil
	}

	resetToken, err := IssuePasswordReset(l.ctx, l.svcCtx.DB, user.ID)
	if err != nil {
		l.Errorf("Error issuing reset token: %v", err)
		return &types.ForgotPasswordResponse{Message: successMessage}, nil
	}

	// Send the reset email asynchronously, so the answer takes as long
	// whether or not the address has an account. It is platform mail: Send
	// gives it the platform's default sender.
	msg := passwordResetMessage(l.svcCtx.Config.App.BaseURL, email, user.Name, resetToken)
	go func() {
		if err := l.svcCtx.EmailService.Send(context.Background(), msg); err != nil {
			l.Errorf("Error sending password reset email: %v", err)
		} else {
			l.Infof("Password reset email sent to: %s", email)
		}
	}()

	return &types.ForgotPasswordResponse{Message: successMessage}, nil
}

// passwordResetMessage is the email with a person's reset link. The link
// opens the app's reset page, /auth/reset-password.
func passwordResetMessage(baseURL, to, name, token string) emailsvc.Message {
	resetURL := ResetURL(baseURL, token)
	body := fmt.Sprintf(`
<html>
<body>
<h2>Password Reset Request</h2>
<p>Hi %s,</p>
<p>We received a request to reset your password. Click the link below to set a new password:</p>
<p><a href="%s">Reset Password</a></p>
<p>This link will expire in 1 hour.</p>
<p>If you didn't request a password reset, you can safely ignore this email. Your password will remain unchanged.</p>
<br>
<p>Best regards,<br>The Outlet Team</p>
</body>
</html>`, html.EscapeString(name), resetURL)
	return emailsvc.Message{To: to, Subject: "Reset your password", HTML: body}
}

// IssuePasswordReset replaces a user's reset tokens with a new one, good for
// an hour and once. It is the one way a reset is issued: "Forgot your
// password?" emails it, and `outlet reset-link` prints it.
func IssuePasswordReset(ctx context.Context, q db.Querier, userID string) (string, error) {
	if err := q.DeleteAuthTokensByUser(ctx, db.DeleteAuthTokensByUserParams{UserID: userID, TokenType: "password_reset"}); err != nil {
		return "", fmt.Errorf("delete old reset tokens: %w", err)
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	if _, err := q.CreateAuthToken(ctx, db.CreateAuthTokenParams{
		ID: generateUUID(), UserID: userID, Token: token, TokenType: "password_reset",
		ExpiresAt: time.Now().Add(time.Hour).Format(time.RFC3339),
	}); err != nil {
		return "", fmt.Errorf("store reset token: %w", err)
	}
	return token, nil
}

// ResetURL is the app's reset page for a token.
func ResetURL(baseURL, token string) string {
	return fmt.Sprintf("%s/auth/reset-password?token=%s", strings.TrimSuffix(baseURL, "/"), url.QueryEscape(token))
}
