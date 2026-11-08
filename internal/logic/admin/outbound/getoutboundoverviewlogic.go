package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
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
	o, err := l.svcCtx.Outbound.Overview(l.ctx, org)
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.OutboundOverview](o)
}
