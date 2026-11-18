package sdk

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/middleware"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// newSubscribeHarness is a brand with a single opt-in list "news" and a
// double opt-in list "confirmed", each with an "On signup" autoresponder.
func newSubscribeHarness(t *testing.T) (*sql.DB, *svc.ServiceContext, context.Context) {
	t.Helper()
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	for _, q := range []string{
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('org-a', 'A', 'a', 'key-a')`,
		`INSERT INTO email_lists (id, public_id, org_id, name, slug, double_optin) VALUES (1, 'p1', 'org-a', 'News', 'news', 0)`,
		`INSERT INTO email_lists (id, public_id, org_id, name, slug, double_optin) VALUES (2, 'p2', 'org-a', 'Confirmed', 'confirmed', 1)`,
		`INSERT INTO email_sequences (id, org_id, list_id, slug, name, trigger_event, is_active) VALUES ('seq-1', 'org-a', 1, 'welcome', 'Welcome', 'signup', 1)`,
		`INSERT INTO email_templates (id, org_id, sequence_id, position, subject, html_body) VALUES ('t-1', 'org-a', 'seq-1', 1, 'Hi', '<p>Hi</p>')`,
		`INSERT INTO email_sequences (id, org_id, list_id, slug, name, trigger_event, is_active) VALUES ('seq-2', 'org-a', 2, 'welcome-2', 'Welcome', 'signup', 1)`,
		`INSERT INTO email_templates (id, org_id, sequence_id, position, subject, html_body) VALUES ('t-2', 'org-a', 'seq-2', 1, 'Hi', '<p>Hi</p>')`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err, q)
	}
	store := db.NewStore(conn)
	emailService := email.NewService(store, nil)
	s := &svc.ServiceContext{DB: store, EmailService: emailService, Sequences: email.NewSequenceServiceWithBaseURL(store, emailService, "https://outlet.test")}
	store.OnSubscriptionActivated(s.Sequences.SubscriptionActivated)
	org, err := store.GetOrganizationByID(context.Background(), "org-a")
	require.NoError(t, err)
	ctx := context.WithValue(context.WithValue(context.Background(), middleware.OrgIDKey, "org-a"), middleware.OrgKey, org)
	return conn, s, ctx
}

func count(t *testing.T, conn *sql.DB, query string) int {
	t.Helper()
	var n int
	require.NoError(t, conn.QueryRow(query).Scan(&n))
	return n
}

// A single opt-in subscribe through the API starts the list's "On signup"
// autoresponder, once. Subscribing the same address twice (a retry, a
// repeated call) is safe: one contact, one subscriber row, the sequence
// started once, and no error.
func TestSubscribeToList_StartsSignupSequence(t *testing.T) {
	conn, s, ctx := newSubscribeHarness(t)

	for i := range 2 {
		resp, err := NewSubscribeToListLogic(ctx, s).SubscribeToList(&types.SubscribeRequest{Slug: "news", Email: "ana@example.com"})
		require.NoError(t, err)
		require.True(t, resp.Success, resp.Message)
		if i == 1 {
			assert.Equal(t, "Already subscribed", resp.Message)
		}
	}
	assert.Equal(t, 1, count(t, conn, `SELECT COUNT(*) FROM contacts WHERE email = 'ana@example.com'`))
	assert.Equal(t, 1, count(t, conn, `SELECT COUNT(*) FROM list_subscribers WHERE list_id = 1 AND status = 'active'`))
	assert.Equal(t, 1, count(t, conn, `SELECT COUNT(*) FROM contact_sequence_state WHERE sequence_id = 'seq-1'`))
	assert.Equal(t, 1, count(t, conn, `SELECT COUNT(*) FROM email_queue WHERE template_id = 't-1'`))
}

// An API subscribe never undoes an unsubscribe: someone who unsubscribed
// from the list stays unsubscribed, single or double opt-in, however often
// the call is made. It succeeds as "Previously unsubscribed", starts no
// sequence and sends nothing.
func TestSubscribeToList_KeepsAnUnsubscribe(t *testing.T) {
	for _, slug := range []string{"news", "confirmed"} {
		t.Run(slug, func(t *testing.T) {
			conn, s, ctx := newSubscribeHarness(t)
			for _, q := range []string{
				`INSERT INTO contacts (id, org_id, name, email) VALUES ('c1', 'org-a', 'Ana', 'ana@example.com')`,
				`INSERT INTO list_subscribers (id, list_id, contact_id, status, unsubscribed_at)
					VALUES ('s1', (SELECT id FROM email_lists WHERE slug = '` + slug + `'), 'c1', 'unsubscribed', datetime('now'))`,
			} {
				_, err := conn.Exec(q)
				require.NoError(t, err, q)
			}
			for range 2 {
				resp, err := NewSubscribeToListLogic(ctx, s).SubscribeToList(&types.SubscribeRequest{Slug: slug, Email: "ana@example.com"})
				require.NoError(t, err)
				assert.True(t, resp.Success)
				assert.Equal(t, "Previously unsubscribed", resp.Message)
			}
			var status string
			require.NoError(t, conn.QueryRow(`SELECT status FROM list_subscribers WHERE id = 's1'`).Scan(&status))
			assert.Equal(t, "unsubscribed", status)
			assert.Equal(t, 0, count(t, conn, `SELECT COUNT(*) FROM contact_sequence_state`))
			assert.Equal(t, 0, count(t, conn, `SELECT COUNT(*) FROM email_queue`))
		})
	}
}

// Someone who unsubscribed from the whole brand (global unsubscribe) is not
// subscribed to any list by the API, and nothing is queued for them.
func TestSubscribeToList_KeepsAGlobalUnsubscribe(t *testing.T) {
	conn, s, ctx := newSubscribeHarness(t)
	_, err := conn.Exec(`INSERT INTO contacts (id, org_id, name, email) VALUES ('c1', 'org-a', 'Ana', 'ana@example.com')`)
	require.NoError(t, err)
	require.NoError(t, s.DB.GlobalUnsubscribeByOrgAndEmail(ctx, db.GlobalUnsubscribeByOrgAndEmailParams{
		OrgID: sql.NullString{String: "org-a", Valid: true}, Email: "ana@example.com",
	}))

	for _, slug := range []string{"news", "confirmed"} {
		resp, err := NewSubscribeToListLogic(ctx, s).SubscribeToList(&types.SubscribeRequest{Slug: slug, Email: "ana@example.com"})
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "Previously unsubscribed", resp.Message, slug)
	}
	assert.Equal(t, 0, count(t, conn, `SELECT COUNT(*) FROM list_subscribers`))
	assert.Equal(t, 0, count(t, conn, `SELECT COUNT(*) FROM email_queue`))
}

// The store's SubscribeToList, whoever calls it, keeps an unsubscribed
// subscriber unsubscribed and starts nothing for them.
func TestStoreSubscribeToList_NeverReactivates(t *testing.T) {
	conn, s, ctx := newSubscribeHarness(t)
	for _, q := range []string{
		`INSERT INTO contacts (id, org_id, name, email) VALUES ('c1', 'org-a', 'Ana', 'ana@example.com')`,
		`INSERT INTO list_subscribers (id, list_id, contact_id, status, unsubscribed_at) VALUES ('s1', 1, 'c1', 'unsubscribed', datetime('now'))`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err, q)
	}
	sub, err := s.DB.SubscribeToList(ctx, db.SubscribeToListParams{ID: "s2", ListID: 1, ContactID: "c1"})
	require.NoError(t, err)
	assert.Equal(t, "unsubscribed", sub.Status.String)
	assert.True(t, sub.UnsubscribedAt.Valid)
	assert.Equal(t, 0, count(t, conn, `SELECT COUNT(*) FROM email_queue`))
}
