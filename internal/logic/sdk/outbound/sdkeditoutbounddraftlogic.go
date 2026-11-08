package outbound

import (
	"context"

	adminoutbound "github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SdkEditOutboundDraftLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSdkEditOutboundDraftLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SdkEditOutboundDraftLogic {
	return &SdkEditOutboundDraftLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SdkEditOutboundDraftLogic) SdkEditOutboundDraft(req *types.OutboundEditDraftRequest) (resp *types.Response, err error) {
	org, err := brand(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	return adminoutbound.EditOutboundDraft(l.ctx, l.svcCtx, org, req)
}
