package outbound

import (
	"context"

	adminoutbound "github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SdkAddOutboundProspectsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSdkAddOutboundProspectsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SdkAddOutboundProspectsLogic {
	return &SdkAddOutboundProspectsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SdkAddOutboundProspectsLogic) SdkAddOutboundProspects(req *types.OutboundAddProspectsRequest) (resp *types.OutboundAddProspectsResponse, err error) {
	org, err := brand(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	return adminoutbound.AddOutboundProspects(l.ctx, l.svcCtx, org, req)
}
