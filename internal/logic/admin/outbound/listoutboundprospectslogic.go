package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
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
	limit := int64(req.Limit)
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := svcCtx.DB.ListProspects(ctx, db.ListProspectsParams{OrgID: org.ID, ListID: req.ListId, LimitCount: limit})
	if err != nil {
		return nil, err
	}
	out := &types.OutboundProspectList{Prospects: make([]types.OutboundProspectInfo, 0, len(rows))}
	for _, p := range rows {
		out.Prospects = append(out.Prospects, types.OutboundProspectInfo{
			Id: p.ID, Email: p.Email, Name: p.Name, EmailSource: p.Source.String, ValidationStatus: p.ValidationStatus.String,
			ListId: p.ListID, ListName: p.ListName, Status: p.Status.String, CreatedAt: p.CreatedAt.String,
		})
	}
	return out, nil
}
