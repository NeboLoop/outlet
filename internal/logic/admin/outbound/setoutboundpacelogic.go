package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetOutboundPaceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSetOutboundPaceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetOutboundPaceLogic {
	return &SetOutboundPaceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SetOutboundPaceLogic) SetOutboundPace(req *types.OutboundPaceRequest) (resp *types.OutboundPace, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Admin)
	if err != nil {
		return nil, err
	}
	p, err := l.svcCtx.Outbound.SetPace(l.ctx, org, req.Pace, req.DailyNewProspects)
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.OutboundPace](p)
}
