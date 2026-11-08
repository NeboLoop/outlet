package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddMailboxConnectionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddMailboxConnectionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddMailboxConnectionLogic {
	return &AddMailboxConnectionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddMailboxConnectionLogic) AddMailboxConnection(req *types.MailboxConnectionRequest) (resp *types.MailboxConnection, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Admin)
	if err != nil {
		return nil, err
	}
	return AddMailboxConnection(l.ctx, l.svcCtx, org, req)
}

// AddMailboxConnection adds a mailbox service connection to a pool; a service's key is tried before it is saved, for a brand already resolved: the admin route's
// signed-in person, or the API key of /sdk/v1/outbound.
func AddMailboxConnection(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.MailboxConnectionRequest) (*types.MailboxConnection, error) {
	in, err := convertInto[outbound.ConnectionInput](req)
	if err != nil {
		return nil, err
	}
	res, err := svcCtx.Outbound.AddConnection(ctx, org, in)
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.MailboxConnection](res)
}
