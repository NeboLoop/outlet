package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
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
	return CreateOutboundCampaign(l.ctx, l.svcCtx, org, req)
}

// CreateOutboundCampaign runs for a brand already resolved: the admin route's signed-in
// person, or the API key of /sdk/v1/outbound.
func CreateOutboundCampaign(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.OutboundCampaignRequest) (*types.OutboundCampaignResponse, error) {
	in, err := convertInto[outbound.CampaignInput](req)
	if err != nil {
		return nil, err
	}
	res, err := svcCtx.Outbound.DraftOutreach(ctx, org, in)
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.OutboundCampaignResponse](res)
}
