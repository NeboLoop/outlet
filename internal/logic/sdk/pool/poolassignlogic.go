package pool

import (
	"context"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/middleware"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PoolAssignLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Assign the tenant free mailboxes
func NewPoolAssignLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PoolAssignLogic {
	return &PoolAssignLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PoolAssignLogic) PoolAssign(req *types.PoolAssignRequest) (resp *types.PoolAssignResponse, err error) {
	orgID, ok := l.ctx.Value(middleware.OrgIDKey).(string)
	if !ok || orgID == "" {
		return nil, errorx.NewUnauthorizedError("API key required")
	}
	c, assigned, err := l.svcCtx.MailPool.Assign(l.ctx, orgID, req.Tenant, req.Count)
	if err != nil {
		return nil, err
	}
	return &types.PoolAssignResponse{Assigned: assigned, Tenant: req.Tenant, TenantHeld: c.TenantHeld, Mailboxes: mailboxCapacities(c)}, nil
}
