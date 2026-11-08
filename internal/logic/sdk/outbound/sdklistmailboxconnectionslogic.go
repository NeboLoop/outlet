package outbound

import (
	"context"

	adminoutbound "github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SdkListMailboxConnectionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSdkListMailboxConnectionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SdkListMailboxConnectionsLogic {
	return &SdkListMailboxConnectionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SdkListMailboxConnectionsLogic) SdkListMailboxConnections(req *types.MailboxConnectionListRequest) (resp *types.MailboxConnectionList, err error) {
	org, err := brand(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	return adminoutbound.ListMailboxConnections(l.ctx, l.svcCtx, org, req)
}
