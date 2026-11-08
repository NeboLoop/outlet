package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ConnectMailboxServiceDomainLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewConnectMailboxServiceDomainLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ConnectMailboxServiceDomainLogic {
	return &ConnectMailboxServiceDomainLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ConnectMailboxServiceDomainLogic) ConnectMailboxServiceDomain(req *types.MailboxServiceConnectRequest) (resp *types.MailboxServiceDomainResponse, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Admin)
	if err != nil {
		return nil, err
	}
	return ConnectMailboxServiceDomain(l.ctx, l.svcCtx, org, req)
}

// ConnectMailboxServiceDomain connects a domain the brand owns to the pool's mailbox service, for a brand already resolved: the admin route's
// signed-in person, or the API key of /sdk/v1/outbound.
func ConnectMailboxServiceDomain(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.MailboxServiceConnectRequest) (*types.MailboxServiceDomainResponse, error) {
	res, err := svcCtx.Outbound.ConnectServiceDomain(ctx, org, req.Id, req.Domain)
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.MailboxServiceDomainResponse](res)
}
