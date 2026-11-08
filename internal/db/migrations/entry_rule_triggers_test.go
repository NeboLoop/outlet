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

// A list rule saved as list_subscribe (the name the MCP tool gave) becomes
// list_join, the trigger that runs.
func TestEntryRuleTriggersMigration(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db"))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	goose.SetBaseFS(embedMigrations)
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.UpTo(conn, ".", 23))
	for _, q := range []string{
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('o', 'O', 'o', 'k')`,
		`INSERT INTO email_sequences (id, org_id, slug, name, trigger_event) VALUES ('s', 'o', 's', 'S', 'manual')`,
		`INSERT INTO sequence_entry_rules (id, sequence_id, trigger_type, source_id) VALUES ('a', 's', 'list_subscribe', '1'), ('b', 's', 'tag_added', 'vip')`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err, q)
	}
	require.NoError(t, Run(conn, nil))

	var a, b string
	require.NoError(t, conn.QueryRow(`SELECT (SELECT trigger_type FROM sequence_entry_rules WHERE id = 'a'), (SELECT trigger_type FROM sequence_entry_rules WHERE id = 'b')`).Scan(&a, &b))
	assert.Equal(t, "list_join", a)
	assert.Equal(t, "tag_added", b)
}
