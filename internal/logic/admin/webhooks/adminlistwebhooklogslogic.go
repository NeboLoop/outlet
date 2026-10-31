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
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminListWebhookLogsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAdminListWebhookLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminListWebhookLogsLogic {
	return &AdminListWebhookLogsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminListWebhookLogsLogic) AdminListWebhookLogs(req *types.ListWebhookLogsRequest) (resp *types.ListWebhookLogsResponse, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Admin)
	if err != nil {
		return nil, err
	}

	webhookID := req.Id

	// Verify webhook belongs to this org
	_, err = l.svcCtx.DB.GetWebhook(l.ctx, db.GetWebhookParams{
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

	// Set default limit
	limit := int64(50)
	if req.Limit > 0 && req.Limit <= 100 {
		limit = int64(req.Limit)
	}

	// List webhook logs
	logs, err := l.svcCtx.DB.ListWebhookLogs(l.ctx, db.ListWebhookLogsParams{
		WebhookID:  webhookID,
		LimitCount: limit,
	})
	if err != nil {
		l.Errorf("Failed to list webhook logs: %v", err)
		return nil, fmt.Errorf("failed to list webhook logs")
	}

	// Convert to response type
	result := make([]types.WebhookLogInfo, 0, len(logs))
	for _, log := range logs {
		info := types.WebhookLogInfo{
			Id:          log.ID,
			Event:       log.Event,
			Payload:     log.Payload,
			StatusCode:  int(log.StatusCode.Int64),
			Duration:    int(log.DurationMs.Int64),
			DeliveredAt: utils.FormatNullString(log.DeliveredAt),
		}
		if log.Response.Valid {
			info.Response = log.Response.String
		}
		if log.Error.Valid {
			info.Error = log.Error.String
		}
		result = append(result, info)
	}

	return &types.ListWebhookLogsResponse{
		Logs: result,
	}, nil
}
