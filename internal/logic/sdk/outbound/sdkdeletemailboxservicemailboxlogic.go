package outbound

import (
	"context"

	adminoutbound "github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SdkDeleteMailboxServiceMailboxLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSdkDeleteMailboxServiceMailboxLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SdkDeleteMailboxServiceMailboxLogic {
	return &SdkDeleteMailboxServiceMailboxLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SdkDeleteMailboxServiceMailboxLogic) SdkDeleteMailboxServiceMailbox(req *types.MailboxServiceDeleteRequest) (resp *types.Response, err error) {
	org, err := brand(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	return adminoutbound.DeleteMailboxServiceMailbox(l.ctx, l.svcCtx, org, req)
}
