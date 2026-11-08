package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOutboundOverviewLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetOutboundOverviewLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOutboundOverviewLogic {
	return &GetOutboundOverviewLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetOutboundOverviewLogic) GetOutboundOverview() (resp *types.OutboundOverview, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	return GetOutboundOverview(l.ctx, l.svcCtx, org)
}

// GetOutboundOverview runs for a brand already resolved: the admin route's signed-in
// person, or the API key of /sdk/v1/outbound.
func GetOutboundOverview(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization) (*types.OutboundOverview, error) {
	o, err := svcCtx.Outbound.Overview(ctx, org)
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.OutboundOverview](o)
}
