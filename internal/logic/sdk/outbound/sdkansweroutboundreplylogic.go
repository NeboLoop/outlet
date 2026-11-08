package outbound

import (
	"context"

	adminoutbound "github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SdkAnswerOutboundReplyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSdkAnswerOutboundReplyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SdkAnswerOutboundReplyLogic {
	return &SdkAnswerOutboundReplyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SdkAnswerOutboundReplyLogic) SdkAnswerOutboundReply(req *types.OutboundAnswerRequest) (resp *types.Response, err error) {
	org, err := brand(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	return adminoutbound.AnswerOutboundReply(l.ctx, l.svcCtx, org, req)
}
