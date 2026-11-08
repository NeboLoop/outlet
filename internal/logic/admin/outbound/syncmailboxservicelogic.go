package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SyncMailboxServiceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSyncMailboxServiceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SyncMailboxServiceLogic {
	return &SyncMailboxServiceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SyncMailboxServiceLogic) SyncMailboxService(req *types.MailboxServiceSyncRequest) (resp *types.MailboxServiceSyncResponse, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Admin)
	if err != nil {
		return nil, err
	}
	return SyncMailboxService(l.ctx, l.svcCtx, org, req)
}

// SyncMailboxService compares the pool with its mailbox service, and imports the service's missing mailboxes when asked, for a brand already resolved: the admin route's
// signed-in person, or the API key of /sdk/v1/outbound.
func SyncMailboxService(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.MailboxServiceSyncRequest) (*types.MailboxServiceSyncResponse, error) {
	res, err := svcCtx.Outbound.SyncService(ctx, org, req.Id, req.Import)
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.MailboxServiceSyncResponse](res)
}
