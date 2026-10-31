package auth

import (
	"context"
	"errors"

	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ValidateInvitationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Validate invitation token and get user details
func NewValidateInvitationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ValidateInvitationLogic {
	return &ValidateInvitationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ValidateInvitationLogic) ValidateInvitation(req *types.ValidateInvitationRequest) (resp *types.ValidateInvitationResponse, err error) {
	inv, err := usableInvitation(l.ctx, l.svcCtx.DB, req.Token)
	if err != nil {
		var ce *errorx.CodeError
		if !errors.As(err, &ce) {
			l.Errorf("Checking invitation: %v", err)
			return nil, errorx.NewInternalError("Could not check this invitation. Try again.")
		}
		return &types.ValidateInvitationResponse{Valid: false, Message: ce.Message}, nil
	}
	return &types.ValidateInvitationResponse{
		Email:     inv.Email,
		Role:      inv.Role,
		BrandName: inv.OrgName,
		Valid:     true,
	}, nil
}
