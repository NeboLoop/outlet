package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PauseOutboundLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPauseOutboundLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PauseOutboundLogic {
	return &PauseOutboundLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PauseOutboundLogic) PauseOutbound(req *types.OutboundPauseRequest) (resp *types.Response, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	return PauseOutbound(l.ctx, l.svcCtx, org, req)
}

// PauseOutbound runs for a brand already resolved: the admin route's signed-in
// person, or the API key of /sdk/v1/outbound.
func PauseOutbound(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.OutboundPauseRequest) (*types.Response, error) {
	if err := svcCtx.Outbound.Pause(ctx, org, req.Scope, req.Id, req.Resume); err != nil {
		return nil, apiErr(err)
	}
	if req.Resume {
		return ok("Resumed."), nil
	}
	return ok("Paused."), nil
}
