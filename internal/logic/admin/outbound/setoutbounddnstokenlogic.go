package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetOutboundDNSTokenLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSetOutboundDNSTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetOutboundDNSTokenLogic {
	return &SetOutboundDNSTokenLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SetOutboundDNSTokenLogic) SetOutboundDNSToken(req *types.OutboundDNSTokenRequest) (resp *types.Response, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Admin)
	if err != nil {
		return nil, err
	}
	return SetOutboundDNSToken(l.ctx, l.svcCtx, org, req)
}

// SetOutboundDNSToken runs for a brand already resolved: the admin route's signed-in
// person, or the API key of /sdk/v1/outbound.
func SetOutboundDNSToken(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.OutboundDNSTokenRequest) (*types.Response, error) {
	if err := svcCtx.Outbound.SetDNSToken(ctx, org.ID, req.Token); err != nil {
		return nil, apiErr(err)
	}
	return ok("DigitalOcean token saved, encrypted."), nil
}
