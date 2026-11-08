package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/services/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateOutboundCampaignLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateOutboundCampaignLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateOutboundCampaignLogic {
	return &CreateOutboundCampaignLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateOutboundCampaignLogic) CreateOutboundCampaign(req *types.OutboundCampaignRequest) (resp *types.OutboundCampaignResponse, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	in, err := convertInto[outbound.CampaignInput](req)
	if err != nil {
		return nil, err
	}
	res, err := l.svcCtx.Outbound.DraftOutreach(l.ctx, org, in)
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.OutboundCampaignResponse](res)
}
