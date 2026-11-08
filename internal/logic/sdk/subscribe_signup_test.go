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

// A single opt-in subscribe through the API starts the list's "On signup"
// autoresponder, once.
func TestSubscribeToList_StartsSignupSequence(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	for _, q := range []string{
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('org-a', 'A', 'a', 'key-a')`,
		`INSERT INTO email_lists (id, public_id, org_id, name, slug, double_optin) VALUES (1, 'p1', 'org-a', 'News', 'news', 0)`,
		`INSERT INTO email_sequences (id, org_id, list_id, slug, name, trigger_event, is_active) VALUES ('seq-1', 'org-a', 1, 'welcome', 'Welcome', 'signup', 1)`,
		`INSERT INTO email_templates (id, org_id, sequence_id, position, subject, html_body) VALUES ('t-1', 'org-a', 'seq-1', 1, 'Hi', '<p>Hi</p>')`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err, q)
	}
	store := db.NewStore(conn)
	emailService := email.NewService(store, nil)
	s := &svc.ServiceContext{DB: store, EmailService: emailService, Sequences: email.NewSequenceServiceWithBaseURL(store, emailService, "https://outlet.test")}
	ctx := context.WithValue(context.Background(), middleware.OrgIDKey, "org-a")

	for range 2 {
		resp, err := NewSubscribeToListLogic(ctx, s).SubscribeToList(&types.SubscribeRequest{Slug: "news", Email: "ana@example.com"})
		require.NoError(t, err)
		require.True(t, resp.Success, resp.Message)
	}
	var n int
	require.NoError(t, conn.QueryRow(`SELECT COUNT(*) FROM email_queue WHERE template_id = 't-1'`).Scan(&n))
	assert.Equal(t, 1, n)
}
