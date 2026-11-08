package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/services/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddOutboundProspectsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddOutboundProspectsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddOutboundProspectsLogic {
	return &AddOutboundProspectsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddOutboundProspectsLogic) AddOutboundProspects(req *types.OutboundAddProspectsRequest) (resp *types.OutboundAddProspectsResponse, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	in, err := convertInto[[]outbound.ProspectInput](req.Prospects)
	if err != nil {
		return nil, err
	}
	res, err := l.svcCtx.Outbound.AddProspects(l.ctx, org, req.List, in)
	if err != nil {
		return nil, apiErr(err)
	}
	out, err := convert[[]types.OutboundProspectResult](res)
	if err != nil {
		return nil, err
	}
	return &types.OutboundAddProspectsResponse{Prospects: *out}, nil
}
