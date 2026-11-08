package outbound

import (
	"context"

	adminoutbound "github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SdkSetOutboundDNSTokenLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSdkSetOutboundDNSTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SdkSetOutboundDNSTokenLogic {
	return &SdkSetOutboundDNSTokenLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SdkSetOutboundDNSTokenLogic) SdkSetOutboundDNSToken(req *types.OutboundDNSTokenRequest) (resp *types.Response, err error) {
	org, err := brand(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	return adminoutbound.SetOutboundDNSToken(l.ctx, l.svcCtx, org, req)
}
