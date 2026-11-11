package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/services/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateOutboundDoNotContactSettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateOutboundDoNotContactSettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateOutboundDoNotContactSettingsLogic {
	return &UpdateOutboundDoNotContactSettingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateOutboundDoNotContactSettingsLogic) UpdateOutboundDoNotContactSettings(req *types.OutboundDNCSettingsRequest) (resp *types.OutboundDNCSettings, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Admin)
	if err != nil {
		return nil, err
	}
	st, err := l.svcCtx.Outbound.UpdateDNCSettings(l.ctx, org, outbound.DNCSettingsInput{CustomerDomains: req.CustomerDomains, CheckURL: req.CheckURL, CheckSecret: req.CheckSecret})
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.OutboundDNCSettings](st)
}
