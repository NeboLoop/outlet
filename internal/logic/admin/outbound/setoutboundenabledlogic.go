package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetOutboundEnabledLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSetOutboundEnabledLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetOutboundEnabledLogic {
	return &SetOutboundEnabledLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SetOutboundEnabledLogic) SetOutboundEnabled(req *types.OutboundEnabledRequest) (resp *types.Response, err error) {
	// Only the brand's admin, and only here: no MCP tool can turn outbound on.
	org, err := brand(l.ctx, l.svcCtx, access.Admin)
	if err != nil {
		return nil, err
	}
	if err := l.svcCtx.Outbound.SetEnabled(l.ctx, org.ID, req.Enabled); err != nil {
		return nil, err
	}
	if req.Enabled {
		return ok("Outbound is on for " + org.Name + "."), nil
	}
	return ok("Outbound is off for " + org.Name + "; nothing sends."), nil
}
