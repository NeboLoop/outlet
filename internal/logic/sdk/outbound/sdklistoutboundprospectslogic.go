package outbound

import (
	"context"

	adminoutbound "github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SdkListOutboundProspectsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSdkListOutboundProspectsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SdkListOutboundProspectsLogic {
	return &SdkListOutboundProspectsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SdkListOutboundProspectsLogic) SdkListOutboundProspects(req *types.OutboundListProspectsRequest) (resp *types.OutboundProspectList, err error) {
	org, err := brand(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	return adminoutbound.ListOutboundProspects(l.ctx, l.svcCtx, org, req)
}
