package tools

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/mcp/mcpctx"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// webhookActions defines valid actions for webhooks.
var webhookActions = []string{"create", "list", "get", "update", "delete", "test", "logs"}

// WebhookInput defines input for the webhook tool.
type WebhookInput struct {
	Action string `json:"action" jsonschema:"required,Action to perform: create, list, get, update, delete, test, logs"`

	// Common
	ID string `json:"id,omitempty" jsonschema:"Webhook ID (for get, update, delete, test, logs)"`

	// Create/Update fields
	URL    string `json:"url,omitempty" jsonschema:"Webhook endpoint URL (required for create)"`
	Events string `json:"events,omitempty" jsonschema:"Comma-separated list of events to subscribe to (e.g., 'contact.created,email.sent')"`
	Active *bool  `json:"active,omitempty" jsonschema:"Whether the webhook is active (default: true)"`

	// Logs pagination
	Limit int64 `json:"limit,omitempty" jsonschema:"Number of log entries to return (default: 20, max: 100)"`
}

// WebhookItem represents a webhook in list output.
type WebhookItem struct {
	ID                string `json:"id"`
	URL               string `json:"url"`
	Events            string `json:"events"`
	Active            bool   `json:"active"`
	DeliveriesTotal   int64  `json:"deliveries_total"`
	DeliveriesSuccess int64  `json:"deliveries_success"`
	DeliveriesFailed  int64  `json:"deliveries_failed"`
	LastDeliveryAt    string `json:"last_delivery_at,omitempty"`
	LastStatus        int64  `json:"last_status,omitempty"`
	CreatedAt         string `json:"created_at"`
}

// WebhookListOutput defines output for webhook list.
type WebhookListOutput struct {
	Webhooks []WebhookItem `json:"webhooks"`
	Total    int           `json:"total"`
}

// WebhookCreateOutput defines output for webhook create.
type WebhookCreateOutput struct {
	ID      string `json:"id"`
	URL     string `json:"url"`
	Events  string `json:"events"`
	Secret  string `json:"secret"`
	Active  bool   `json:"active"`
	Created bool   `json:"created"`
}

// WebhookGetOutput defines output for webhook get.
type WebhookGetOutput struct {
	ID                string `json:"id"`
	URL               string `json:"url"`
	Events            string `json:"events"`
	Secret            string `json:"secret"`
	Active            bool   `json:"active"`
	DeliveriesTotal   int64  `json:"deliveries_total"`
	DeliveriesSuccess int64  `json:"deliveries_success"`
	DeliveriesFailed  int64  `json:"deliveries_failed"`
	LastDeliveryAt    string `json:"last_delivery_at,omitempty"`
	LastStatus        int64  `json:"last_status,omitempty"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

// WebhookUpdateOutput defines output for webhook update.
type WebhookUpdateOutput struct {
	ID      string `json:"id"`
	URL     string `json:"url"`
	Events  string `json:"events"`
	Active  bool   `json:"active"`
	Updated bool   `json:"updated"`
}

// WebhookTestOutput defines output for webhook test.
type WebhookTestOutput struct {
	Success    bool   `json:"success"`
	StatusCode int    `json:"status_code"`
	Response   string `json:"response,omitempty"`
	Error      string `json:"error,omitempty"`
	DurationMs int    `json:"duration_ms"`
}

// WebhookLogItem represents a webhook log entry.
type WebhookLogItem struct {
	ID          string `json:"id"`
	Event       string `json:"event"`
	StatusCode  int64  `json:"status_code,omitempty"`
	Response    string `json:"response,omitempty"`
	Error       string `json:"error,omitempty"`
	DurationMs  int64  `json:"duration_ms,omitempty"`
	DeliveredAt string `json:"delivered_at"`
}

// WebhookLogsOutput defines output for webhook logs.
type WebhookLogsOutput struct {
	WebhookID string           `json:"webhook_id"`
	Logs      []WebhookLogItem `json:"logs"`
	Total     int              `json:"total"`
}

// RegisterWebhookTool registers the webhook tool.
func RegisterWebhookTool(server *mcp.Server, toolCtx *mcpctx.ToolContext) {
	mcp.AddTool(server, &mcp.Tool{
		Name:  "webhook",
		Title: "Webhook Management",
		Description: `Manage outbound webhooks for event notifications.

PREREQUISITE: You must first select a brand using brand(resource: brand, action: select).

Actions and Required Fields:
- create: Create a new webhook (requires: url, events)
- list: List all webhooks
- get: Get webhook details including secret (requires: id)
- update: Update a webhook (requires: id)
- delete: Delete a webhook (requires: id)
- test: Send a test event to the webhook (requires: id)
- logs: Get recent delivery logs (requires: id, optional: limit)

Available Events:
- contact.created, contact.updated, contact.deleted
- contact.subscribed, contact.unsubscribed
- email.sent, email.opened, email.clicked
- email.bounced, email.complained
- campaign.sent, campaign.completed

Examples:
  webhook(action: create, url: "https://example.com/webhook", events: "contact.created,email.sent")
  webhook(action: list)
  webhook(action: get, id: "uuid")
  webhook(action: update, id: "uuid", active: false)
  webhook(action: delete, id: "uuid")
  webhook(action: test, id: "uuid")
  webhook(action: logs, id: "uuid", limit: 50)`,
	}, webhookHandler(toolCtx))
}

func webhookHandler(toolCtx *mcpctx.ToolContext) func(ctx context.Context, req *mcp.CallToolRequest, input WebhookInput) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input WebhookInput) (*mcp.CallToolResult, any, error) {
		// Validate action
		if !slices.Contains(webhookActions, input.Action) {
			return nil, nil, mcpctx.NewValidationError(
				fmt.Sprintf("invalid action '%s', must be: %s", input.Action, strings.Join(webhookActions, ", ")),
				"action")
		}

		switch input.Action {
		case "create":
			return handleWebhookCreate(ctx, toolCtx, input)
		case "list":
			return handleWebhookList(ctx, toolCtx, input)
		case "get":
			return handleWebhookGet(ctx, toolCtx, input)
		case "update":
			return handleWebhookUpdate(ctx, toolCtx, input)
		case "delete":
			return handleWebhookDelete(ctx, toolCtx, input)
		case "test":
			return handleWebhookTest(ctx, toolCtx, input)
		case "logs":
			return handleWebhookLogs(ctx, toolCtx, input)
		}
		return nil, nil, nil
	}
}

func handleWebhookCreate(ctx context.Context, toolCtx *mcpctx.ToolContext, input WebhookInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.URL) == "" {
		return nil, nil, mcpctx.NewValidationError("url is required", "url")
	}
	if strings.TrimSpace(input.Events) == "" {
		return nil, nil, mcpctx.NewValidationError("events is required (comma-separated list)", "events")
	}

	// Generate a secure secret
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, nil, fmt.Errorf("failed to generate secret: %w", err)
	}
	secret := hex.EncodeToString(secretBytes)

	active := true
	if input.Active != nil {
		active = *input.Active
	}

	webhookID := uuid.New().String()
	webhook, err := toolCtx.DB().CreateWebhook(ctx, db.CreateWebhookParams{
		ID:     webhookID,
		OrgID:  toolCtx.BrandID(ctx),
		Url:    input.URL,
		Secret: secret,
		Events: input.Events,
		Active: sql.NullInt64{Int64: boolToInt64(active), Valid: true},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create webhook: %w", err)
	}

	return nil, WebhookCreateOutput{
		ID:      webhook.ID,
		URL:     webhook.Url,
		Events:  webhook.Events,
		Secret:  webhook.Secret,
		Active:  int64ToBool(webhook.Active),
		Created: true,
	}, nil
}

func handleWebhookList(ctx context.Context, toolCtx *mcpctx.ToolContext, input WebhookInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	webhooks, err := toolCtx.DB().ListWebhooks(ctx, toolCtx.BrandID(ctx))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list webhooks: %w", err)
	}

	items := make([]WebhookItem, 0, len(webhooks))
	for _, w := range webhooks {
		items = append(items, WebhookItem{
			ID:                w.ID,
			URL:               w.Url,
			Events:            w.Events,
			Active:            int64ToBool(w.Active),
			DeliveriesTotal:   w.DeliveriesTotal.Int64,
			DeliveriesSuccess: w.DeliveriesSuccess.Int64,
			DeliveriesFailed:  w.DeliveriesFailed.Int64,
			LastDeliveryAt:    w.LastDeliveryAt.String,
			LastStatus:        w.LastStatus.Int64,
			CreatedAt:         w.CreatedAt.String,
		})
	}

	return nil, WebhookListOutput{
		Webhooks: items,
		Total:    len(items),
	}, nil
}

func handleWebhookGet(ctx context.Context, toolCtx *mcpctx.ToolContext, input WebhookInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	webhook, err := toolCtx.DB().GetWebhook(ctx, db.GetWebhookParams{
		ID:    input.ID,
		OrgID: toolCtx.BrandID(ctx),
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("webhook %s not found", input.ID))
	}

	return nil, WebhookGetOutput{
		ID:                webhook.ID,
		URL:               webhook.Url,
		Events:            webhook.Events,
		Secret:            webhook.Secret,
		Active:            int64ToBool(webhook.Active),
		DeliveriesTotal:   webhook.DeliveriesTotal.Int64,
		DeliveriesSuccess: webhook.DeliveriesSuccess.Int64,
		DeliveriesFailed:  webhook.DeliveriesFailed.Int64,
		LastDeliveryAt:    webhook.LastDeliveryAt.String,
		LastStatus:        webhook.LastStatus.Int64,
		CreatedAt:         webhook.CreatedAt.String,
		UpdatedAt:         webhook.UpdatedAt.String,
	}, nil
}

func handleWebhookUpdate(ctx context.Context, toolCtx *mcpctx.ToolContext, input WebhookInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	// Get existing webhook to merge updates
	existing, err := toolCtx.DB().GetWebhook(ctx, db.GetWebhookParams{
		ID:    input.ID,
		OrgID: toolCtx.BrandID(ctx),
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("webhook %s not found", input.ID))
	}

	// Use existing values as defaults
	active := int64ToBool(existing.Active)
	if input.Active != nil {
		active = *input.Active
	}

	webhook, err := toolCtx.DB().UpdateWebhook(ctx, db.UpdateWebhookParams{
		ID:     input.ID,
		OrgID:  toolCtx.BrandID(ctx),
		Url:    input.URL,
		Events: input.Events,
		Active: sql.NullInt64{Int64: boolToInt64(active), Valid: true},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update webhook: %w", err)
	}

	return nil, WebhookUpdateOutput{
		ID:      webhook.ID,
		URL:     webhook.Url,
		Events:  webhook.Events,
		Active:  int64ToBool(webhook.Active),
		Updated: true,
	}, nil
}

func handleWebhookDelete(ctx context.Context, toolCtx *mcpctx.ToolContext, input WebhookInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	// Verify webhook exists
	_, err := toolCtx.DB().GetWebhook(ctx, db.GetWebhookParams{
		ID:    input.ID,
		OrgID: toolCtx.BrandID(ctx),
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("webhook %s not found", input.ID))
	}

	err = toolCtx.DB().DeleteWebhook(ctx, db.DeleteWebhookParams{
		ID:    input.ID,
		OrgID: toolCtx.BrandID(ctx),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to delete webhook: %w", err)
	}

	return nil, DeleteOutput{
		Success: true,
		Message: fmt.Sprintf("Webhook %s deleted successfully", input.ID),
	}, nil
}

func handleWebhookTest(ctx context.Context, toolCtx *mcpctx.ToolContext, input WebhookInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	webhook, err := toolCtx.DB().GetWebhook(ctx, db.GetWebhookParams{
		ID:    input.ID,
		OrgID: toolCtx.BrandID(ctx),
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("webhook %s not found", input.ID))
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
		return nil, nil, fmt.Errorf("failed to create test payload: %w", err)
	}

	// Sign the payload
	mac := hmac.New(sha256.New, []byte(webhook.Secret))
	mac.Write(payloadBytes)
	signature := hex.EncodeToString(mac.Sum(nil))

	startTime := time.Now()

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.Url, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create test request: %w", err)
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
	} else {
		defer httpResp.Body.Close()
		statusCode = httpResp.StatusCode

		bodyBytes, _ := io.ReadAll(io.LimitReader(httpResp.Body, 1024))
		responseBody = string(bodyBytes)

		success = statusCode >= 200 && statusCode < 300
	}

	// Log the test delivery
	toolCtx.DB().CreateWebhookLog(ctx, db.CreateWebhookLogParams{
		ID:         uuid.New().String(),
		WebhookID:  webhook.ID,
		Event:      "test",
		Payload:    string(payloadBytes),
		StatusCode: sql.NullInt64{Int64: int64(statusCode), Valid: statusCode > 0},
		Response:   sql.NullString{String: responseBody, Valid: responseBody != ""},
		Error:      sql.NullString{String: testError, Valid: testError != ""},
		DurationMs: sql.NullInt64{Int64: int64(durationMs), Valid: true},
	})

	// Update delivery stats
	toolCtx.DB().UpdateWebhookDeliveryStats(ctx, db.UpdateWebhookDeliveryStatsParams{
		ID:         webhook.ID,
		Success:    success,
		LastStatus: sql.NullInt64{Int64: int64(statusCode), Valid: statusCode > 0},
	})

	return nil, WebhookTestOutput{
		Success:    success,
		StatusCode: statusCode,
		Response:   responseBody,
		Error:      testError,
		DurationMs: durationMs,
	}, nil
}

func handleWebhookLogs(ctx context.Context, toolCtx *mcpctx.ToolContext, input WebhookInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	// Verify webhook exists
	_, err := toolCtx.DB().GetWebhook(ctx, db.GetWebhookParams{
		ID:    input.ID,
		OrgID: toolCtx.BrandID(ctx),
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("webhook %s not found", input.ID))
	}

	limit := input.Limit
	if limit < 1 || limit > 100 {
		limit = 20
	}

	logs, err := toolCtx.DB().ListWebhookLogs(ctx, db.ListWebhookLogsParams{
		WebhookID:  input.ID,
		LimitCount: limit,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list webhook logs: %w", err)
	}

	items := make([]WebhookLogItem, 0, len(logs))
	for _, l := range logs {
		items = append(items, WebhookLogItem{
			ID:          l.ID,
			Event:       l.Event,
			StatusCode:  l.StatusCode.Int64,
			Response:    l.Response.String,
			Error:       l.Error.String,
			DurationMs:  l.DurationMs.Int64,
			DeliveredAt: l.DeliveredAt.String,
		})
	}

	return nil, WebhookLogsOutput{
		WebhookID: input.ID,
		Logs:      items,
		Total:     len(items),
	}, nil
}

// registerWebhookToolToRegistry registers webhook tool to the direct-call registry.
func registerWebhookToolToRegistry(registry *ToolRegistry, toolCtx *mcpctx.ToolContext) {
	registry.Register("webhook", func(ctx context.Context, args json.RawMessage) (interface{}, error) {
		var input WebhookInput
		if err := json.Unmarshal(args, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		handler := webhookHandler(toolCtx)
		_, output, err := handler(ctx, nil, input)
		return output, err
	})
}
