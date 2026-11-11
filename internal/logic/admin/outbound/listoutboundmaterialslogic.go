package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListOutboundMaterialsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListOutboundMaterialsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListOutboundMaterialsLogic {
	return &ListOutboundMaterialsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListOutboundMaterialsLogic) ListOutboundMaterials(req *types.OutboundMaterialListRequest) (resp *types.OutboundMaterialList, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	mats, err := l.svcCtx.Outbound.Materials(l.ctx, org, req.Kind, req.Segment)
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.OutboundMaterialList](map[string]any{"materials": mats})
}
