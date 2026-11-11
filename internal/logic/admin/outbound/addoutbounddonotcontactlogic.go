package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddOutboundDoNotContactLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddOutboundDoNotContactLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddOutboundDoNotContactLogic {
	return &AddOutboundDoNotContactLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddOutboundDoNotContactLogic) AddOutboundDoNotContact(req *types.OutboundDNCAddRequest) (resp *types.OutboundDNCAddResponse, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	res, err := l.svcCtx.Outbound.AddDoNotContact(l.ctx, org, req.List, req.Entries)
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.OutboundDNCAddResponse](res)
}
