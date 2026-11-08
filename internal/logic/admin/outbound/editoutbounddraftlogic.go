package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
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
	return EditOutboundDraft(l.ctx, l.svcCtx, org, req)
}

// EditOutboundDraft runs for a brand already resolved: the admin route's signed-in
// person, or the API key of /sdk/v1/outbound.
func EditOutboundDraft(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.OutboundEditDraftRequest) (*types.Response, error) {
	if err := svcCtx.Outbound.EditDraft(ctx, org, req.Id, req.Subject, req.Body); err != nil {
		return nil, apiErr(err)
	}
	return ok("Saved."), nil
}
