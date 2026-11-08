package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListMailboxConnectionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListMailboxConnectionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListMailboxConnectionsLogic {
	return &ListMailboxConnectionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListMailboxConnectionsLogic) ListMailboxConnections(req *types.MailboxConnectionListRequest) (resp *types.MailboxConnectionList, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	return ListMailboxConnections(l.ctx, l.svcCtx, org, req)
}

// ListMailboxConnections lists the brand's mailbox service connections, one pool's or every pool's, for a brand already resolved: the admin route's
// signed-in person, or the API key of /sdk/v1/outbound.
func ListMailboxConnections(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.MailboxConnectionListRequest) (*types.MailboxConnectionList, error) {
	res, err := svcCtx.Outbound.ListConnections(ctx, org, req.PoolId)
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.MailboxConnectionList](map[string]any{"connections": res})
}
