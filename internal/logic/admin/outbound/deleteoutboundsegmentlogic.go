package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteOutboundSegmentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteOutboundSegmentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteOutboundSegmentLogic {
	return &DeleteOutboundSegmentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteOutboundSegmentLogic) DeleteOutboundSegment(req *types.OutboundIDRequest) (resp *types.Response, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	if err := l.svcCtx.Outbound.DeleteSegment(l.ctx, org, req.Id); err != nil {
		return nil, apiErr(err)
	}
	return ok("Removed."), nil
}
