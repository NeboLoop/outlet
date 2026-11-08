package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type EditOutboundDraftLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewEditOutboundDraftLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EditOutboundDraftLogic {
	return &EditOutboundDraftLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EditOutboundDraftLogic) EditOutboundDraft(req *types.OutboundEditDraftRequest) (resp *types.Response, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	if err := l.svcCtx.Outbound.EditDraft(l.ctx, org, req.Id, req.Subject, req.Body); err != nil {
		return nil, apiErr(err)
	}
	return ok("Saved."), nil
}
