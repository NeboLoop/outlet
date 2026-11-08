package outbound

import (
	"context"

	adminoutbound "github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SdkUpdateMailboxConnectionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSdkUpdateMailboxConnectionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SdkUpdateMailboxConnectionLogic {
	return &SdkUpdateMailboxConnectionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SdkUpdateMailboxConnectionLogic) SdkUpdateMailboxConnection(req *types.UpdateMailboxConnectionRequest) (resp *types.MailboxConnection, err error) {
	org, err := brand(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	return adminoutbound.UpdateMailboxConnection(l.ctx, l.svcCtx, org, req)
}
