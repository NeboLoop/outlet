package webhooks

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/webhook"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type AdminCreateWebhookLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAdminCreateWebhookLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminCreateWebhookLogic {
	return &AdminCreateWebhookLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminCreateWebhookLogic) AdminCreateWebhook(req *types.RegisterWebhookRequest) (resp *types.RegisterWebhookResponse, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Admin)
	if err != nil {
		return nil, err
	}

	if req.Url == "" {
		return nil, fmt.Errorf("webhook URL is required")
	}

	events, err := webhook.EncodeEvents(req.Events)
	if err != nil {
		return nil, err
	}

	// Generate secret if not provided
	secret := req.Secret
	if secret == "" {
		secretBytes := make([]byte, 32)
		if _, err := rand.Read(secretBytes); err != nil {
			l.Errorf("Failed to generate webhook secret: %v", err)
			return nil, fmt.Errorf("failed to generate webhook secret")
		}
		secret = hex.EncodeToString(secretBytes)
	}

	activeVal := int64(1)
	if !req.Active {
		activeVal = 0
	}

	webhook, err := l.svcCtx.DB.CreateWebhook(l.ctx, db.CreateWebhookParams{
		ID:     uuid.New().String(),
		OrgID:  orgID,
		Url:    req.Url,
		Secret: secret,
		Events: events,
		Active: sql.NullInt64{Int64: activeVal, Valid: true},
	})
	if err != nil {
		l.Errorf("Failed to create webhook: %v", err)
		return nil, fmt.Errorf("failed to create webhook")
	}

	return &types.RegisterWebhookResponse{
		Success:   true,
		WebhookId: webhook.ID,
		Secret:    secret,
	}, nil
}
