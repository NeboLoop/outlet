package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CheckOutboundDoNotContactLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCheckOutboundDoNotContactLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckOutboundDoNotContactLogic {
	return &CheckOutboundDoNotContactLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CheckOutboundDoNotContactLogic) CheckOutboundDoNotContact(req *types.OutboundDNCCheckRequest) (resp *types.OutboundDNCCheck, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	ex, excluded := l.svcCtx.Outbound.DoNotContact(l.ctx, org, req.Email)
	return &types.OutboundDNCCheck{Excluded: excluded, Source: ex.Source, Reason: ex.Reason}, nil
}
