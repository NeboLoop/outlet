package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetMailboxServiceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSetMailboxServiceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetMailboxServiceLogic {
	return &SetMailboxServiceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SetMailboxServiceLogic) SetMailboxService(req *types.MailboxServiceRequest) (resp *types.MailboxServiceStatus, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Admin)
	if err != nil {
		return nil, err
	}
	return SetMailboxService(l.ctx, l.svcCtx, org, req)
}

// SetMailboxService picks the pool's mailbox service and stores its API key, for a brand already resolved: the admin route's
// signed-in person, or the API key of /sdk/v1/outbound.
func SetMailboxService(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.MailboxServiceRequest) (*types.MailboxServiceStatus, error) {
	res, err := svcCtx.Outbound.SetMailboxService(ctx, org, req.Id, req.Service, req.APIKey)
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.MailboxServiceStatus](res)
}
