package webhooks

import (
	"context"
	"fmt"

	"github.com/outlet-sh/outlet/internal/middleware"
	webhooksvc "github.com/outlet-sh/outlet/internal/services/webhook"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListWebhooksLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListWebhooksLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListWebhooksLogic {
	return &ListWebhooksLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListWebhooksLogic) ListWebhooks() (resp *types.ListWebhooksResponse, err error) {
	orgID, ok := l.ctx.Value(middleware.OrgIDKey).(string)
	if !ok {
		return nil, fmt.Errorf("organization not found in context")
	}

	webhooks, err := l.svcCtx.DB.ListWebhooks(l.ctx, orgID)
	if err != nil {
		l.Errorf("Failed to list webhooks: %v", err)
		return nil, fmt.Errorf("failed to list webhooks")
	}

	result := make([]types.WebhookInfo, 0, len(webhooks))
	for _, w := range webhooks {
		info := types.WebhookInfo{
			Id:                w.ID,
			Url:               w.Url,
			Events:            webhooksvc.DecodeEvents(w.Events),
			Active:            w.Active.Valid && w.Active.Int64 == 1,
			CreatedAt:         w.CreatedAt.String,
			DeliveriesTotal:   int(w.DeliveriesTotal.Int64),
			DeliveriesSuccess: int(w.DeliveriesSuccess.Int64),
			DeliveriesFailed:  int(w.DeliveriesFailed.Int64),
		}
		if w.LastDeliveryAt.Valid {
			info.LastDeliveryAt = w.LastDeliveryAt.String
		}
		if w.LastStatus.Valid {
			info.LastStatus = int(w.LastStatus.Int64)
		}
		result = append(result, info)
	}

	return &types.ListWebhooksResponse{
		Webhooks: result,
	}, nil
}
