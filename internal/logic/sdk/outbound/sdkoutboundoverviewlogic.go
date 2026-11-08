package outbound

import (
	"context"

	adminoutbound "github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SdkOutboundOverviewLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSdkOutboundOverviewLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SdkOutboundOverviewLogic {
	return &SdkOutboundOverviewLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SdkOutboundOverviewLogic) SdkOutboundOverview() (resp *types.OutboundOverview, err error) {
	org, err := brand(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	return adminoutbound.GetOutboundOverview(l.ctx, l.svcCtx, org)
}
