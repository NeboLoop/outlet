package public

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func newSubscribeFixture(t *testing.T) (*Handler, *db.Store) {
	t.Helper()
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn))
	store := db.NewStore(conn)
	for _, q := range []string{
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('org1', 'Brand', 'brand', 'k1')`,
		// Double opt-in list; no email is configured, so a confirmation can't be sent
		`INSERT INTO email_lists (id, public_id, org_id, name, slug, double_optin) VALUES (1, 'news', 'org1', 'News', 'news', 1)`,
		`INSERT INTO blocked_domains (org_id, domain) VALUES ('org1', 'spam.example')`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err, q)
	}
	h, err := NewHandler(&svc.ServiceContext{DB: store, EmailService: email.NewService(store, nil)})
	require.NoError(t, err)
	return h, store
}

func subscribe(h *Handler, address string) string {
	form := url.Values{"email": {address}, "name": {"Pat"}}
	req := httptest.NewRequest(http.MethodPost, "/s/news", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.HandleSubscribe(rec, req)
	return rec.Body.String()
}

// An address at a blocked domain is refused: no contact is stored, and the
// attempt is counted on the domain.
func TestHostedForm_RefusesABlockedDomain(t *testing.T) {
	h, store := newSubscribeFixture(t)
	ctx := context.Background()

	body := subscribe(h, "someone@SPAM.example")
	assert.Contains(t, body, "can&#39;t subscribe")

	_, err := store.GetContactByOrgAndEmail(ctx, db.GetContactByOrgAndEmailParams{OrgID: sql.NullString{String: "org1", Valid: true}, Email: "someone@spam.example"})
	assert.ErrorIs(t, err, sql.ErrNoRows)
	domains, err := store.ListBlockedDomains(ctx, db.ListBlockedDomainsParams{OrgID: "org1", PageSize: 10})
	require.NoError(t, err)
	require.Len(t, domains, 1)
	assert.EqualValues(t, 1, domains[0].BlockAttempts.Int64)

	// Every other path that creates a contact is refused too
	_, err = store.CreateContact(ctx, db.CreateContactParams{ID: "c", OrgID: sql.NullString{String: "org1", Valid: true}, Email: "x@spam.example"})
	assert.ErrorIs(t, err, db.ErrBlockedDomain)
}

// When the confirmation email can't be sent, the form says so instead of
// telling the person to check their inbox.
func TestHostedForm_SaysWhenTheConfirmationWasntSent(t *testing.T) {
	h, _ := newSubscribeFixture(t)

	body := subscribe(h, "pat@example.com")
	assert.NotContains(t, body, "sent a confirmation email")
	assert.Contains(t, body, "couldn&#39;t send your confirmation email")
}

// Every public page names the app's icon, so browsers don't ask for a
// /favicon.ico that isn't there.
func TestPublicPagesNameTheirIcon(t *testing.T) {
	entries, err := Templates.ReadDir("templates")
	require.NoError(t, err)
	require.NotEmpty(t, entries)
	for _, e := range entries {
		b, err := Templates.ReadFile("templates/" + e.Name())
		require.NoError(t, err)
		assert.Contains(t, string(b), `<link rel="icon" href="/favicon.png">`, e.Name())
	}
}
