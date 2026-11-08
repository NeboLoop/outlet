package outbound

import (
	"context"

	adminoutbound "github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SdkExplainOutboundLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSdkExplainOutboundLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SdkExplainOutboundLogic {
	return &SdkExplainOutboundLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SdkExplainOutboundLogic) SdkExplainOutbound(req *types.OutboundExplainRequest) (resp *types.OutboundExplainResponse, err error) {
	org, err := brand(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	return adminoutbound.ExplainOutbound(l.ctx, l.svcCtx, org, req)
}
