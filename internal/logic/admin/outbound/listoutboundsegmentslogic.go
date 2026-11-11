package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListOutboundSegmentsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListOutboundSegmentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListOutboundSegmentsLogic {
	return &ListOutboundSegmentsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListOutboundSegmentsLogic) ListOutboundSegments() (resp *types.OutboundSegmentList, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	segs, err := l.svcCtx.Outbound.Segments(l.ctx, org)
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.OutboundSegmentList](map[string]any{"segments": segs})
}
