package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteOutboundDoNotContactListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteOutboundDoNotContactListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteOutboundDoNotContactListLogic {
	return &DeleteOutboundDoNotContactListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteOutboundDoNotContactListLogic) DeleteOutboundDoNotContactList(req *types.OutboundDNCDeleteRequest) (resp *types.Response, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	if _, err := l.svcCtx.Outbound.DeleteDoNotContactList(l.ctx, org, req.List); err != nil {
		return nil, apiErr(err)
	}
	return ok("Deleted."), nil
}
