package auth

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginRequest) (resp *types.LoginResponse, err error) {
	// Validate input
	if req.Email == "" || req.Password == "" {
		return nil, fmt.Errorf("email and password are required")
	}

	// Get user from database by email
	user, err := l.svcCtx.DB.GetUserByEmail(l.ctx, req.Email)
	if err == sql.ErrNoRows {
		l.Errorf("Login attempt for non-existent user: %s", req.Email)
		return nil, fmt.Errorf("invalid email or password")
	}
	if err != nil {
		l.Errorf("Database error getting user: %v", err)
		return nil, fmt.Errorf("database error")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		l.Errorf("Invalid password attempt for user: %s", req.Email)
		// Increment failed login attempts
		_ = l.svcCtx.DB.IncrementFailedLogins(l.ctx, user.ID)
		return nil, fmt.Errorf("invalid email or password")
	}

	// Check if account is locked
	if user.LockedUntil.Valid && user.LockedUntil.String != "" {
		lockedUntil, parseErr := time.Parse(time.RFC3339, user.LockedUntil.String)
		if parseErr == nil && time.Now().Before(lockedUntil) {
			return nil, fmt.Errorf("account is locked until %s", user.LockedUntil.String)
		}
	}

	// A disabled or pending account gets no tokens.
	if _, err := access.Active(l.ctx, l.svcCtx.DB, user.ID); err != nil {
		return nil, err
	}

	// Generate access token
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
		return nil, fmt.Errorf("failed to generate authentication token")
	}

	// Generate refresh token
	refreshExpire := time.Duration(l.svcCtx.Config.Auth.RefreshExpire) * time.Second
	refreshToken, err := utils.GenerateToken(
		user.ID,
		user.Email,
		user.Role,
		l.svcCtx.Config.Auth.RefreshSecret,
		refreshExpire,
	)
	if err != nil {
		l.Errorf("Error generating refresh token: %v", err)
		return nil, fmt.Errorf("failed to generate refresh token")
	}

	// Update last login and reset failed attempts
	if err := l.svcCtx.DB.UpdateLastLogin(l.ctx, user.ID); err != nil {
		l.Errorf("Failed to update last login: %v", err)
		// Don't fail the login, just log the error
	}

	l.Infof("User logged in successfully: %s (role: %s)", user.Email, user.Role)

	// Calculate expiration time
	expiresAt := time.Now().Add(accessExpire)

	return &types.LoginResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt.Format(time.RFC3339),
		User: types.UserInfo{
			Id:        user.ID,
			Email:     user.Email,
			Name:      user.Name,
			Role:      user.Role,
			Active:    user.Status == "active",
			CreatedAt: utils.FormatNullString(user.CreatedAt),
		},
	}, nil
}
