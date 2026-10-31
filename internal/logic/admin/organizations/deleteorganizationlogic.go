package organizations

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteOrganizationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteOrganizationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteOrganizationLogic {
	return &DeleteOrganizationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteOrganizationLogic) DeleteOrganization(req *types.DeleteOrgRequest) (*types.Response, error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, req.Id, access.Admin); err != nil {
		return nil, err
	}

	err := l.svcCtx.DB.DeleteOrganization(l.ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &types.Response{
		Success: true,
		Message: "Organization deleted successfully",
	}, nil
}
