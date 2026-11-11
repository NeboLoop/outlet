package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/outlet-sh/outlet/internal/access"

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
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}

	// The same answer whether or not the address has an account, so the
	// form can't be used to find out who has one.
	sent := &types.SendSignInEmailResponse{Message: "If that email has an account, we sent it a sign-in link and code"}

	u, err := l.svcCtx.DB.GetUserByEmail(l.ctx, email)
	if errors.Is(err, sql.ErrNoRows) {
		l.Infof("Sign-in email requested for an address with no account")
		return sent, nil
	}
	if err != nil {
		l.Errorf("Database error checking email: %v", err)
		return sent, nil
	}
	// Only an active account can sign in, so only it gets the email.
	user, err := access.Active(l.ctx, l.svcCtx.DB, u.ID)
	if err != nil {
		l.Infof("Sign-in email requested for an account that can't sign in: %s", u.ID)
		return sent, nil
	}

	token, code, err := IssueSignIn(l.ctx, l.svcCtx.DB, l.svcCtx.Config.Auth.AccessSecret, user.ID)
	if err != nil {
		l.Errorf("Error issuing sign-in code: %v", err)
		return sent, nil
	}

	// Sent asynchronously, the way the reset email is, so the answer takes as
	// long whether or not the address has an account.
	msg := signInMessage(l.svcCtx.Config.App.BaseURL, user.Email, user.Name, token, code)
	go func() {
		if err := l.svcCtx.EmailService.Send(context.Background(), msg); err != nil {
			l.Errorf("Error sending sign-in email: %v", err)
		} else {
			l.Infof("Sign-in email sent to: %s", user.Email)
		}
	}()

	return sent, nil
}
