package webhooks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminListWebhooksLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAdminListWebhooksLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminListWebhooksLogic {
	return &AdminListWebhooksLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminListWebhooksLogic) AdminListWebhooks() (resp *types.ListWebhooksResponse, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Admin)
	if err != nil {
		return nil, err
	}

	webhooks, err := l.svcCtx.DB.ListWebhooks(l.ctx, orgID)
	if err != nil {
		l.Errorf("Failed to list webhooks: %v", err)
		return nil, fmt.Errorf("failed to list webhooks")
	}

	result := make([]types.WebhookInfo, 0, len(webhooks))
	for _, w := range webhooks {
		// Decode events from JSON string
		var events []string
		if err := json.Unmarshal([]byte(w.Events), &events); err != nil {
			l.Errorf("Failed to decode events for webhook %s: %v", w.ID, err)
			events = []string{}
		}

		info := types.WebhookInfo{
			Id:                w.ID,
			Url:               w.Url,
			Events:            events,
			Active:            w.Active.Int64 == 1,
			CreatedAt:         utils.FormatNullString(w.CreatedAt),
			DeliveriesTotal:   int(w.DeliveriesTotal.Int64),
			DeliveriesSuccess: int(w.DeliveriesSuccess.Int64),
			DeliveriesFailed:  int(w.DeliveriesFailed.Int64),
		}
		if w.LastDeliveryAt.Valid {
			info.LastDeliveryAt = utils.FormatNullString(w.LastDeliveryAt)
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
