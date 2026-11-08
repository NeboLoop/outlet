package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
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
	return ExplainOutbound(l.ctx, l.svcCtx, org, req)
}

// ExplainOutbound runs for a brand already resolved: the admin route's signed-in
// person, or the API key of /sdk/v1/outbound.
func ExplainOutbound(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.OutboundExplainRequest) (*types.OutboundExplainResponse, error) {
	a, err := svcCtx.Outbound.Explain(ctx, org, req.MessageId, req.MailboxId, req.ProspectId)
	if err != nil {
		return nil, apiErr(err)
	}
	return &types.OutboundExplainResponse{Answer: a}, nil
}
