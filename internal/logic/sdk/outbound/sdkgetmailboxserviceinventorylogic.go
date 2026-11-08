package outbound

import (
	"context"

	adminoutbound "github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SdkGetMailboxServiceInventoryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSdkGetMailboxServiceInventoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SdkGetMailboxServiceInventoryLogic {
	return &SdkGetMailboxServiceInventoryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SdkGetMailboxServiceInventoryLogic) SdkGetMailboxServiceInventory(req *types.MailboxServicePoolRequest) (resp *types.MailboxServiceInventory, err error) {
	org, err := brand(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	return adminoutbound.GetMailboxServiceInventory(l.ctx, l.svcCtx, org, req)
}
