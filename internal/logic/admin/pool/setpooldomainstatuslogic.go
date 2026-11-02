package pool

import (
	"context"
	"github.com/outlet-sh/outlet/internal/access"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetPoolDomainStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSetPoolDomainStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetPoolDomainStatusLogic {
	return &SetPoolDomainStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SetPoolDomainStatusLogic) SetPoolDomainStatus(req *types.SetPoolStatusRequest) (resp *types.PoolDomainInfo, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return nil, err
	}
	d, err := l.svcCtx.MailPool.SetDomainStatus(l.ctx, req.Id, req.Status, req.Reason)
	if err != nil {
		return nil, err
	}
	info := domainInfo(d)
	return &info, nil
}
