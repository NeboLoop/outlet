package pool

import (
	"context"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/middleware"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PoolCapacityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// The tenant's mailboxes with today's allowance left
func NewPoolCapacityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PoolCapacityLogic {
	return &PoolCapacityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PoolCapacityLogic) PoolCapacity(req *types.PoolCapacityRequest) (resp *types.PoolCapacityResponse, err error) {
	orgID, ok := l.ctx.Value(middleware.OrgIDKey).(string)
	if !ok || orgID == "" {
		return nil, errorx.NewUnauthorizedError("API key required")
	}
	c, err := l.svcCtx.MailPool.Capacity(l.ctx, orgID, req.Tenant)
	if err != nil {
		return nil, err
	}
	return &types.PoolCapacityResponse{Tenant: req.Tenant, TenantHeld: c.TenantHeld, Mailboxes: mailboxCapacities(c)}, nil
}
