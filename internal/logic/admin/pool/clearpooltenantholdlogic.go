package pool

import (
	"context"
	"github.com/outlet-sh/outlet/internal/access"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ClearPoolTenantHoldLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewClearPoolTenantHoldLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ClearPoolTenantHoldLogic {
	return &ClearPoolTenantHoldLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ClearPoolTenantHoldLogic) ClearPoolTenantHold(req *types.ClearPoolTenantHoldRequest) (resp *types.ClearPoolTenantHoldResponse, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return nil, err
	}
	n, err := l.svcCtx.MailPool.ClearTenantHold(l.ctx, req.OrgId, req.Tenant)
	if err != nil {
		return nil, err
	}
	return &types.ClearPoolTenantHoldResponse{Cleared: n}, nil
}
