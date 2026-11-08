package webhooks

import (
	"context"
	"database/sql"
	"errors"

	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/services/webhook"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminResendWebhookDeliveryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Send a delivery again now, signed the same way
func NewAdminResendWebhookDeliveryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminResendWebhookDeliveryLogic {
	return &AdminResendWebhookDeliveryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminResendWebhookDeliveryLogic) AdminResendWebhookDelivery(req *types.ResendWebhookDeliveryRequest) (resp *types.ResendWebhookDeliveryResponse, err error) {
	wh, err := brandWebhook(l.ctx, l.svcCtx, req.Id)
	if err != nil {
		return nil, err
	}

	delivery, res, err := l.svcCtx.WebhookDispatcher.Resend(l.ctx, wh, req.DeliveryId)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errorx.NewNotFoundError("Delivery not found.")
	}
	if errors.Is(err, webhook.ErrDeliveryInFlight) {
		return nil, errorx.NewBadRequestError("This delivery is being sent right now. Try again in a minute.")
	}
	if err != nil {
		l.Errorf("Failed to resend webhook delivery: %v", err)
		return nil, errorx.NewInternalError("Failed to send the delivery again.")
	}

	return &types.ResendWebhookDeliveryResponse{
		Delivery: deliveryInfo(delivery, l.svcCtx.WebhookDispatcher.MaxAttempts()),
		Success:  res.Success,
		Error:    res.Error,
	}, nil
}
