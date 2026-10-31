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

// Migration 0012 rewrites comma-separated campaign lists into the one stored
// form, and gives OAuth grants made without a scope the default scope.
func TestMigration0012(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db"))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	goose.SetBaseFS(embedMigrations)
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.UpTo(conn, ".", 11))

	_, err = conn.Exec(`INSERT INTO email_campaigns (id, org_id, name, subject, html_body, list_ids, exclude_list_ids) VALUES
		('mcp', 'o', 'n', 's', 'b', '1, 2', '3'),
		('ui', 'o', 'n', 's', 'b', '["4","5"]', NULL),
		('none', 'o', 'n', 's', 'b', '', NULL)`)
	require.NoError(t, err)
	_, err = conn.Exec(`INSERT INTO mcp_oauth_tokens (id, client_id, user_id, access_token_hash, scopes, expires_at) VALUES
		('empty', 'c', 'u', 'h1', '', '2030-01-01T00:00:00Z'),
		('full', 'c', 'u', 'h2', 'mcp:full offline_access', '2030-01-01T00:00:00Z')`)
	require.NoError(t, err)

	require.NoError(t, goose.UpTo(conn, ".", 12))

	lists := func(id string) (string, sql.NullString) {
		var l string
		var x sql.NullString
		require.NoError(t, conn.QueryRow(`SELECT list_ids, exclude_list_ids FROM email_campaigns WHERE id = ?`, id).Scan(&l, &x))
		return l, x
	}
	l, x := lists("mcp")
	assert.Equal(t, `["1","2"]`, l)
	assert.Equal(t, `["3"]`, x.String)
	l, x = lists("ui")
	assert.Equal(t, `["4","5"]`, l)
	assert.False(t, x.Valid)
	l, _ = lists("none")
	assert.Equal(t, `[]`, l)

	scopes := func(id string) string {
		var s string
		require.NoError(t, conn.QueryRow(`SELECT scopes FROM mcp_oauth_tokens WHERE id = ?`, id).Scan(&s))
		return s
	}
	assert.Equal(t, "mcp:full", scopes("empty"))
	assert.Equal(t, "mcp:full offline_access", scopes("full"))
}
