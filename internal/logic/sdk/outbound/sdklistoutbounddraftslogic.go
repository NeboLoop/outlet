package outbound

import (
	"context"

	adminoutbound "github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SdkListOutboundDraftsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSdkListOutboundDraftsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SdkListOutboundDraftsLogic {
	return &SdkListOutboundDraftsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SdkListOutboundDraftsLogic) SdkListOutboundDrafts(req *types.OutboundListRequest) (resp *types.OutboundDraftList, err error) {
	org, err := brand(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	return adminoutbound.ListOutboundDrafts(l.ctx, l.svcCtx, org, req)
}
