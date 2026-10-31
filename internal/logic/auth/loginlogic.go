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
	// Checks the password, the lock after wrong passwords, and that the
	// account is active; records the sign-in.
	user, err := access.SignIn(l.ctx, l.svcCtx.DB, req.Email, req.Password)
	if err != nil {
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
