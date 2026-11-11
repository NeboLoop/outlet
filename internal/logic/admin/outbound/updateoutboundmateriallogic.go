package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateOutboundMaterialLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateOutboundMaterialLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateOutboundMaterialLogic {
	return &UpdateOutboundMaterialLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateOutboundMaterialLogic) UpdateOutboundMaterial(req *types.UpdateOutboundMaterialRequest) (resp *types.OutboundMaterial, err error) {
	return saveMaterial(l.ctx, l.svcCtx, req.Id, req)
}
