package webhooks

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminUpdateWebhookLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAdminUpdateWebhookLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminUpdateWebhookLogic {
	return &AdminUpdateWebhookLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminUpdateWebhookLogic) AdminUpdateWebhook(req *types.UpdateWebhookRequest) (resp *types.WebhookInfo, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Admin)
	if err != nil {
		return nil, err
	}

	webhookID := req.Id

	// Get existing webhook (scoped to org)
	existing, err := l.svcCtx.DB.GetWebhook(l.ctx, db.GetWebhookParams{
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

	// Prepare update params
	url := existing.Url
	if req.Url != "" {
		url = req.Url
	}

	eventsStr := existing.Events
	if len(req.Events) > 0 {
		eventsJSON, err := json.Marshal(req.Events)
		if err != nil {
			l.Errorf("Failed to encode events: %v", err)
			return nil, fmt.Errorf("failed to encode events")
		}
		eventsStr = string(eventsJSON)
	}

	activeVal := int64(0)
	if req.Active {
		activeVal = 1
	}

	w, err := l.svcCtx.DB.UpdateWebhook(l.ctx, db.UpdateWebhookParams{
		ID:     webhookID,
		OrgID:  orgID,
		Url:    url,
		Events: eventsStr,
		Active: sql.NullInt64{Int64: activeVal, Valid: true},
	})
	if err != nil {
		l.Errorf("Failed to update webhook: %v", err)
		return nil, fmt.Errorf("failed to update webhook")
	}

	// Decode events from JSON string for response
	var events []string
	if err := json.Unmarshal([]byte(w.Events), &events); err != nil {
		l.Errorf("Failed to decode events: %v", err)
		events = []string{}
	}

	info := &types.WebhookInfo{
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

	return info, nil
}
