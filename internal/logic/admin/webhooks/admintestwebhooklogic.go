package webhooks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminTestWebhookLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAdminTestWebhookLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminTestWebhookLogic {
	return &AdminTestWebhookLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminTestWebhookLogic) AdminTestWebhook(req *types.TestWebhookRequest) (resp *types.TestWebhookResponse, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Admin)
	if err != nil {
		return nil, err
	}

	webhookID := req.Id

	// Get webhook (scoped to org)
	webhook, err := l.svcCtx.DB.GetWebhook(l.ctx, db.GetWebhookParams{
		ID:    webhookID,
		OrgID: orgID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("webhook not found")
	}
	if err != nil {
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
