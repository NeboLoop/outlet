package outbound

import (
	"context"

	adminoutbound "github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SdkGetOutboundCampaignLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSdkGetOutboundCampaignLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SdkGetOutboundCampaignLogic {
	return &SdkGetOutboundCampaignLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SdkGetOutboundCampaignLogic) SdkGetOutboundCampaign(req *types.OutboundIDRequest) (resp *types.OutboundCampaignResults, err error) {
	org, err := brand(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	return adminoutbound.GetOutboundCampaign(l.ctx, l.svcCtx, org, req)
}
