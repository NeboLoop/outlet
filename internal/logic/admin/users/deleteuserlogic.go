package users

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteUserLogic {
	return &DeleteUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteUserLogic) DeleteUser(req *types.DeleteUserRequest) (resp *types.AnalyticsResponse, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return nil, err
	}

	n, err := l.svcCtx.DB.DeleteUser(l.ctx, req.Id)
	if err != nil {
		l.Errorf("Failed to delete user: %v", err)
		return nil, err
	}
	if n == 0 {
		if _, err := l.svcCtx.DB.GetUserByID(l.ctx, req.Id); err != nil {
			return nil, errorx.NewNotFoundError("User not found.")
		}
		return nil, errorx.NewBadRequestError("This is the last super admin. Make someone else a super admin first.")
	}

	return &types.AnalyticsResponse{
		Success: true,
	}, nil
}
