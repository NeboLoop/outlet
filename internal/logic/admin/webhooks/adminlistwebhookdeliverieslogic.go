package webhooks

import (
	"context"
	"database/sql"
	"errors"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/services/webhook"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminListWebhookDeliveriesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// A webhook's deliveries, newest first: status, tries and next retry
func NewAdminListWebhookDeliveriesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminListWebhookDeliveriesLogic {
	return &AdminListWebhookDeliveriesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminListWebhookDeliveriesLogic) AdminListWebhookDeliveries(req *types.ListWebhookDeliveriesRequest) (resp *types.ListWebhookDeliveriesResponse, err error) {
	webhook, err := brandWebhook(l.ctx, l.svcCtx, req.Id)
	if err != nil {
		return nil, err
	}

	limit := int64(50)
	if req.Limit > 0 && req.Limit <= 200 {
		limit = int64(req.Limit)
	}
	deliveries, err := l.svcCtx.DB.ListWebhookDeliveries(l.ctx, db.ListWebhookDeliveriesParams{WebhookID: webhook.ID, LimitCount: limit})
	if err != nil {
		l.Errorf("Failed to list webhook deliveries: %v", err)
		return nil, errorx.NewInternalError("Failed to list webhook deliveries.")
	}

	out := make([]types.WebhookDeliveryInfo, 0, len(deliveries))
	for _, d := range deliveries {
		out = append(out, deliveryInfo(d, l.svcCtx.WebhookDispatcher.MaxAttempts()))
	}
	return &types.ListWebhookDeliveriesResponse{Deliveries: out}, nil
}

// brandWebhook returns a webhook of the brand the request names, for its
// admins.
func brandWebhook(ctx context.Context, svcCtx *svc.ServiceContext, id string) (db.Webhook, error) {
	orgID, err := access.RequireBrand(ctx, svcCtx.DB, access.Admin)
	if err != nil {
		return db.Webhook{}, err
	}
	webhook, err := svcCtx.DB.GetWebhook(ctx, db.GetWebhookParams{ID: id, OrgID: orgID})
	if errors.Is(err, sql.ErrNoRows) {
		return db.Webhook{}, errorx.NewNotFoundError("Webhook not found.")
	}
	return webhook, err
}

// deliveryInfo is a delivery as the admin app shows it.
func deliveryInfo(d db.WebhookDelivery, maxAttempts int) types.WebhookDeliveryInfo {
	info := types.WebhookDeliveryInfo{
		Id:             d.ID,
		Event:          d.Event,
		Status:         d.Status,
		Attempts:       int(d.Attempts),
		MaxAttempts:    maxAttempts,
		LastStatusCode: int(d.LastStatusCode.Int64),
		LastError:      d.LastError.String,
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
	}
	if d.Status == webhook.StatusPending {
		info.NextRetryAt = d.NextAttemptAt.String
	}
	return info
}
