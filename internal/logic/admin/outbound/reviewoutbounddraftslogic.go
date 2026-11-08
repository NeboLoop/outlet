package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
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
	if req.Action != "approve" && req.Action != "skip" {
		return nil, errorx.NewBadRequestError("action must be approve or skip")
	}
	r, err := l.svcCtx.Outbound.Review(l.ctx, org, req.Action, req.Ids)
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.OutboundReviewResponse](r)
}
