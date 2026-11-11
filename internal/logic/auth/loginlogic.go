package auth

import (
	"context"
	"fmt"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

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

	resp, err = issueSession(l.svcCtx.Config, user)
	if err != nil {
		l.Errorf("Error issuing session: %v", err)
		return nil, fmt.Errorf("failed to generate authentication token")
	}
	l.Infof("User logged in successfully: %s (role: %s)", user.Email, user.Role)
	return resp, nil
}
