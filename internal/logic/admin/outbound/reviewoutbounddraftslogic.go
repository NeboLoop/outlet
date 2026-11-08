package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReviewOutboundDraftsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewReviewOutboundDraftsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReviewOutboundDraftsLogic {
	return &ReviewOutboundDraftsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ReviewOutboundDraftsLogic) ReviewOutboundDrafts(req *types.OutboundReviewRequest) (resp *types.OutboundReviewResponse, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	return ReviewOutboundDrafts(l.ctx, l.svcCtx, org, req)
}

// ReviewOutboundDrafts runs for a brand already resolved: the admin route's signed-in
// person, or the API key of /sdk/v1/outbound.
func ReviewOutboundDrafts(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.OutboundReviewRequest) (*types.OutboundReviewResponse, error) {
	if req.Action != "approve" && req.Action != "skip" {
		return nil, errorx.NewBadRequestError("action must be approve or skip")
	}
	r, err := svcCtx.Outbound.Review(ctx, org, req.Action, req.Ids)
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.OutboundReviewResponse](r)
}
