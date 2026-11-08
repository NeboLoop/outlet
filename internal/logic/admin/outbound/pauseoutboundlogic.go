package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
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
	if err := l.svcCtx.Outbound.Pause(l.ctx, org, req.Scope, req.Id, req.Resume); err != nil {
		return nil, apiErr(err)
	}
	if req.Resume {
		return ok("Resumed."), nil
	}
	return ok("Paused."), nil
}
