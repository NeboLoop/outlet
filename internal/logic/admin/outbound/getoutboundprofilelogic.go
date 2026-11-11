package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOutboundProfileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetOutboundProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOutboundProfileLogic {
	return &GetOutboundProfileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetOutboundProfileLogic) GetOutboundProfile() (resp *types.OutboundProfile, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	p, err := l.svcCtx.Outbound.Profile(l.ctx, org)
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.OutboundProfile](p)
}
