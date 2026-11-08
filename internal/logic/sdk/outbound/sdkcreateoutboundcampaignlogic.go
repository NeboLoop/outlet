package outbound

import (
	"context"

	adminoutbound "github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SdkCreateOutboundCampaignLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSdkCreateOutboundCampaignLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SdkCreateOutboundCampaignLogic {
	return &SdkCreateOutboundCampaignLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SdkCreateOutboundCampaignLogic) SdkCreateOutboundCampaign(req *types.OutboundCampaignRequest) (resp *types.OutboundCampaignResponse, err error) {
	org, err := brand(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	return adminoutbound.CreateOutboundCampaign(l.ctx, l.svcCtx, org, req)
}
