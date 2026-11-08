package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ClassifyOutboundReplyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewClassifyOutboundReplyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ClassifyOutboundReplyLogic {
	return &ClassifyOutboundReplyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ClassifyOutboundReplyLogic) ClassifyOutboundReply(req *types.OutboundClassifyRequest) (resp *types.Response, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	return ClassifyOutboundReply(l.ctx, l.svcCtx, org, req)
}

// ClassifyOutboundReply runs for a brand already resolved: the admin route's signed-in
// person, or the API key of /sdk/v1/outbound.
func ClassifyOutboundReply(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.OutboundClassifyRequest) (*types.Response, error) {
	if err := svcCtx.Outbound.Classify(ctx, org, req.Id, req.Classification); err != nil {
		return nil, apiErr(err)
	}
	return ok("Saved."), nil
}
