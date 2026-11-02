package migrations

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// Migration 0014 adds the pool tables and links bounces and complaints to
// pool sends; it rolls back cleanly and keeps existing bounce rows.
func TestMigration0014(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db"))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	goose.SetBaseFS(embedMigrations)
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.UpTo(conn, ".", 13))
	_, err = conn.Exec(`INSERT INTO email_bounce (id, email, email_lower, bounce_type) VALUES ('b1', 'a@x.test', 'a@x.test', 'Permanent')`)
	require.NoError(t, err)

	require.NoError(t, goose.UpTo(conn, ".", 14))
	for _, table := range []string{"pool_domains", "pool_mailboxes", "pool_sends", "pool_pauses"} {
		var n int
		require.NoError(t, conn.QueryRow(`SELECT COUNT(*) FROM `+table).Scan(&n), table)
	}
	var link sql.NullString
	require.NoError(t, conn.QueryRow(`SELECT pool_send_id FROM email_bounce WHERE id = 'b1'`).Scan(&link))
	require.False(t, link.Valid)

	require.NoError(t, goose.DownTo(conn, ".", 13))
	var n int
	require.NoError(t, conn.QueryRow(`SELECT COUNT(*) FROM email_bounce`).Scan(&n))
	require.Equal(t, 1, n)
	require.Error(t, conn.QueryRow(`SELECT COUNT(*) FROM pool_sends`).Scan(&n))
}
