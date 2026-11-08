package webhooks

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/middleware"
	webhooksvc "github.com/outlet-sh/outlet/internal/services/webhook"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetWebhookLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetWebhookLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetWebhookLogic {
	return &GetWebhookLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetWebhookLogic) GetWebhook(req *types.GetWebhookRequest) (resp *types.WebhookInfo, err error) {
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

	info := &types.WebhookInfo{
		Id:                webhook.ID,
		Url:               webhook.Url,
		Events:            webhooksvc.DecodeEvents(webhook.Events),
		Active:            webhook.Active.Valid && webhook.Active.Int64 == 1,
		CreatedAt:         webhook.CreatedAt.String,
		DeliveriesTotal:   int(webhook.DeliveriesTotal.Int64),
		DeliveriesSuccess: int(webhook.DeliveriesSuccess.Int64),
		DeliveriesFailed:  int(webhook.DeliveriesFailed.Int64),
	}
	if webhook.LastDeliveryAt.Valid {
		info.LastDeliveryAt = webhook.LastDeliveryAt.String
	}
	if webhook.LastStatus.Valid {
		info.LastStatus = int(webhook.LastStatus.Int64)
	}

	return info, nil
}
