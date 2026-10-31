package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/google/uuid"
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

	// Create test payload
	testPayload := map[string]interface{}{
		"event":     "test",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"data": map[string]interface{}{
			"message":    "This is a test webhook delivery",
			"webhook_id": webhook.ID,
		},
	}

	payloadBytes, err := json.Marshal(testPayload)
	if err != nil {
		l.Errorf("Failed to marshal test payload: %v", err)
		return nil, fmt.Errorf("failed to create test payload")
	}

	// Create HMAC signature
	mac := hmac.New(sha256.New, []byte(webhook.Secret))
	mac.Write(payloadBytes)
	signature := hex.EncodeToString(mac.Sum(nil))

	// Send test request
	startTime := time.Now()

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	httpReq, err := http.NewRequestWithContext(l.ctx, http.MethodPost, webhook.Url, bytes.NewReader(payloadBytes))
	if err != nil {
		l.Errorf("Failed to create HTTP request: %v", err)
		return nil, fmt.Errorf("failed to create test request")
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Webhook-Signature", signature)
	httpReq.Header.Set("X-Webhook-Event", "test")
	httpReq.Header.Set("X-Webhook-ID", webhook.ID)
	httpReq.Header.Set("User-Agent", "Outlet-Webhook/1.0")

	httpResp, err := client.Do(httpReq)
	durationMs := int(time.Since(startTime).Milliseconds())

	var statusCode int
	var responseBody string
	var testError string
	success := false

	if err != nil {
		testError = err.Error()
		l.Infof("Webhook test failed: %v", err)
	} else {
		defer httpResp.Body.Close()
		statusCode = httpResp.StatusCode

		// Read response body (limit to 1KB)
		bodyBytes, _ := io.ReadAll(io.LimitReader(httpResp.Body, 1024))
		responseBody = string(bodyBytes)

		success = statusCode >= 200 && statusCode < 300
	}

	// Log the test delivery
	_, logErr := l.svcCtx.DB.CreateWebhookLog(l.ctx, db.CreateWebhookLogParams{
		ID:         uuid.New().String(),
		WebhookID:  webhook.ID,
		Event:      "test",
		Payload:    string(payloadBytes),
		StatusCode: sql.NullInt64{Int64: int64(statusCode), Valid: statusCode > 0},
		Response:   sql.NullString{String: responseBody, Valid: responseBody != ""},
		Error:      sql.NullString{String: testError, Valid: testError != ""},
		DurationMs: sql.NullInt64{Int64: int64(durationMs), Valid: true},
	})
	if logErr != nil {
		l.Errorf("Failed to log webhook test: %v", logErr)
	}

	// Update webhook delivery stats
	successVal := 0
	if success {
		successVal = 1
	}
	statsErr := l.svcCtx.DB.UpdateWebhookDeliveryStats(l.ctx, db.UpdateWebhookDeliveryStatsParams{
		ID:         webhook.ID,
		Success:    successVal,
		LastStatus: sql.NullInt64{Int64: int64(statusCode), Valid: statusCode > 0},
	})
	if statsErr != nil {
		l.Errorf("Failed to update webhook stats: %v", statsErr)
	}

	return &types.TestWebhookResponse{
		Success:    success,
		StatusCode: statusCode,
		Response:   responseBody,
		Error:      testError,
	}, nil
}
