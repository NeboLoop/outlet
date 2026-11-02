package pool

import (
	"context"
	"github.com/outlet-sh/outlet/internal/access"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListPoolMailboxesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListPoolMailboxesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListPoolMailboxesLogic {
	return &ListPoolMailboxesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListPoolMailboxesLogic) ListPoolMailboxes() (resp *types.ListPoolMailboxesResponse, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return nil, err
	}
	mbs, err := l.svcCtx.MailPool.ListMailboxes(l.ctx)
	if err != nil {
		return nil, err
	}
	resp = &types.ListPoolMailboxesResponse{Mailboxes: make([]types.PoolMailboxInfo, 0, len(mbs))}
	for _, mb := range mbs {
		resp.Mailboxes = append(resp.Mailboxes, mailboxInfo(mb))
	}
	return resp, nil
}
