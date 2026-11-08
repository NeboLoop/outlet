package outbound

import (
	"context"

	adminoutbound "github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SdkAddOutboundMailboxesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSdkAddOutboundMailboxesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SdkAddOutboundMailboxesLogic {
	return &SdkAddOutboundMailboxesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SdkAddOutboundMailboxesLogic) SdkAddOutboundMailboxes(req *types.OutboundAddMailboxesRequest) (resp *types.OutboundAddMailboxesResponse, err error) {
	org, err := brand(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	return adminoutbound.AddOutboundMailboxes(l.ctx, l.svcCtx, org, req)
}
