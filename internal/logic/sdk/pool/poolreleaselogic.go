package pool

import (
	"context"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/middleware"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PoolReleaseLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Release every mailbox of the tenant
func NewPoolReleaseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PoolReleaseLogic {
	return &PoolReleaseLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PoolReleaseLogic) PoolRelease(req *types.PoolReleaseRequest) (resp *types.PoolReleaseResponse, err error) {
	orgID, ok := l.ctx.Value(middleware.OrgIDKey).(string)
	if !ok || orgID == "" {
		return nil, errorx.NewUnauthorizedError("API key required")
	}
	n, err := l.svcCtx.MailPool.Release(l.ctx, orgID, req.Tenant)
	if err != nil {
		return nil, err
	}
	return &types.PoolReleaseResponse{Released: n}, nil
}
