package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AnswerOutboundReplyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAnswerOutboundReplyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AnswerOutboundReplyLogic {
	return &AnswerOutboundReplyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AnswerOutboundReplyLogic) AnswerOutboundReply(req *types.OutboundAnswerRequest) (resp *types.Response, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	if _, err := l.svcCtx.Outbound.Answer(l.ctx, org, req.Id, req.Body); err != nil {
		return nil, apiErr(err)
	}
	return ok("Queued. It goes from the same mailbox, in the same thread, within its daily limit."), nil
}
