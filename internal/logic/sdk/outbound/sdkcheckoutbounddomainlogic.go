package outbound

import (
	"context"

	adminoutbound "github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SdkCheckOutboundDomainLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSdkCheckOutboundDomainLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SdkCheckOutboundDomainLogic {
	return &SdkCheckOutboundDomainLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SdkCheckOutboundDomainLogic) SdkCheckOutboundDomain(req *types.OutboundIDRequest) (resp *types.OutboundDomainInfo, err error) {
	org, err := brand(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	return adminoutbound.CheckOutboundDomain(l.ctx, l.svcCtx, org, req)
}
