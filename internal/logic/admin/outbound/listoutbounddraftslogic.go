package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListOutboundDraftsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListOutboundDraftsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListOutboundDraftsLogic {
	return &ListOutboundDraftsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListOutboundDraftsLogic) ListOutboundDrafts(req *types.OutboundListRequest) (resp *types.OutboundDraftList, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	return ListOutboundDrafts(l.ctx, l.svcCtx, org, req)
}

// ListOutboundDrafts runs for a brand already resolved: the admin route's signed-in
// person, or the API key of /sdk/v1/outbound.
func ListOutboundDrafts(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.OutboundListRequest) (*types.OutboundDraftList, error) {
	d, err := svcCtx.Outbound.ListDrafts(ctx, org, req.Limit)
	if err != nil {
		return nil, err
	}
	out, err := convert[[]types.OutboundDraft](d)
	if err != nil {
		return nil, err
	}
	return &types.OutboundDraftList{Drafts: *out}, nil
}
