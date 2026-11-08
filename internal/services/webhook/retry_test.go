package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/events"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// hit is one request a flaky endpoint got.
type hit struct {
	body      []byte
	signature string
	delivery  string
	attempt   string
}

// flaky is an endpoint that fails its first failures requests, then
// succeeds.
type flaky struct {
	mu       sync.Mutex
	failures int
	hits     []hit
}

func (f *flaky) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.hits = append(f.hits, hit{body, r.Header.Get("X-Webhook-Signature"), r.Header.Get("X-Webhook-Delivery"), r.Header.Get("X-Webhook-Attempt")})
	if len(f.hits) <= f.failures {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (f *flaky) got() []hit {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]hit(nil), f.hits...)
}

// retryFixture is a dispatcher with a clock the test moves, and one webhook
// pointing at a flaky endpoint.
type retryFixture struct {
	ctx   context.Context
	store *db.Store
	d     *Dispatcher
	ep    *flaky
	wh    db.Webhook
	clock time.Time
}

func newRetryFixture(t *testing.T, failures int, cfg RetryConfig) *retryFixture {
	t.Helper()
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	_, err = conn.Exec(`INSERT INTO organizations (id, name, slug, api_key) VALUES ('org1', 'Org', 'org', 'k1')`)
	require.NoError(t, err)

	f := &retryFixture{ctx: context.Background(), store: db.NewStore(conn), ep: &flaky{failures: failures}}
	f.clock = time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	srv := httptest.NewServer(f.ep)
	t.Cleanup(srv.Close)

	f.d = NewDispatcher(f.store.Queries, events.NewSubject(), cfg)
	f.d.now = func() time.Time { return f.clock }
	f.wh, err = f.store.CreateWebhook(f.ctx, db.CreateWebhookParams{ID: "wh1", OrgID: "org1", Url: srv.URL, Secret: "shh", Events: "*", Active: sql.NullInt64{Int64: 1, Valid: true}})
	require.NoError(t, err)
	return f
}

func (f *retryFixture) deliveries(t *testing.T) []db.WebhookDelivery {
	t.Helper()
	ds, err := f.store.ListWebhookDeliveries(f.ctx, db.ListWebhookDeliveriesParams{WebhookID: f.wh.ID, LimitCount: 100})
	require.NoError(t, err)
	return ds
}

// advance moves the clock and runs one look for due retries.
func (f *retryFixture) advance(by time.Duration) {
	f.clock = f.clock.Add(by)
	f.d.RetryDue(f.ctx)
}

func sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// A delivery that fails is tried again after each backoff, not before,
// with the same bytes and the same signature, and is done once one try
// succeeds. Every try is logged.
func TestFailedDeliveryIsRetriedWithBackoffAndTheSameSignature(t *testing.T) {
	f := newRetryFixture(t, 2, DefaultRetryConfig())
	require.NoError(t, f.d.DispatchEvent(f.ctx, "org1", events.TopicContactCreated, map[string]interface{}{"contact_id": "c1"}))

	ds := f.deliveries(t)
	require.Len(t, ds, 1)
	assert.Equal(t, StatusPending, ds[0].Status)
	assert.EqualValues(t, 1, ds[0].Attempts)
	assert.EqualValues(t, 503, ds[0].LastStatusCode.Int64)
	assert.Equal(t, "2026-11-08 12:05:00", ds[0].NextAttemptAt.String, "first retry 5 minutes on")

	// Not due yet: nothing is sent
	f.advance(4*time.Minute + 59*time.Second)
	assert.Len(t, f.ep.got(), 1)

	f.advance(time.Second)
	assert.Len(t, f.ep.got(), 2)
	ds = f.deliveries(t)
	assert.Equal(t, StatusPending, ds[0].Status)
	assert.EqualValues(t, 2, ds[0].Attempts)
	assert.Equal(t, "2026-11-08 13:05:00", ds[0].NextAttemptAt.String, "second retry an hour on")

	f.advance(59 * time.Minute)
	assert.Len(t, f.ep.got(), 2)
	f.advance(time.Minute)
	hits := f.ep.got()
	require.Len(t, hits, 3)

	ds = f.deliveries(t)
	assert.Equal(t, StatusSucceeded, ds[0].Status)
	assert.EqualValues(t, 3, ds[0].Attempts)
	assert.False(t, ds[0].NextAttemptAt.Valid)

	for i, h := range hits {
		assert.Equal(t, hits[0].body, h.body, "try %d sends the same payload", i+1)
		assert.Equal(t, sign("shh", h.body), h.signature, "try %d is signed with the secret", i+1)
		assert.Equal(t, ds[0].ID, h.delivery)
	}
	assert.Equal(t, []string{"1", "2", "3"}, []string{hits[0].attempt, hits[1].attempt, hits[2].attempt})

	logs, err := f.store.ListWebhookLogs(f.ctx, db.ListWebhookLogsParams{WebhookID: f.wh.ID, LimitCount: 10})
	require.NoError(t, err)
	assert.Len(t, logs, 3)
	for _, l := range logs {
		assert.Equal(t, ds[0].ID, l.DeliveryID.String)
	}

	// Done: later looks send nothing more
	f.advance(48 * time.Hour)
	assert.Len(t, f.ep.got(), 3)
}

// A delivery gets 5 tries over about a day, then stops for good.
func TestDeliveryStopsAfterItsLastTry(t *testing.T) {
	cfg := DefaultRetryConfig()
	require.Equal(t, 5, cfg.MaxAttempts())
	var total time.Duration
	for _, b := range cfg.Backoff {
		total += b
	}
	assert.True(t, total >= 20*time.Hour && total <= 30*time.Hour, "tries span about a day, got %v", total)

	f := newRetryFixture(t, 1000, cfg)
	require.NoError(t, f.d.DispatchEvent(f.ctx, "org1", events.TopicContactCreated, nil))
	for _, b := range cfg.Backoff {
		f.advance(b)
	}
	assert.Len(t, f.ep.got(), 5)
	ds := f.deliveries(t)
	assert.Equal(t, StatusFailed, ds[0].Status)
	assert.EqualValues(t, 5, ds[0].Attempts)
	assert.False(t, ds[0].NextAttemptAt.Valid)

	f.advance(7 * 24 * time.Hour)
	assert.Len(t, f.ep.got(), 5, "no tries after the last one")
}

// An endpoint that is down for good can't pile up retries: once its backlog
// is full, a new failure is not retried, and a look tries it once, not once
// per waiting delivery.
func TestDownEndpointBacklogIsBounded(t *testing.T) {
	cfg := DefaultRetryConfig()
	cfg.MaxBacklog = 3
	f := newRetryFixture(t, 1000, cfg)
	for i := 0; i < 5; i++ {
		require.NoError(t, f.d.DispatchEvent(f.ctx, "org1", events.TopicContactCreated, nil))
	}
	byStatus := map[string]int{}
	for _, d := range f.deliveries(t) {
		byStatus[d.Status]++
	}
	assert.Equal(t, map[string]int{StatusPending: 3, StatusFailed: 2}, byStatus)
	assert.Len(t, f.ep.got(), 5)

	// All three are due; the first fails and ends this endpoint's turn
	f.advance(cfg.Backoff[0])
	assert.Len(t, f.ep.got(), 6)

	// Old finished deliveries are cleared out, with their tries
	require.NoError(t, f.store.DeleteFinishedWebhookDeliveries(f.ctx, "2026-11-09 00:00:00"))
	byStatus = map[string]int{}
	for _, d := range f.deliveries(t) {
		byStatus[d.Status]++
	}
	assert.Equal(t, map[string]int{StatusPending: 3}, byStatus)
}

// Each webhook's retries go oldest first.
func TestRetriesGoOldestFirst(t *testing.T) {
	f := newRetryFixture(t, 2, DefaultRetryConfig())
	require.NoError(t, f.d.DispatchEvent(f.ctx, "org1", events.TopicContactCreated, map[string]interface{}{"n": "first"}))
	f.clock = f.clock.Add(time.Second)
	require.NoError(t, f.d.DispatchEvent(f.ctx, "org1", events.TopicContactCreated, map[string]interface{}{"n": "second"}))

	f.advance(10 * time.Minute)
	hits := f.ep.got()
	require.Len(t, hits, 4)
	assert.Contains(t, string(hits[2].body), `"first"`)
	assert.Contains(t, string(hits[3].body), `"second"`)
}

// Send again tries a delivery now, whatever its state, signed the same way.
func TestResendSendsAgainNow(t *testing.T) {
	f := newRetryFixture(t, 1, DefaultRetryConfig())
	require.NoError(t, f.d.DispatchEvent(f.ctx, "org1", events.TopicContactCreated, nil))
	id := f.deliveries(t)[0].ID

	got, res, err := f.d.Resend(f.ctx, f.wh, id)
	require.NoError(t, err)
	assert.True(t, res.Success)
	assert.Equal(t, StatusSucceeded, got.Status)
	assert.EqualValues(t, 2, got.Attempts)

	// A succeeded one can be sent again too
	got, _, err = f.d.Resend(f.ctx, f.wh, id)
	require.NoError(t, err)
	assert.EqualValues(t, 3, got.Attempts)
	hits := f.ep.got()
	require.Len(t, hits, 3)
	assert.Equal(t, hits[0].signature, hits[2].signature)

	// Another webhook's delivery is not found
	_, _, err = f.d.Resend(f.ctx, db.Webhook{ID: "other"}, id)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}
