package pool

import (
	"context"
	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/services/mailpool"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreatePoolMailboxLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreatePoolMailboxLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreatePoolMailboxLogic {
	return &CreatePoolMailboxLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreatePoolMailboxLogic) CreatePoolMailbox(req *types.CreatePoolMailboxRequest) (resp *types.PoolMailboxInfo, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return nil, err
	}
	mb, err := l.svcCtx.MailPool.CreateMailbox(l.ctx, mailpool.MailboxInput{
		DomainID:      req.DomainId,
		Address:       req.Address,
		SMTPHost:      req.SmtpHost,
		SMTPPort:      req.SmtpPort,
		SMTPUsername:  req.SmtpUsername,
		SMTPPassword:  req.SmtpPassword,
		WarmStartedAt: req.WarmStartedAt,
	})
	if err != nil {
		return nil, err
	}
	info := mailboxInfo(mb)
	return &info, nil
}
