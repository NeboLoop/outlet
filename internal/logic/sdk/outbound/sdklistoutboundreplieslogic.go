package outbound

import (
	"context"

	adminoutbound "github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SdkListOutboundRepliesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSdkListOutboundRepliesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SdkListOutboundRepliesLogic {
	return &SdkListOutboundRepliesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SdkListOutboundRepliesLogic) SdkListOutboundReplies(req *types.OutboundListRepliesRequest) (resp *types.OutboundReplyList, err error) {
	org, err := brand(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	return adminoutbound.ListOutboundReplies(l.ctx, l.svcCtx, org, req)
}
