package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/events"
)

// Dispatcher delivers webhooks to registered endpoints when events occur.
type Dispatcher struct {
	db         *db.Queries
	httpClient *http.Client
	events     *events.Subject

	// Subscription management
	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
}

// WebhookPayload is the standard payload sent to webhook endpoints.
type WebhookPayload struct {
	Event     string                 `json:"event"`
	Timestamp string                 `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// NewDispatcher creates a new webhook dispatcher.
func NewDispatcher(db *db.Queries, eventBus *events.Subject) *Dispatcher {
	return &Dispatcher{
		db:     db,
		events: eventBus,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Start begins listening for events and dispatching webhooks.
func (d *Dispatcher) Start(ctx context.Context) error {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return nil
	}
	d.running = true
	ctx, d.cancel = context.WithCancel(ctx)
	d.mu.Unlock()

	// Subscribe to all webhook-dispatchable events
	eventTopics := []string{
		// Contact events
		events.TopicContactCreated,
		events.TopicContactUnsubscribed,

		// Email events
		events.TopicEmailSent,
		events.TopicEmailBounced,
		events.TopicEmailOpened,
		events.TopicEmailClicked,

		// Mailbox pool events
		events.TopicPoolMessageBounced,
		events.TopicPoolMessageComplained,
		events.TopicPoolMailboxPaused,
		events.TopicPoolMailboxResumed,
	}

	for _, topic := range eventTopics {
		topic := topic // capture for closure
		events.Subscribe[any](d.events, topic, func(_ context.Context, data any) error {
			d.handleEvent(ctx, topic, data)
			return nil
		})
	}

	fmt.Println("[Webhook Dispatcher] Started, listening for events")
	return nil
}

// Stop stops the dispatcher.
func (d *Dispatcher) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.cancel != nil {
		d.cancel()
	}
	d.running = false
	fmt.Println("[Webhook Dispatcher] Stopped")
}

// handleEvent processes an event and dispatches to registered webhooks.
func (d *Dispatcher) handleEvent(ctx context.Context, topic string, data interface{}) {
	// Extract org ID from event data
	orgID, err := extractOrgID(data)
	if err != nil {
		fmt.Printf("[Webhook Dispatcher] Could not extract org_id from event %s: %v\n", topic, err)
		return
	}

	// Get all active webhooks for this org that subscribe to this event
	webhooks, err := d.db.ListWebhooks(ctx, orgID)
	if err != nil {
		fmt.Printf("[Webhook Dispatcher] Failed to list webhooks for org %s: %v\n", orgID, err)
		return
	}

	// Filter webhooks that subscribe to this event
	var matchingWebhooks []db.Webhook
	for _, wh := range webhooks {
		if wh.Active.Valid && wh.Active.Int64 == 0 {
			continue
		}
		// Events is stored as comma-separated string in SQLite
		eventList := strings.Split(wh.Events, ",")
		for _, event := range eventList {
			event = strings.TrimSpace(event)
			if event == topic || event == "*" {
				matchingWebhooks = append(matchingWebhooks, wh)
				break
			}
		}
	}

	if len(matchingWebhooks) == 0 {
		return // No webhooks registered for this event
	}

	// Prepare payload
	payload := WebhookPayload{
		Event:     topic,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data:      toMap(data),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("[Webhook Dispatcher] Failed to marshal payload: %v\n", err)
		return
	}

	// Dispatch to all matching webhooks concurrently
	var wg sync.WaitGroup
	for _, wh := range matchingWebhooks {
		wg.Add(1)
		go func(webhook db.Webhook) {
			defer wg.Done()
			d.deliverWebhook(ctx, webhook, topic, payloadBytes)
		}(wh)
	}
	wg.Wait()
}

// deliverWebhook sends the webhook to the registered endpoint.
func (d *Dispatcher) deliverWebhook(ctx context.Context, webhook db.Webhook, event string, payload []byte) {
	startTime := time.Now()

	// Create HMAC signature
	mac := hmac.New(sha256.New, []byte(webhook.Secret))
	mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.Url, bytes.NewReader(payload))
	if err != nil {
		d.logDelivery(ctx, webhook.ID, event, payload, 0, "", err.Error(), 0)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Webhook-Event", event)
	req.Header.Set("X-Webhook-ID", webhook.ID)
	req.Header.Set("User-Agent", "Outlet-Webhook/1.0")

	// Send request
	resp, err := d.httpClient.Do(req)
	durationMs := int(time.Since(startTime).Milliseconds())

	var statusCode int
	var responseBody string
	var deliveryError string

	if err != nil {
		deliveryError = err.Error()
	} else {
		defer resp.Body.Close()
		statusCode = resp.StatusCode

		// Read response body (limit to 1KB)
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		responseBody = string(bodyBytes)

		if statusCode >= 400 {
			deliveryError = fmt.Sprintf("HTTP %d: %s", statusCode, responseBody)
		}
	}

	success := statusCode >= 200 && statusCode < 300

	// Log the delivery
	d.logDelivery(ctx, webhook.ID, event, payload, statusCode, responseBody, deliveryError, durationMs)

	// Update webhook stats
	d.db.UpdateWebhookDeliveryStats(ctx, db.UpdateWebhookDeliveryStatsParams{
		ID:         webhook.ID,
		Success:    success,
		LastStatus: sql.NullInt64{Int64: int64(statusCode), Valid: statusCode > 0},
	})

	if success {
		fmt.Printf("[Webhook Dispatcher] Delivered %s to %s (status: %d, duration: %dms)\n",
			event, webhook.Url, statusCode, durationMs)
	} else {
		fmt.Printf("[Webhook Dispatcher] Failed to deliver %s to %s: %s\n",
			event, webhook.Url, deliveryError)
	}
}

// logDelivery records the webhook delivery attempt.
func (d *Dispatcher) logDelivery(ctx context.Context, webhookID string, event string, payload []byte, statusCode int, response, errMsg string, durationMs int) {
	_, err := d.db.CreateWebhookLog(ctx, db.CreateWebhookLogParams{
		WebhookID:  webhookID,
		Event:      event,
		Payload:    string(payload),
		StatusCode: sql.NullInt64{Int64: int64(statusCode), Valid: statusCode > 0},
		Response:   sql.NullString{String: response, Valid: response != ""},
		Error:      sql.NullString{String: errMsg, Valid: errMsg != ""},
		DurationMs: sql.NullInt64{Int64: int64(durationMs), Valid: true},
	})
	if err != nil {
		fmt.Printf("[Webhook Dispatcher] Failed to log delivery: %v\n", err)
	}
}

// extractOrgID extracts the org_id from event data.
func extractOrgID(data interface{}) (string, error) {
	// Try to extract from struct field or map
	switch v := data.(type) {
	case map[string]interface{}:
		if orgIDStr, ok := v["org_id"].(string); ok {
			return orgIDStr, nil
		}
	default:
		// Use reflection or type assertion for known event types
		if m := toMap(data); m != nil {
			if orgIDStr, ok := m["org_id"].(string); ok {
				return orgIDStr, nil
			}
		}
	}
	return "", fmt.Errorf("org_id not found in event data")
}

// toMap converts event data to a map for the webhook payload.
func toMap(data interface{}) map[string]interface{} {
	// Marshal and unmarshal to convert struct to map
	b, err := json.Marshal(data)
	if err != nil {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil
	}
	return m
}

// DispatchEvent manually dispatches a webhook event (useful for testing or direct calls).
func (d *Dispatcher) DispatchEvent(ctx context.Context, orgID string, event string, data map[string]interface{}) error {
	// Add org_id to data if not present
	if data == nil {
		data = make(map[string]interface{})
	}
	data["org_id"] = orgID

	d.handleEvent(ctx, event, data)
	return nil
}
