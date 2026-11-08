package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type BuyMailboxServiceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBuyMailboxServiceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BuyMailboxServiceLogic {
	return &BuyMailboxServiceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BuyMailboxServiceLogic) BuyMailboxService(req *types.MailboxServiceBuyRequest) (resp *types.MailboxServiceBuyResponse, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Admin)
	if err != nil {
		return nil, err
	}
	return BuyMailboxService(l.ctx, l.svcCtx, org, req)
}

// BuyMailboxService buys what a quote priced; it needs confirm, for a brand already resolved: the admin route's
// signed-in person, or the API key of /sdk/v1/outbound.
func BuyMailboxService(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.MailboxServiceBuyRequest) (*types.MailboxServiceBuyResponse, error) {
	res, err := svcCtx.Outbound.BuyService(ctx, org, req.Id, req.QuoteId, req.Confirm)
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.MailboxServiceBuyResponse](res)
}
