package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type RefreshTokenLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Refresh access token using refresh token
func NewRefreshTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefreshTokenLogic {
	return &RefreshTokenLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RefreshTokenLogic) RefreshToken(req *types.RefreshTokenRequest) (resp *types.LoginResponse, err error) {
	if req.RefreshToken == "" {
		return nil, fmt.Errorf("refresh token is required")
	}

	// Validate the refresh token
	claims, err := utils.ValidateToken(req.RefreshToken, l.svcCtx.Config.Auth.RefreshSecret)
	if err != nil {
		l.Errorf("Invalid refresh token: %v", err)
		return nil, fmt.Errorf("invalid or expired refresh token")
	}

	// The person must still exist and be active: a disabled account gets no
	// new tokens.
	user, err := access.Active(l.ctx, l.svcCtx.DB, claims.UserID)
	if err != nil {
		return nil, err
	}

	// Check if account is locked
	if user.LockedUntil.Valid {
		lockedUntil, err := time.Parse(time.RFC3339, user.LockedUntil.String)
		if err == nil && time.Now().Before(lockedUntil) {
			return nil, fmt.Errorf("account is locked until %s", user.LockedUntil.String)
		}
	}

	// A new access token and a rotated refresh token.
	resp, err = issueSession(l.svcCtx.Config, user)
	if err != nil {
		l.Errorf("Error issuing session: %v", err)
		return nil, fmt.Errorf("failed to generate access token")
	}
	l.Infof("Token refreshed for user: %s", user.Email)
	return resp, nil
}
