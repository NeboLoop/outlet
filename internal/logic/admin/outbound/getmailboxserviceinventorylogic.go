package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMailboxServiceInventoryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetMailboxServiceInventoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMailboxServiceInventoryLogic {
	return &GetMailboxServiceInventoryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMailboxServiceInventoryLogic) GetMailboxServiceInventory(req *types.MailboxConnectionIDRequest) (resp *types.MailboxServiceInventory, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	return GetMailboxServiceInventory(l.ctx, l.svcCtx, org, req)
}

// GetMailboxServiceInventory lists the domains and mailboxes the brand has at a connection's mailbox service, for a brand already resolved: the admin route's
// signed-in person, or the API key of /sdk/v1/outbound.
func GetMailboxServiceInventory(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.MailboxConnectionIDRequest) (*types.MailboxServiceInventory, error) {
	res, err := svcCtx.Outbound.MailboxServiceInventory(ctx, org, req.Id)
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.MailboxServiceInventory](res)
}
