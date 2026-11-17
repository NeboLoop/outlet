package auth

import (
	"context"
	"fmt"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SignInWithEmailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Sign in with the emailed link's token, or with the email and its code
func NewSignInWithEmailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SignInWithEmailLogic {
	return &SignInWithEmailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SignInWithEmailLogic) SignInWithEmail(req *types.SignInWithEmailRequest) (resp *types.LoginResponse, err error) {
	user, err := SignInByEmail(l.ctx, l.svcCtx, req.Token, req.Email, req.Code)
	if err != nil {
		return nil, err
	}
	resp, err = issueSession(l.svcCtx.Config, user)
	if err != nil {
		l.Errorf("Error issuing session: %v", err)
		return nil, fmt.Errorf("failed to generate authentication token")
	}
	l.Infof("User signed in by email: %s (role: %s)", user.Email, user.Role)
	return resp, nil
}
