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

type UpdateMailboxConnectionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateMailboxConnectionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateMailboxConnectionLogic {
	return &UpdateMailboxConnectionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateMailboxConnectionLogic) UpdateMailboxConnection(req *types.UpdateMailboxConnectionRequest) (resp *types.MailboxConnection, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Admin)
	if err != nil {
		return nil, err
	}
	return UpdateMailboxConnection(l.ctx, l.svcCtx, org, req)
}

// UpdateMailboxConnection renames a connection, replaces its key and sets its mailboxes' daily limit and warm-up, for a brand already resolved: the admin route's
// signed-in person, or the API key of /sdk/v1/outbound.
func UpdateMailboxConnection(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.UpdateMailboxConnectionRequest) (*types.MailboxConnection, error) {
	res, err := svcCtx.Outbound.UpdateConnection(ctx, org, req.Id, outbound.ConnectionInput{
		Name: req.Name, APIKey: req.APIKey,
		ConnectionLimits: outbound.ConnectionLimits{DailyLimit: req.DailyLimit, WarmupStart: req.WarmupStart, WarmupStep: req.WarmupStep},
	})
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.MailboxConnection](res)
}
