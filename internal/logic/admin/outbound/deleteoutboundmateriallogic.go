package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteOutboundMaterialLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteOutboundMaterialLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteOutboundMaterialLogic {
	return &DeleteOutboundMaterialLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteOutboundMaterialLogic) DeleteOutboundMaterial(req *types.OutboundMaterialIDRequest) (resp *types.Response, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	if err := l.svcCtx.Outbound.DeleteMaterial(l.ctx, org, req.Id); err != nil {
		return nil, apiErr(err)
	}
	return ok("Deleted."), nil
}
