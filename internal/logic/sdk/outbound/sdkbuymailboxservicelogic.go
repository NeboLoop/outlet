package outbound

import (
	"context"

	adminoutbound "github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SdkBuyMailboxServiceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSdkBuyMailboxServiceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SdkBuyMailboxServiceLogic {
	return &SdkBuyMailboxServiceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SdkBuyMailboxServiceLogic) SdkBuyMailboxService(req *types.MailboxServiceBuyRequest) (resp *types.MailboxServiceBuyResponse, err error) {
	org, err := brand(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	return adminoutbound.BuyMailboxService(l.ctx, l.svcCtx, org, req)
}
