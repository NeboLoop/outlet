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

	// Delete any existing password reset tokens for this user
	err = l.svcCtx.DB.DeleteAuthTokensByUser(l.ctx, db.DeleteAuthTokensByUserParams{
		UserID:    user.ID,
		TokenType: "password_reset",
	})
	if err != nil {
		l.Errorf("Error deleting existing reset tokens: %v", err)
		// Continue anyway
	}

	// Generate secure random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		l.Errorf("Error generating reset token: %v", err)
		return &types.ForgotPasswordResponse{Message: successMessage}, nil
	}
	resetToken := hex.EncodeToString(tokenBytes)
	expiresAt := time.Now().Add(1 * time.Hour) // Token valid for 1 hour

	// Store reset token
	_, err = l.svcCtx.DB.CreateAuthToken(l.ctx, db.CreateAuthTokenParams{
		ID:        generateUUID(),
		UserID:    user.ID,
		Token:     resetToken,
		TokenType: "password_reset",
		ExpiresAt: expiresAt.Format(time.RFC3339),
	})
	if err != nil {
		l.Errorf("Error storing reset token: %v", err)
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
	resetURL := fmt.Sprintf("%s/auth/reset-password?token=%s", strings.TrimSuffix(baseURL, "/"), url.QueryEscape(token))
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
