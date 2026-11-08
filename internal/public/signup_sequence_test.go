package public

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// newSignupFixture makes a brand with two lists, one single opt-in and one
// double opt-in, each with an active "On signup" autoresponder of one email.
func newSignupFixture(t *testing.T) (*Handler, *db.Store) {
	t.Helper()
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	for _, q := range []string{
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('org-a', 'A', 'a', 'key-a')`,
		`INSERT INTO email_lists (id, public_id, org_id, name, slug, double_optin) VALUES (1, 'single', 'org-a', 'Single', 'single', 0)`,
		`INSERT INTO email_lists (id, public_id, org_id, name, slug, double_optin) VALUES (2, 'double', 'org-a', 'Double', 'double', 1)`,
		`INSERT INTO email_sequences (id, org_id, list_id, slug, name, trigger_event, is_active) VALUES ('seq-1', 'org-a', 1, 'welcome-1', 'Welcome', 'signup', 1)`,
		`INSERT INTO email_sequences (id, org_id, list_id, slug, name, trigger_event, is_active) VALUES ('seq-2', 'org-a', 2, 'welcome-2', 'Welcome', 'signup', 1)`,
		`INSERT INTO email_templates (id, org_id, sequence_id, position, subject, html_body) VALUES ('t-1', 'org-a', 'seq-1', 1, 'Hi', '<p>Hi</p>')`,
		`INSERT INTO email_templates (id, org_id, sequence_id, position, subject, html_body) VALUES ('t-2', 'org-a', 'seq-2', 1, 'Hi', '<p>Hi</p>')`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err, q)
	}
	store := db.NewStore(conn)
	emailService := email.NewService(store, nil)
	s := &svc.ServiceContext{DB: store, EmailService: emailService, Sequences: email.NewSequenceServiceWithBaseURL(store, emailService, "https://outlet.test")}
	store.OnSubscriptionActivated(s.Sequences.SubscriptionActivated)
	s.Config.App.BaseURL = "https://outlet.test"
	h, err := NewHandler(s)
	require.NoError(t, err)
	return h, store
}

func subscribe(t *testing.T, h *Handler, publicID, address string) {
	t.Helper()
	w := postSubscribe(h, publicID, address)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

// queued counts the contact's signup emails waiting in the queue.
func queued(t *testing.T, store *db.Store, address, sequenceID string) int {
	t.Helper()
	var n int
	require.NoError(t, store.GetDB().QueryRow(`
		SELECT COUNT(*) FROM email_queue q
		JOIN contacts c ON c.id = q.contact_id
		JOIN email_templates t ON t.id = q.template_id
		WHERE c.email = ? AND t.sequence_id = ? AND q.status = 'pending'`, address, sequenceID).Scan(&n))
	return n
}

func TestSignupSequence_SingleOptInStartsOnSubscribe(t *testing.T) {
	h, store := newSignupFixture(t)

	subscribe(t, h, "single", "ana@example.com")
	assert.Equal(t, 1, queued(t, store, "ana@example.com", "seq-1"), "the welcome email is queued")

	// Subscribing again doesn't start the sequence a second time.
	subscribe(t, h, "single", "ana@example.com")
	assert.Equal(t, 1, queued(t, store, "ana@example.com", "seq-1"))
}

func TestSignupSequence_DoubleOptInStartsOnConfirm(t *testing.T) {
	h, store := newSignupFixture(t)

	subscribe(t, h, "double", "bo@example.com")
	assert.Equal(t, 0, queued(t, store, "bo@example.com", "seq-2"), "nothing until the address is confirmed")

	var token string
	require.NoError(t, store.GetDB().QueryRow(`
		SELECT ls.verification_token FROM list_subscribers ls JOIN contacts c ON c.id = ls.contact_id
		WHERE c.email = 'bo@example.com' AND ls.list_id = 2`).Scan(&token))

	confirm := func() {
		w := httptest.NewRecorder()
		h.HandleConfirm(w, httptest.NewRequest(http.MethodGet, "/confirm/"+token, nil))
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	}
	confirm()
	assert.Equal(t, 1, queued(t, store, "bo@example.com", "seq-2"), "confirming starts the welcome sequence")

	// A link scanner opening the link again changes nothing.
	confirm()
	assert.Equal(t, 1, queued(t, store, "bo@example.com", "seq-2"))
}
