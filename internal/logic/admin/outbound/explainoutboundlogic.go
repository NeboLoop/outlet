package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ExplainOutboundLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewExplainOutboundLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ExplainOutboundLogic {
	return &ExplainOutboundLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ExplainOutboundLogic) ExplainOutbound(req *types.OutboundExplainRequest) (resp *types.OutboundExplainResponse, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	a, err := l.svcCtx.Outbound.Explain(l.ctx, org, req.MessageId, req.MailboxId, req.ProspectId)
	if err != nil {
		return nil, apiErr(err)
	}
	return &types.OutboundExplainResponse{Answer: a}, nil
}
