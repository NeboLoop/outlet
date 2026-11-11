package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOutboundPaceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetOutboundPaceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOutboundPaceLogic {
	return &GetOutboundPaceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetOutboundPaceLogic) GetOutboundPace() (resp *types.OutboundPace, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	p, err := l.svcCtx.Outbound.Pace(l.ctx, org)
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.OutboundPace](p)
}
