package migrations

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// A seeded Branded design loses its link to the preferences page Outlet
// doesn't have, and keeps its unsubscribe link.
func TestDesignsWithoutPreferencesLinkMigration(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db"))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	goose.SetBaseFS(embedMigrations)
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.UpTo(conn, ".", 26))
	seeded := `<p style="margin: 12px 0 0 0;">
        <a href="{{unsubscribe_url}}" style="color: #6b7280;">Unsubscribe</a> ·
        <a href="{{preferences_url}}" style="color: #6b7280;">Email Preferences</a>
      </p>`
	for _, q := range []string{
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('o', 'O', 'o', 'k')`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err, q)
	}
	_, err = conn.Exec(`INSERT INTO email_designs (org_id, name, slug, html_body) VALUES ('o', 'Branded', 'branded', ?)`, seeded)
	require.NoError(t, err)
	require.NoError(t, Run(conn, nil))

	var html string
	require.NoError(t, conn.QueryRow(`SELECT html_body FROM email_designs WHERE slug = 'branded'`).Scan(&html))
	assert.NotContains(t, html, "preferences_url")
	assert.Contains(t, html, `<a href="{{unsubscribe_url}}" style="color: #6b7280;">Unsubscribe</a>
      </p>`)
}
