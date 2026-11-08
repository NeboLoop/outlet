package outbound

import (
	"context"

	adminoutbound "github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SdkMarkOutboundPartnerLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSdkMarkOutboundPartnerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SdkMarkOutboundPartnerLogic {
	return &SdkMarkOutboundPartnerLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SdkMarkOutboundPartnerLogic) SdkMarkOutboundPartner(req *types.OutboundIDRequest) (resp *types.Response, err error) {
	org, err := brand(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	return adminoutbound.MarkOutboundPartner(l.ctx, l.svcCtx, org, req)
}
