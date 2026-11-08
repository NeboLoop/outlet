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

type ListOutboundProspectsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListOutboundProspectsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListOutboundProspectsLogic {
	return &ListOutboundProspectsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListOutboundProspectsLogic) ListOutboundProspects(req *types.OutboundListProspectsRequest) (resp *types.OutboundProspectList, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	return ListOutboundProspects(l.ctx, l.svcCtx, org, req)
}

// ListOutboundProspects runs for a brand already resolved: the admin route's signed-in
// person, or the API key of /sdk/v1/outbound.
func ListOutboundProspects(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.OutboundListProspectsRequest) (*types.OutboundProspectList, error) {
	page, err := svcCtx.Outbound.ListProspects(ctx, org, outbound.ProspectQuery{
		ListID: req.ListId, Status: req.Status, Search: req.Search, Enrollable: req.Enrollable, Limit: req.Limit, Offset: req.Offset,
	})
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.OutboundProspectList](page)
}
