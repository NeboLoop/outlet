package outbound

import (
	"context"

	adminoutbound "github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SdkClassifyOutboundReplyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSdkClassifyOutboundReplyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SdkClassifyOutboundReplyLogic {
	return &SdkClassifyOutboundReplyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SdkClassifyOutboundReplyLogic) SdkClassifyOutboundReply(req *types.OutboundClassifyRequest) (resp *types.Response, err error) {
	org, err := brand(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	return adminoutbound.ClassifyOutboundReply(l.ctx, l.svcCtx, org, req)
}
