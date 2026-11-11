package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateOutboundSegmentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateOutboundSegmentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateOutboundSegmentLogic {
	return &CreateOutboundSegmentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateOutboundSegmentLogic) CreateOutboundSegment(req *types.OutboundSegmentRequest) (resp *types.OutboundSegment, err error) {
	return saveSegment(l.ctx, l.svcCtx, "", req)
}
