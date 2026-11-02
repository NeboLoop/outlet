package pool

import (
	"context"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/middleware"
	"github.com/outlet-sh/outlet/internal/services/mailpool"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PoolSendLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Send one email from the tenant's sticky pool mailbox
func NewPoolSendLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PoolSendLogic {
	return &PoolSendLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PoolSendLogic) PoolSend(req *types.PoolSendRequest) (resp *types.PoolSendResponse, err error) {
	orgID, ok := l.ctx.Value(middleware.OrgIDKey).(string)
	if !ok || orgID == "" {
		return nil, errorx.NewUnauthorizedError("API key required")
	}
	res, err := l.svcCtx.MailPool.Send(l.ctx, mailpool.SendInput{
		OrgID:                 orgID,
		Tenant:                req.Tenant,
		MailboxID:             req.MailboxId,
		To:                    req.To,
		FromName:              req.FromName,
		Subject:               req.Subject,
		Text:                  req.Text,
		HTML:                  req.Html,
		InReplyTo:             req.InReplyTo,
		References:            req.References,
		Headers:               req.Headers,
		ListUnsubscribeURL:    req.ListUnsubscribeUrl,
		ListUnsubscribeMailto: req.ListUnsubscribeMailto,
		IdempotencyKey:        req.IdempotencyKey,
	})
	if err != nil {
		return nil, err
	}
	return &types.PoolSendResponse{
		MessageId:      res.MessageID,
		MailboxId:      res.MailboxID,
		MailboxAddress: res.MailboxAddress,
		Status:         res.Status,
		FailureKind:    res.FailureKind,
		SmtpCode:       res.SMTPCode,
		SmtpResponse:   res.SMTPResponse,
	}, nil
}
