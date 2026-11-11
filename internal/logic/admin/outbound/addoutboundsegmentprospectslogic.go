package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddOutboundSegmentProspectsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddOutboundSegmentProspectsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddOutboundSegmentProspectsLogic {
	return &AddOutboundSegmentProspectsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddOutboundSegmentProspectsLogic) AddOutboundSegmentProspects(req *types.OutboundSegmentAddRequest) (resp *types.OutboundAddProspectsResponse, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	res, err := l.svcCtx.Outbound.AddToSegment(l.ctx, org, req.Id, req.ProspectIds)
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.OutboundAddProspectsResponse](map[string]any{"prospects": res})
}
