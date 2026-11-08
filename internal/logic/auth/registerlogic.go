package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	emailsvc "github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"
)

type RegisterLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterLogic) Register(req *types.RegisterRequest) (resp *types.RegisterResponse, err error) {
	// Validate input
	if req.Email == "" {
		return nil, fmt.Errorf("email is required")
	}
	if req.Password == "" {
		return nil, fmt.Errorf("password is required")
	}
	if len(req.Password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters")
	}

	// Normalize email
	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Check if email already exists
	exists, err := l.svcCtx.DB.CheckEmailExists(l.ctx, email)
	if err != nil {
		l.Errorf("Error checking email existence: %v", err)
		return nil, fmt.Errorf("failed to check email availability")
	}
	if exists != 0 {
		return nil, fmt.Errorf("email already registered")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		l.Errorf("Error hashing password: %v", err)
		return nil, fmt.Errorf("failed to process password")
	}

	// Build name from first and last name
	name := strings.TrimSpace(req.FirstName + " " + req.LastName)
	if name == "" || name == " " {
		// Use part before @ as default name
		if idx := strings.Index(email, "@"); idx > 0 {
			name = email[:idx]
		}
	}

	// Create user with pending status (awaiting email verification)
	user, err := l.svcCtx.DB.CreateUser(l.ctx, db.CreateUserParams{
		ID:            uuid.New().String(),
		Email:         email,
		PasswordHash:  string(hashedPassword),
		Name:          name,
		Role:          "user", // Default role for self-registration
		Status:        "pending",
		EmailVerified: int64(0), // email_verified = false
		Phone:         sql.NullString{String: req.Phone, Valid: req.Phone != ""},
	})
	if err != nil {
		l.Errorf("Error creating user: %v", err)
		return nil, fmt.Errorf("failed to create account")
	}

	// Generate email verification token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		l.Errorf("Error generating verification token: %v", err)
		// Continue - user is created, they can request verification later
	} else {
		verificationToken := hex.EncodeToString(tokenBytes)
		expiresAt := time.Now().Add(24 * time.Hour) // Token valid for 24 hours

		// Store verification token
		_, err = l.svcCtx.DB.CreateAuthToken(l.ctx, db.CreateAuthTokenParams{
			ID:        uuid.New().String(),
			UserID:    user.ID,
			Token:     verificationToken,
			TokenType: "email_verification",
			ExpiresAt: expiresAt.Format(time.RFC3339),
		})
		if err != nil {
			l.Errorf("Error storing verification token: %v", err)
			// Continue - user can request new verification email
		} else {
			// Send verification email asynchronously
			go func() {
				verifyURL := fmt.Sprintf("%s/api/v1/auth/verify-email?token=%s", l.svcCtx.Config.App.BaseURL, verificationToken)
				subject := "Verify your email address"
				body := fmt.Sprintf(`
<html>
<body>
<h2>Welcome to Outlet!</h2>
<p>Hi %s,</p>
<p>Thank you for registering. Please verify your email address by clicking the link below:</p>
<p><a href="%s">Verify Email Address</a></p>
<p>This link will expire in 24 hours.</p>
<p>If you didn't create an account, you can safely ignore this email.</p>
<br>
<p>Best regards,<br>The Outlet Team</p>
</body>
</html>`, name, verifyURL)
				err := l.svcCtx.EmailService.Send(context.Background(), emailsvc.Message{
					To:      email,
					Subject: subject,
					HTML:    body,
				})
				if err != nil {
					l.Errorf("Error sending verification email: %v", err)
				}
			}()
		}
	}

	// Generate access token (allow login but with limited access until verified)
	accessExpire := time.Duration(l.svcCtx.Config.Auth.AccessExpire) * time.Second
	accessToken, err := utils.GenerateToken(
		user.ID,
		user.Email,
		user.Role,
		l.svcCtx.Config.Auth.AccessSecret,
		accessExpire,
	)
	if err != nil {
		l.Errorf("Error generating access token: %v", err)
		return nil, fmt.Errorf("account created but failed to generate token - please login")
	}

	l.Infof("User registered successfully: %s", email)

	return &types.RegisterResponse{
		Token: accessToken,
		User: types.UserInfo{
			Id:        user.ID,
			Email:     user.Email,
			Name:      user.Name,
			Role:      user.Role,
			CreatedAt: utils.FormatNullString(user.CreatedAt),
		},
	}, nil
}
