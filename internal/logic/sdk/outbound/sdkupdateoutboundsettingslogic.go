package outbound

import (
	"context"

	adminoutbound "github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SdkUpdateOutboundSettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSdkUpdateOutboundSettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SdkUpdateOutboundSettingsLogic {
	return &SdkUpdateOutboundSettingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SdkUpdateOutboundSettingsLogic) SdkUpdateOutboundSettings(req *types.OutboundSettingsRequest) (resp *types.Response, err error) {
	org, err := brand(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	return adminoutbound.UpdateOutboundSettings(l.ctx, l.svcCtx, org, req)
}
