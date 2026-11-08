package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/services/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CheckOutboundDomainLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCheckOutboundDomainLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckOutboundDomainLogic {
	return &CheckOutboundDomainLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CheckOutboundDomainLogic) CheckOutboundDomain(req *types.OutboundIDRequest) (resp *types.OutboundDomainInfo, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	return CheckOutboundDomain(l.ctx, l.svcCtx, org, req)
}

// CheckOutboundDomain runs for a brand already resolved: the admin route's signed-in
// person, or the API key of /sdk/v1/outbound.
func CheckOutboundDomain(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.OutboundIDRequest) (*types.OutboundDomainInfo, error) {
	d, err := svcCtx.DB.GetOutreachDomain(ctx, db.GetOutreachDomainParams{ID: req.Id, OrgID: org.ID})
	if err != nil {
		return nil, errorx.NewNotFoundError("No such domain.")
	}
	if _, err := svcCtx.Outbound.CheckDomain(ctx, d); err != nil {
		return nil, apiErr(err)
	}
	if d, err = svcCtx.DB.GetOutreachDomain(ctx, db.GetOutreachDomainParams{ID: req.Id, OrgID: org.ID}); err != nil {
		return nil, err
	}
	return convert[types.OutboundDomainInfo](outbound.DomainInfoOf(d))
}
