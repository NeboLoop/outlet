package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateOutboundMaterialLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateOutboundMaterialLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateOutboundMaterialLogic {
	return &CreateOutboundMaterialLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateOutboundMaterialLogic) CreateOutboundMaterial(req *types.OutboundMaterialRequest) (resp *types.OutboundMaterial, err error) {
	return saveMaterial(l.ctx, l.svcCtx, "", req)
}
