package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type MoveOutboundDomainConnectionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewMoveOutboundDomainConnectionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MoveOutboundDomainConnectionLogic {
	return &MoveOutboundDomainConnectionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MoveOutboundDomainConnectionLogic) MoveOutboundDomainConnection(req *types.MoveDomainConnectionRequest) (resp *types.MailboxConnection, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Admin)
	if err != nil {
		return nil, err
	}
	return MoveOutboundDomainConnection(l.ctx, l.svcCtx, org, req)
}

// MoveOutboundDomainConnection moves an outreach domain, with its mailboxes, to another connection of its pool, for a brand already resolved: the admin route's
// signed-in person, or the API key of /sdk/v1/outbound.
func MoveOutboundDomainConnection(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.MoveDomainConnectionRequest) (*types.MailboxConnection, error) {
	res, err := svcCtx.Outbound.MoveDomainToConnection(ctx, org, req.Id, req.ConnectionId)
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.MailboxConnection](res)
}
