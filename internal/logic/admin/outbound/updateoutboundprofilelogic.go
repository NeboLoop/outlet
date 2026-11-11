package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/services/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateOutboundProfileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateOutboundProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateOutboundProfileLogic {
	return &UpdateOutboundProfileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateOutboundProfileLogic) UpdateOutboundProfile(req *types.OutboundProfile) (resp *types.OutboundProfile, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	in, err := convertInto[outbound.Profile](req)
	if err != nil {
		return nil, err
	}
	p, err := l.svcCtx.Outbound.UpdateProfile(l.ctx, org, in)
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.OutboundProfile](p)
}
