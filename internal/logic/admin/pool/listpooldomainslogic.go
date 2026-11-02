package pool

import (
	"context"
	"github.com/outlet-sh/outlet/internal/access"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListPoolDomainsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListPoolDomainsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListPoolDomainsLogic {
	return &ListPoolDomainsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListPoolDomainsLogic) ListPoolDomains() (resp *types.ListPoolDomainsResponse, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return nil, err
	}
	ds, err := l.svcCtx.MailPool.ListDomains(l.ctx)
	if err != nil {
		return nil, err
	}
	resp = &types.ListPoolDomainsResponse{Domains: make([]types.PoolDomainInfo, 0, len(ds))}
	for _, d := range ds {
		resp.Domains = append(resp.Domains, domainInfo(d))
	}
	return resp, nil
}
