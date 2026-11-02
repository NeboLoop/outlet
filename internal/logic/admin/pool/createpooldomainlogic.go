package pool

import (
	"context"
	"github.com/outlet-sh/outlet/internal/access"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreatePoolDomainLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreatePoolDomainLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreatePoolDomainLogic {
	return &CreatePoolDomainLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreatePoolDomainLogic) CreatePoolDomain(req *types.CreatePoolDomainRequest) (resp *types.PoolDomainInfo, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return nil, err
	}
	d, err := l.svcCtx.MailPool.CreateDomain(l.ctx, req.Domain)
	if err != nil {
		return nil, err
	}
	info := domainInfo(d)
	return &info, nil
}
