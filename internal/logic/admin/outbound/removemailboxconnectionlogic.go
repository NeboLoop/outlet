package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RemoveMailboxConnectionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRemoveMailboxConnectionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveMailboxConnectionLogic {
	return &RemoveMailboxConnectionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RemoveMailboxConnectionLogic) RemoveMailboxConnection(req *types.MailboxConnectionIDRequest) (resp *types.Response, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Admin)
	if err != nil {
		return nil, err
	}
	return RemoveMailboxConnection(l.ctx, l.svcCtx, org, req)
}

// RemoveMailboxConnection removes a connection that holds no domains, for a brand already resolved: the admin route's
// signed-in person, or the API key of /sdk/v1/outbound.
func RemoveMailboxConnection(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.MailboxConnectionIDRequest) (*types.Response, error) {
	if err := svcCtx.Outbound.RemoveConnection(ctx, org, req.Id); err != nil {
		return nil, apiErr(err)
	}
	return ok("Connection removed."), nil
}
