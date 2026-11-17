package auth

import (
	"context"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendSignInEmailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Email a sign-in link and code (the same answer whether or not the address has an account)
func NewSendSignInEmailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendSignInEmailLogic {
	return &SendSignInEmailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SendSignInEmailLogic) SendSignInEmail(req *types.SendSignInEmailRequest) (resp *types.SendSignInEmailResponse, err error) {
	baseURL := l.svcCtx.Config.App.BaseURL
	if err := SendSignIn(l.ctx, l.svcCtx, req.Email, func(token string) string { return SignInURL(baseURL, token) }); err != nil {
		return nil, err
	}
	return &types.SendSignInEmailResponse{Message: SignInEmailSent}, nil
}
