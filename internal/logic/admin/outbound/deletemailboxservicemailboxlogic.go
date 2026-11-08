package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteMailboxServiceMailboxLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteMailboxServiceMailboxLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteMailboxServiceMailboxLogic {
	return &DeleteMailboxServiceMailboxLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteMailboxServiceMailboxLogic) DeleteMailboxServiceMailbox(req *types.MailboxServiceDeleteRequest) (resp *types.Response, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Admin)
	if err != nil {
		return nil, err
	}
	return DeleteMailboxServiceMailbox(l.ctx, l.svcCtx, org, req)
}

// DeleteMailboxServiceMailbox deletes or cancels a pool mailbox at its mailbox service, for a brand already resolved: the admin route's
// signed-in person, or the API key of /sdk/v1/outbound.
func DeleteMailboxServiceMailbox(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.MailboxServiceDeleteRequest) (*types.Response, error) {
	if err := svcCtx.Outbound.DeleteServiceMailbox(ctx, org, req.Id, req.MailboxId); err != nil {
		return nil, apiErr(err)
	}
	return ok("Deleted at the mailbox service; it stays in the pool, paused."), nil
}
