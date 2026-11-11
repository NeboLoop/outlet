package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateOutboundSegmentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateOutboundSegmentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateOutboundSegmentLogic {
	return &UpdateOutboundSegmentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateOutboundSegmentLogic) UpdateOutboundSegment(req *types.UpdateOutboundSegmentRequest) (resp *types.OutboundSegment, err error) {
	return saveSegment(l.ctx, l.svcCtx, req.Id, req)
}
