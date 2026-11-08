package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type MarkOutboundPartnerLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewMarkOutboundPartnerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MarkOutboundPartnerLogic {
	return &MarkOutboundPartnerLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MarkOutboundPartnerLogic) MarkOutboundPartner(req *types.OutboundIDRequest) (resp *types.Response, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	if err := l.svcCtx.Outbound.MarkPartner(l.ctx, org, req.Id); err != nil {
		return nil, apiErr(err)
	}
	return ok("Marked as a partner."), nil
}
