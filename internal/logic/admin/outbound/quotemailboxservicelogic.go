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

type QuoteMailboxServiceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQuoteMailboxServiceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QuoteMailboxServiceLogic {
	return &QuoteMailboxServiceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QuoteMailboxServiceLogic) QuoteMailboxService(req *types.MailboxServiceQuoteRequest) (resp *types.MailboxServiceQuote, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	return QuoteMailboxService(l.ctx, l.svcCtx, org, req)
}

// QuoteMailboxService prices a domain, a connection or mailboxes at the pool's mailbox service; it buys nothing, for a brand already resolved: the admin route's
// signed-in person, or the API key of /sdk/v1/outbound.
func QuoteMailboxService(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.MailboxServiceQuoteRequest) (*types.MailboxServiceQuote, error) {
	in, err := convertInto[outbound.QuoteInput](req)
	if err != nil {
		return nil, err
	}
	res, err := svcCtx.Outbound.QuoteService(ctx, org, req.Id, in)
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.MailboxServiceQuote](res)
}
