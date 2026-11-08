package outbound

import (
	"context"

	adminoutbound "github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SdkAddMailboxConnectionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSdkAddMailboxConnectionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SdkAddMailboxConnectionLogic {
	return &SdkAddMailboxConnectionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SdkAddMailboxConnectionLogic) SdkAddMailboxConnection(req *types.MailboxConnectionRequest) (resp *types.MailboxConnection, err error) {
	org, err := brand(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	return adminoutbound.AddMailboxConnection(l.ctx, l.svcCtx, org, req)
}
