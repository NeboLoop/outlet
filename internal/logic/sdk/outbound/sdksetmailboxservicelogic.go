package outbound

import (
	"context"

	adminoutbound "github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SdkSetMailboxServiceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSdkSetMailboxServiceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SdkSetMailboxServiceLogic {
	return &SdkSetMailboxServiceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SdkSetMailboxServiceLogic) SdkSetMailboxService(req *types.MailboxServiceRequest) (resp *types.MailboxServiceStatus, err error) {
	org, err := brand(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	return adminoutbound.SetMailboxService(l.ctx, l.svcCtx, org, req)
}
