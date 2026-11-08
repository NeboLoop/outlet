package webhook

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
	"strconv"
	"sync"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/events"

	"github.com/google/uuid"
)

// Dispatcher delivers webhooks to registered endpoints when events occur.
// Each event for each webhook is a delivery: tried at once and, when the
// endpoint fails, tried again on the RetryConfig's schedule until it
// succeeds or runs out of tries.
type Dispatcher struct {
	db         *db.Queries
	httpClient *http.Client
	events     *events.Subject
	retry      RetryConfig
	now        func() time.Time

	// Subscription management
	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
	loop    sync.WaitGroup
}

// RetryConfig says how failed deliveries are tried again.
type RetryConfig struct {
	// Backoff is the wait before each retry. A delivery gets one try at
	// once and one more per entry.
	Backoff []time.Duration
	// PollInterval is how often the dispatcher looks for retries that are due.
	PollInterval time.Duration
	// BatchSize is how many due deliveries one look takes.
	BatchSize int
	// MaxBacklog is how many deliveries of one webhook may wait for a retry.
	// A delivery that fails while its webhook's backlog is full is not
	// retried, so an endpoint that is down for good can't pile up work.
	MaxBacklog int
	// Keep is how long finished deliveries, and their tries, are kept.
	Keep time.Duration
}

// DefaultRetryConfig gives a delivery 5 tries over about a day.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		Backoff:      []time.Duration{5 * time.Minute, time.Hour, 6 * time.Hour, 18 * time.Hour},
		PollInterval: 30 * time.Second,
		BatchSize:    100,
		MaxBacklog:   1000,
		Keep:         30 * 24 * time.Hour,
	}
}

// MaxAttempts is how many tries a delivery gets.
func (c RetryConfig) MaxAttempts() int {
	return len(c.Backoff) + 1
}

// MaxAttempts is how many tries each delivery gets.
func (d *Dispatcher) MaxAttempts() int {
	return d.retry.MaxAttempts()
}

// Delivery statuses (webhook_deliveries.status).
const (
	StatusPending   = "pending"
	StatusSending   = "sending"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
)

// ErrDeliveryInFlight is returned by Resend while a try is under way.
var ErrDeliveryInFlight = errors.New("this delivery is being sent right now")

// dbTime is how the database stores times: UTC, like datetime('now').
const dbTime = "2006-01-02 15:04:05"

// WebhookPayload is the standard payload sent to webhook endpoints.
type WebhookPayload struct {
	Event     string                 `json:"event"`
	Timestamp string                 `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// Result is the outcome of one POST to a webhook endpoint.
type Result struct {
	Success    bool
	StatusCode int
	Response   string
	Error      string
	DurationMs int
}

// NewDispatcher creates a new webhook dispatcher.
func NewDispatcher(db *db.Queries, eventBus *events.Subject, retry RetryConfig) *Dispatcher {
	return &Dispatcher{
		db:     db,
		events: eventBus,
		retry:  retry,
		now:    time.Now,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Start begins listening for events, dispatching webhooks and retrying the
// ones that failed.
func (d *Dispatcher) Start(ctx context.Context) error {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return nil
	}
	d.running = true
	ctx, d.cancel = context.WithCancel(ctx)
	d.mu.Unlock()

	// Subscribe to every event a webhook can subscribe to
	eventTopics := Events

	for _, topic := range eventTopics {
		topic := topic // capture for closure
		events.Subscribe[any](d.events, topic, func(_ context.Context, data any) error {
			d.handleEvent(ctx, topic, data)
			return nil
		})
	}

	d.loop.Add(1)
	go d.retryLoop(ctx)

	fmt.Println("[Webhook Dispatcher] Started, listening for events")
	return nil
}

// Stop stops the dispatcher.
func (d *Dispatcher) Stop() {
	d.mu.Lock()
	if d.cancel != nil {
		d.cancel()
	}
	d.running = false
	d.mu.Unlock()
	d.loop.Wait()
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
		for _, event := range DecodeEvents(wh.Events) {
			if event == topic || event == "*" {
				matchingWebhooks = append(matchingWebhooks, wh)
				break
			}
		}
	}

	if len(matchingWebhooks) == 0 {
		return // No webhooks registered for this event
	}

	// Prepare payload. Its bytes are stored with each delivery, so every
	// try sends, and signs, exactly these bytes.
	payload := WebhookPayload{
		Event:     topic,
		Timestamp: d.now().UTC().Format(time.RFC3339),
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
			now := d.now()
			delivery, err := d.db.CreateWebhookDelivery(ctx, db.CreateWebhookDeliveryParams{
				ID:         uuid.NewString(),
				WebhookID:  webhook.ID,
				Event:      topic,
				Payload:    string(payloadBytes),
				ClaimUntil: sql.NullString{String: d.claimUntil(now), Valid: true},
				Now:        now.UTC().Format(dbTime),
			})
			if err != nil {
				fmt.Printf("[Webhook Dispatcher] Failed to record delivery of %s to %s: %v\n", topic, webhook.Url, err)
				return
			}
			d.attempt(ctx, webhook, delivery)
		}(wh)
	}
	wg.Wait()
}

// claimUntil is when a claim taken at now lapses: well after any try
// started then has finished.
func (d *Dispatcher) claimUntil(now time.Time) string {
	return now.Add(d.httpClient.Timeout + time.Minute).UTC().Format(dbTime)
}

// attempt makes one try of a claimed delivery, records it, and schedules
// the next try or finishes the delivery.
func (d *Dispatcher) attempt(ctx context.Context, webhook db.Webhook, delivery db.WebhookDelivery) Result {
	attempt := int(delivery.Attempts) + 1
	res := d.post(ctx, webhook, delivery.Event, delivery.ID, attempt, []byte(delivery.Payload))

	_, err := d.db.CreateWebhookLog(ctx, db.CreateWebhookLogParams{
		ID:         uuid.NewString(),
		WebhookID:  webhook.ID,
		DeliveryID: sql.NullString{String: delivery.ID, Valid: true},
		Attempt:    sql.NullInt64{Int64: int64(attempt), Valid: true},
		Event:      delivery.Event,
		Payload:    delivery.Payload,
		StatusCode: sql.NullInt64{Int64: int64(res.StatusCode), Valid: res.StatusCode > 0},
		Response:   sql.NullString{String: res.Response, Valid: res.Response != ""},
		Error:      sql.NullString{String: res.Error, Valid: res.Error != ""},
		DurationMs: sql.NullInt64{Int64: int64(res.DurationMs), Valid: true},
	})
	if err != nil {
		fmt.Printf("[Webhook Dispatcher] Failed to log delivery: %v\n", err)
	}
	d.db.UpdateWebhookDeliveryStats(ctx, db.UpdateWebhookDeliveryStatsParams{
		ID:         webhook.ID,
		Success:    res.Success,
		LastStatus: sql.NullInt64{Int64: int64(res.StatusCode), Valid: res.StatusCode > 0},
	})

	now := d.now()
	finish := db.FinishWebhookDeliveryAttemptParams{
		ID:             delivery.ID,
		Status:         StatusSucceeded,
		LastStatusCode: sql.NullInt64{Int64: int64(res.StatusCode), Valid: res.StatusCode > 0},
		LastError:      sql.NullString{String: res.Error, Valid: res.Error != ""},
		Now:            now.UTC().Format(dbTime),
	}
	if !res.Success {
		finish.Status = StatusFailed
		if attempt < d.retry.MaxAttempts() {
			backlog, err := d.db.CountWebhookBacklog(ctx, webhook.ID)
			if err == nil && backlog <= int64(d.retry.MaxBacklog) {
				finish.Status = StatusPending
				finish.NextAttemptAt = sql.NullString{String: now.Add(d.retry.Backoff[attempt-1]).UTC().Format(dbTime), Valid: true}
			} else if err == nil {
				finish.LastError.String = fmt.Sprintf("%s (not retried: %d deliveries are already waiting for this endpoint)", res.Error, backlog-1)
				finish.LastError.Valid = true
			}
		}
	}
	if err := d.db.FinishWebhookDeliveryAttempt(ctx, finish); err != nil {
		fmt.Printf("[Webhook Dispatcher] Failed to record try of delivery %s: %v\n", delivery.ID, err)
	}

	if res.Success {
		fmt.Printf("[Webhook Dispatcher] Delivered %s to %s (try %d, status: %d, duration: %dms)\n",
			delivery.Event, webhook.Url, attempt, res.StatusCode, res.DurationMs)
	} else {
		fmt.Printf("[Webhook Dispatcher] Failed to deliver %s to %s (try %d of %d): %s\n",
			delivery.Event, webhook.Url, attempt, d.retry.MaxAttempts(), res.Error)
	}
	return res
}

// post sends one signed payload to a webhook's endpoint. The signature is
// the HMAC-SHA256 of the payload bytes with the webhook's secret, the same
// on every try.
func (d *Dispatcher) post(ctx context.Context, webhook db.Webhook, event, deliveryID string, attempt int, payload []byte) Result {
	startTime := time.Now()

	mac := hmac.New(sha256.New, []byte(webhook.Secret))
	mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.Url, bytes.NewReader(payload))
	if err != nil {
		return Result{Error: err.Error()}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Webhook-Event", event)
	req.Header.Set("X-Webhook-ID", webhook.ID)
	if deliveryID != "" {
		// The same on every try of a delivery, so a receiver can tell a
		// retry from a new event
		req.Header.Set("X-Webhook-Delivery", deliveryID)
		req.Header.Set("X-Webhook-Attempt", strconv.Itoa(attempt))
	}
	req.Header.Set("User-Agent", "Outlet-Webhook/1.0")

	resp, err := d.httpClient.Do(req)
	res := Result{DurationMs: int(time.Since(startTime).Milliseconds())}
	if err != nil {
		res.Error = err.Error()
		return res
	}
	defer resp.Body.Close()
	res.StatusCode = resp.StatusCode

	// Read response body (limit to 1KB)
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	res.Response = string(bodyBytes)

	res.Success = res.StatusCode >= 200 && res.StatusCode < 300
	if !res.Success {
		res.Error = fmt.Sprintf("HTTP %d: %s", res.StatusCode, res.Response)
	}
	return res
}

// Test sends a test event to a webhook once, signed like every delivery,
// and logs it. Test sends are not retried.
func (d *Dispatcher) Test(ctx context.Context, webhook db.Webhook) Result {
	payloadBytes, _ := json.Marshal(WebhookPayload{
		Event:     "test",
		Timestamp: d.now().UTC().Format(time.RFC3339),
		Data: map[string]interface{}{
			"message":    "This is a test webhook delivery",
			"webhook_id": webhook.ID,
		},
	})
	res := d.post(ctx, webhook, "test", "", 0, payloadBytes)

	_, err := d.db.CreateWebhookLog(ctx, db.CreateWebhookLogParams{
		ID:         uuid.NewString(),
		WebhookID:  webhook.ID,
		Event:      "test",
		Payload:    string(payloadBytes),
		StatusCode: sql.NullInt64{Int64: int64(res.StatusCode), Valid: res.StatusCode > 0},
		Response:   sql.NullString{String: res.Response, Valid: res.Response != ""},
		Error:      sql.NullString{String: res.Error, Valid: res.Error != ""},
		DurationMs: sql.NullInt64{Int64: int64(res.DurationMs), Valid: true},
	})
	if err != nil {
		fmt.Printf("[Webhook Dispatcher] Failed to log test delivery: %v\n", err)
	}
	d.db.UpdateWebhookDeliveryStats(ctx, db.UpdateWebhookDeliveryStatsParams{
		ID:         webhook.ID,
		Success:    res.Success,
		LastStatus: sql.NullInt64{Int64: int64(res.StatusCode), Valid: res.StatusCode > 0},
	})
	return res
}

// Resend tries a delivery of a webhook again now, whatever its status, and
// returns it as it stands after the try. The try counts like any other: when
// it fails, a delivery with tries left is scheduled again, and one out of
// tries stays failed.
func (d *Dispatcher) Resend(ctx context.Context, webhook db.Webhook, deliveryID string) (db.WebhookDelivery, Result, error) {
	if _, err := d.db.GetWebhookDelivery(ctx, db.GetWebhookDeliveryParams{ID: deliveryID, WebhookID: webhook.ID}); err != nil {
		return db.WebhookDelivery{}, Result{}, err
	}
	now := d.now()
	delivery, err := d.db.ClaimWebhookDelivery(ctx, db.ClaimWebhookDeliveryParams{
		ID:         deliveryID,
		ClaimUntil: sql.NullString{String: d.claimUntil(now), Valid: true},
		Now:        now.UTC().Format(dbTime),
		Force:      true,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return db.WebhookDelivery{}, Result{}, ErrDeliveryInFlight
	}
	if err != nil {
		return db.WebhookDelivery{}, Result{}, err
	}
	res := d.attempt(ctx, webhook, delivery)
	got, err := d.db.GetWebhookDelivery(ctx, db.GetWebhookDeliveryParams{ID: deliveryID, WebhookID: webhook.ID})
	return got, res, err
}

// retryLoop tries due deliveries again until ctx ends, and clears out old
// finished ones.
func (d *Dispatcher) retryLoop(ctx context.Context) {
	defer d.loop.Done()
	ticker := time.NewTicker(d.retry.PollInterval)
	defer ticker.Stop()
	var lastPrune time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.RetryDue(ctx)
			if now := d.now(); now.Sub(lastPrune) >= time.Hour {
				lastPrune = now
				if err := d.db.DeleteFinishedWebhookDeliveries(ctx, now.Add(-d.retry.Keep).UTC().Format(dbTime)); err != nil && ctx.Err() == nil {
					fmt.Printf("[Webhook Dispatcher] Failed to clear old deliveries: %v\n", err)
				}
			}
		}
	}
}

// RetryDue tries every delivery whose next try is due. Each webhook's
// deliveries go oldest first, one at a time; the first that fails ends that
// webhook's turn, so a down endpoint gets one try per look, and the rest
// keep their place for the next look. Webhooks don't wait on each other.
func (d *Dispatcher) RetryDue(ctx context.Context) {
	now := d.now()
	due, err := d.db.ListDueWebhookDeliveries(ctx, db.ListDueWebhookDeliveriesParams{
		Now:        sql.NullString{String: now.UTC().Format(dbTime), Valid: true},
		LimitCount: int64(d.retry.BatchSize),
	})
	if err != nil {
		if ctx.Err() == nil {
			fmt.Printf("[Webhook Dispatcher] Failed to list due deliveries: %v\n", err)
		}
		return
	}
	var order []string
	byWebhook := map[string][]string{}
	for _, row := range due {
		if _, ok := byWebhook[row.WebhookID]; !ok {
			order = append(order, row.WebhookID)
		}
		byWebhook[row.WebhookID] = append(byWebhook[row.WebhookID], row.ID)
	}

	var wg sync.WaitGroup
	for _, webhookID := range order {
		wg.Add(1)
		go func(webhookID string, ids []string) {
			defer wg.Done()
			webhook, err := d.db.GetWebhookByID(ctx, webhookID)
			if err != nil {
				return
			}
			for _, id := range ids {
				now := d.now()
				delivery, err := d.db.ClaimWebhookDelivery(ctx, db.ClaimWebhookDeliveryParams{
					ID:         id,
					ClaimUntil: sql.NullString{String: d.claimUntil(now), Valid: true},
					Now:        now.UTC().Format(dbTime),
					Force:      false,
				})
				if err != nil {
					continue // taken or no longer due
				}
				if !d.attempt(ctx, webhook, delivery).Success {
					return
				}
			}
		}(webhookID, byWebhook[webhookID])
	}
	wg.Wait()
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
