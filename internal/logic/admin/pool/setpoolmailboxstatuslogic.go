package pool

import (
	"context"
	"github.com/outlet-sh/outlet/internal/access"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetPoolMailboxStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSetPoolMailboxStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetPoolMailboxStatusLogic {
	return &SetPoolMailboxStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SetPoolMailboxStatusLogic) SetPoolMailboxStatus(req *types.SetPoolStatusRequest) (resp *types.PoolMailboxInfo, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return nil, err
	}
	mb, err := l.svcCtx.MailPool.SetMailboxStatus(l.ctx, req.Id, req.Status, req.Reason)
	if err != nil {
		return nil, err
	}
	info := mailboxInfo(mb)
	return &info, nil
}
