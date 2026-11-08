package outbound

import (
	"context"

	adminoutbound "github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SdkConnectMailboxServiceDomainLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSdkConnectMailboxServiceDomainLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SdkConnectMailboxServiceDomainLogic {
	return &SdkConnectMailboxServiceDomainLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SdkConnectMailboxServiceDomainLogic) SdkConnectMailboxServiceDomain(req *types.MailboxServiceConnectRequest) (resp *types.MailboxServiceDomainResponse, err error) {
	org, err := brand(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	return adminoutbound.ConnectMailboxServiceDomain(l.ctx, l.svcCtx, org, req)
}
