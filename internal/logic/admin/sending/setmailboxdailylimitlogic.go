package sending

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetMailboxDailyLimitLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSetMailboxDailyLimitLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetMailboxDailyLimitLogic {
	return &SetMailboxDailyLimitLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SetMailboxDailyLimitLogic) SetMailboxDailyLimit(req *types.MailboxDailyLimitRequest) (resp *types.Response, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Admin)
	if err != nil {
		return nil, err
	}
	if err := l.svcCtx.EmailService.SetMailboxDailyLimit(l.ctx, orgID, req.Id, req.DailyLimit); err != nil {
		return nil, err
	}
	return &types.Response{Success: true, Message: "Daily limit saved"}, nil
}
