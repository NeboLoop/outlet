package outbound

import (
	"context"

	adminoutbound "github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SdkRemoveMailboxConnectionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSdkRemoveMailboxConnectionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SdkRemoveMailboxConnectionLogic {
	return &SdkRemoveMailboxConnectionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SdkRemoveMailboxConnectionLogic) SdkRemoveMailboxConnection(req *types.MailboxConnectionIDRequest) (resp *types.Response, err error) {
	org, err := brand(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	return adminoutbound.RemoveMailboxConnection(l.ctx, l.svcCtx, org, req)
}
