package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/services/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOutboundDoNotContactLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetOutboundDoNotContactLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOutboundDoNotContactLogic {
	return &GetOutboundDoNotContactLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetOutboundDoNotContactLogic) GetOutboundDoNotContact() (resp *types.OutboundDNC, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	lists, err := l.svcCtx.Outbound.DoNotContactLists(l.ctx, org)
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.OutboundDNC](map[string]any{"settings": outbound.DNCSettingsOf(org), "lists": lists})
}
