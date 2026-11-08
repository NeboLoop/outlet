package webhook

import (
	"context"
	"database/sql"
	"encoding/json"
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
	"github.com/outlet-sh/outlet/internal/services/tracking"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// receiver records the webhook deliveries it gets.
type receiver struct {
	mu  sync.Mutex
	got []WebhookPayload
}

func (r *receiver) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	body, _ := io.ReadAll(req.Body)
	var p WebhookPayload
	_ = json.Unmarshal(body, &p)
	r.mu.Lock()
	r.got = append(r.got, p)
	r.mu.Unlock()
	w.WriteHeader(http.StatusOK)
}

// wait returns the first delivery of event, failing the test when none comes.
func (r *receiver) wait(t *testing.T, event string) WebhookPayload {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		r.mu.Lock()
		for i, p := range r.got {
			if p.Event == event {
				r.got = append(r.got[:i], r.got[i+1:]...)
				r.mu.Unlock()
				return p
			}
		}
		r.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("no %s delivery", event)
	return WebhookPayload{}
}

// Every event a webhook can subscribe to reaches it, from the write that
// makes it happen, whichever path made that write.
func TestEveryEventReachesAWebhook(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	ctx := context.Background()

	bus := events.NewSubject(events.WithBufferSize(64))
	t.Cleanup(func() { events.Complete(bus) })
	store := db.NewStore(conn)
	store.SetEvents(bus)
	d := NewDispatcher(store.Queries, bus, DefaultRetryConfig())
	require.NoError(t, d.Start(ctx))
	t.Cleanup(d.Stop)

	rcv := &receiver{}
	srv := httptest.NewServer(rcv)
	t.Cleanup(srv.Close)

	for _, q := range []string{
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('org1', 'Org', 'org', 'k1'), ('org2', 'Other', 'other', 'k2')`,
		`INSERT INTO email_lists (id, public_id, org_id, name, slug) VALUES (7, 'pub7', 'org1', 'News', 'news')`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err)
	}
	subscribed, err := EncodeEvents(Events)
	require.NoError(t, err)
	_, err = store.CreateWebhook(ctx, db.CreateWebhookParams{ID: "wh1", OrgID: "org1", Url: srv.URL, Secret: "s", Events: subscribed, Active: sql.NullInt64{Int64: 1, Valid: true}})
	require.NoError(t, err)

	// contact.created, from any path that creates a contact
	c, err := store.CreateContact(ctx, db.CreateContactParams{ID: "c1", OrgID: sql.NullString{String: "org1", Valid: true}, Name: "Ann", Email: "ann@example.com", Source: sql.NullString{String: "form", Valid: true}, Status: "new"})
	require.NoError(t, err)
	p := rcv.wait(t, events.TopicContactCreated)
	assert.Equal(t, "org1", p.Data["org_id"])
	assert.Equal(t, c.ID, p.Data["contact_id"])
	assert.Equal(t, "form", p.Data["source"])

	// A contact of another brand is not this brand's webhook's business
	_, err = store.CreateContact(ctx, db.CreateContactParams{ID: "c2", OrgID: sql.NullString{String: "org2", Valid: true}, Name: "Bo", Email: "bo@example.com", Status: "new"})
	require.NoError(t, err)

	// contact.unsubscribed, from a list and from everything
	_, err = conn.Exec(`INSERT INTO list_subscribers (id, list_id, contact_id) VALUES ('ls1', 7, 'c1')`)
	require.NoError(t, err)
	require.NoError(t, store.UnsubscribeFromList(ctx, db.UnsubscribeFromListParams{ListID: 7, ContactID: "c1"}))
	p = rcv.wait(t, events.TopicContactUnsubscribed)
	assert.Equal(t, "7", p.Data["list_id"])
	require.NoError(t, store.UnsubscribeContact(ctx, "c1"))
	p = rcv.wait(t, events.TopicContactUnsubscribed)
	assert.Equal(t, "ann@example.com", p.Data["email"])

	// email.sent for a campaign send, a queued (sequence) email and a
	// transactional send reports the subject the person got, its merge tags
	// filled, not the template's
	for _, q := range []string{
		`INSERT INTO email_campaigns (id, org_id, name, subject, html_body) VALUES ('camp1', 'org1', 'Launch', 'Hello {{first_name}}', '<p>hi</p>')`,
		`INSERT INTO campaign_sends (id, campaign_id, contact_id, list_id, tracking_token) VALUES ('cs1', 'camp1', 'c1', 7, 'tok-campaign')`,
		`INSERT INTO email_templates (id, subject, html_body, position) VALUES ('et1', 'Welcome {{name}}', '<p>w</p>', 1)`,
		`INSERT INTO email_queue (id, contact_id, template_id, scheduled_for, tracking_token) VALUES ('eq1', 'c1', 'et1', datetime('now'), 'tok-queue')`,
		`INSERT INTO transactional_emails (id, org_id, name, slug, subject, html_body) VALUES ('te1', 'org1', 'Receipt', 'receipt', 'Your receipt, {{name}}', '<p>r</p>')`,
		`INSERT INTO transactional_sends (id, template_id, org_id, to_email, subject) VALUES ('ts1', 'te1', 'org1', 'buyer@example.com', 'Your receipt, Bea')`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err)
	}
	require.NoError(t, store.MarkCampaignSendSent(ctx, db.MarkCampaignSendSentParams{ID: "cs1", Subject: sql.NullString{String: "Hello Ann", Valid: true}}))
	p = rcv.wait(t, events.TopicEmailSent)
	assert.Equal(t, "camp1", p.Data["campaign_id"])
	assert.Equal(t, "Hello Ann", p.Data["subject"])
	require.NoError(t, store.MarkEmailSent(ctx, db.MarkEmailSentParams{ID: "eq1", Subject: sql.NullString{String: "Welcome Ann", Valid: true}}))
	p = rcv.wait(t, events.TopicEmailSent)
	assert.Equal(t, "eq1", p.Data["email_id"])
	assert.Equal(t, "Welcome Ann", p.Data["subject"])
	require.NoError(t, store.UpdateTransactionalSendStatus(ctx, db.UpdateTransactionalSendStatusParams{ID: "ts1", Status: sql.NullString{String: "sent", Valid: true}}))
	p = rcv.wait(t, events.TopicEmailSent)
	assert.Equal(t, "buyer@example.com", p.Data["email"])
	assert.Equal(t, "Your receipt, Bea", p.Data["subject"])

	// email.opened and email.clicked for campaign and queued mail
	tr := tracking.New(store, bus)
	require.NoError(t, tr.RecordOpen(ctx, "tok-campaign"))
	p = rcv.wait(t, events.TopicEmailOpened)
	assert.Equal(t, "camp1", p.Data["campaign_id"])
	assert.Equal(t, "Hello Ann", p.Data["subject"])
	require.NoError(t, tr.RecordClick(ctx, "tok-campaign", "https://example.com/a"))
	p = rcv.wait(t, events.TopicEmailClicked)
	assert.Equal(t, "https://example.com/a", p.Data["clicked_url"])
	require.NoError(t, tr.RecordOpen(ctx, "tok-queue"))
	p = rcv.wait(t, events.TopicEmailOpened)
	assert.Equal(t, "org1", p.Data["org_id"])
	require.NoError(t, tr.RecordClick(ctx, "tok-queue", "https://example.com/b"))
	rcv.wait(t, events.TopicEmailClicked)

	// Every delivery was for org1: nothing of org2 arrived. Each one is logged.
	time.Sleep(50 * time.Millisecond)
	logs, err := store.ListWebhookLogs(ctx, db.ListWebhookLogsParams{WebhookID: "wh1", LimitCount: 100})
	require.NoError(t, err)
	assert.Len(t, logs, 10)
	rcv.mu.Lock()
	defer rcv.mu.Unlock()
	for _, p := range rcv.got {
		assert.Equal(t, "org1", p.Data["org_id"], "%s", p.Event)
	}
}

func TestEncodeEventsRefusesEventsThatDontExist(t *testing.T) {
	_, err := EncodeEvents([]string{"contact.created", "order.created"})
	assert.ErrorContains(t, err, "order.created")
	_, err = EncodeEvents(nil)
	assert.Error(t, err)
	got, err := EncodeEvents([]string{" contact.created", "email.sent "})
	require.NoError(t, err)
	assert.Equal(t, "contact.created,email.sent", got)
	assert.Equal(t, []string{"contact.created", "email.sent"}, DecodeEvents(got))
}
