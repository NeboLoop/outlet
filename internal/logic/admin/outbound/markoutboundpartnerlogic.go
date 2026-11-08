package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type MarkOutboundPartnerLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewMarkOutboundPartnerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MarkOutboundPartnerLogic {
	return &MarkOutboundPartnerLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MarkOutboundPartnerLogic) MarkOutboundPartner(req *types.OutboundIDRequest) (resp *types.Response, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	return MarkOutboundPartner(l.ctx, l.svcCtx, org, req)
}

// MarkOutboundPartner runs for a brand already resolved: the admin route's signed-in
// person, or the API key of /sdk/v1/outbound.
func MarkOutboundPartner(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.OutboundIDRequest) (*types.Response, error) {
	if err := svcCtx.Outbound.MarkPartner(ctx, org, req.Id); err != nil {
		return nil, apiErr(err)
	}
	return ok("Marked as a partner."), nil
}
