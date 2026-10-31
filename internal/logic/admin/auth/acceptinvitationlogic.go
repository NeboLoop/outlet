package auth

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AcceptInvitationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Accept an invitation to a brand as the signed-in person (the invitation's email must be theirs)
func NewAcceptInvitationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AcceptInvitationLogic {
	return &AcceptInvitationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// AcceptInvitation joins the signed-in person to the brand an invitation
// names. Someone who already has an account joins a brand this way: they
// need the token (it goes out only by email) and to be signed in as the
// invited address.
func (l *AcceptInvitationLogic) AcceptInvitation(req *types.AcceptInvitationRequest) (resp *types.Response, err error) {
	signedIn, ok := access.User(l.ctx)
	if !ok {
		return nil, errorx.NewUnauthorizedError("Sign in first.")
	}
	user, err := access.Active(l.ctx, l.svcCtx.DB, signedIn.ID)
	if err != nil {
		return nil, err
	}
	inv, err := access.Invitation(l.ctx, l.svcCtx.DB, req.Token)
	if err != nil {
		return nil, err
	}
	if err := l.svcCtx.DB.ExecTx(l.ctx, func(q *db.Queries) error {
		return access.Join(l.ctx, q, inv, user)
	}); err != nil {
		return nil, err
	}
	l.Infof("Invitation accepted: %s joined brand %s as %s", user.Email, inv.OrgID, inv.Role)
	return &types.Response{Success: true, Message: "You joined " + inv.OrgName + "."}, nil
}
