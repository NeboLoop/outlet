package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateOutboundSettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateOutboundSettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateOutboundSettingsLogic {
	return &UpdateOutboundSettingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateOutboundSettingsLogic) UpdateOutboundSettings(req *types.OutboundSettingsRequest) (resp *types.Response, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	return UpdateOutboundSettings(l.ctx, l.svcCtx, org, req)
}

// UpdateOutboundSettings runs for a brand already resolved: the admin route's signed-in
// person, or the API key of /sdk/v1/outbound.
func UpdateOutboundSettings(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.OutboundSettingsRequest) (*types.Response, error) {
	if _, err := svcCtx.Outbound.UpdateSettings(ctx, org, outbound.SettingsInput{
		MailingAddress: req.MailingAddress, OptOutLine: req.OptOutLine, Timezone: req.Timezone,
	}); err != nil {
		return nil, apiErr(err)
	}
	return ok("Saved."), nil
}
