package webhooks

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/middleware"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type TestWebhookLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewTestWebhookLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TestWebhookLogic {
	return &TestWebhookLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TestWebhookLogic) TestWebhook(req *types.TestWebhookRequest) (resp *types.TestWebhookResponse, err error) {
	orgID, ok := l.ctx.Value(middleware.OrgIDKey).(string)
	if !ok {
		return nil, fmt.Errorf("organization not found in context")
	}

	if req.Id == "" {
		return nil, fmt.Errorf("invalid webhook ID")
	}

	webhook, err := l.svcCtx.DB.GetWebhook(l.ctx, db.GetWebhookParams{
		ID:    req.Id,
		OrgID: orgID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("webhook not found")
		}
		l.Errorf("Failed to get webhook: %v", err)
		return nil, fmt.Errorf("failed to get webhook")
	}

	res := l.svcCtx.WebhookDispatcher.Test(l.ctx, webhook)
	return &types.TestWebhookResponse{
		Success:    res.Success,
		StatusCode: res.StatusCode,
		Response:   res.Response,
		Error:      res.Error,
	}, nil
}
