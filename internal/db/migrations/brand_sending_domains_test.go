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

// Migration 0013 gives each brand a domain identity for its from_email's
// domain when it has none, so brands made before domain identities keep
// sending; a brand that already holds the domain, a domain another brand
// holds, and a brand with no from_email are left alone.
func TestMigration0013(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db"))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	goose.SetBaseFS(embedMigrations)
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.UpTo(conn, ".", 12))

	_, err = conn.Exec(`INSERT INTO organizations (id, name, slug, api_key, from_email) VALUES
		('nebo', 'NeboAI', 'neboai', 'k1', 'hello@Mail.NeboAI.com'),
		('bot', 'Nebo Bot', 'nebo-bot', 'k2', ' noreply@nebo.bot '),
		('held', 'Held', 'held', 'k3', 'a@held.example'),
		('taken', 'Taken', 'taken', 'k4', 'b@shared.example'),
		('owner', 'Owner', 'owner', 'k5', 'c@other.example'),
		('blank', 'Blank', 'blank', 'k6', NULL)`)
	require.NoError(t, err)
	_, err = conn.Exec(`INSERT INTO domain_identities (id, org_id, domain, verification_status) VALUES
		('d1', 'held', 'held.example', 'pending'),
		('d2', 'owner', 'shared.example', 'success')`)
	require.NoError(t, err)

	require.NoError(t, goose.UpTo(conn, ".", 13))

	domains := func(org string) []string {
		rows, err := conn.Query(`SELECT domain || ':' || verification_status FROM domain_identities WHERE org_id = ? ORDER BY domain`, org)
		require.NoError(t, err)
		defer rows.Close()
		var out []string
		for rows.Next() {
			var d string
			require.NoError(t, rows.Scan(&d))
			out = append(out, d)
		}
		return out
	}
	assert.Equal(t, []string{"mail.neboai.com:success"}, domains("nebo"))
	assert.Equal(t, []string{"nebo.bot:success"}, domains("bot"))
	assert.Equal(t, []string{"held.example:pending"}, domains("held"))
	assert.Empty(t, domains("taken"))
	assert.Equal(t, []string{"other.example:success", "shared.example:success"}, domains("owner"))
	assert.Empty(t, domains("blank"))

	var id string
	require.NoError(t, conn.QueryRow(`SELECT id FROM domain_identities WHERE org_id = 'nebo'`).Scan(&id))
	assert.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`, id)

	// A second run adds nothing.
	_, err = conn.Exec(`DELETE FROM goose_db_version WHERE version_id = 13`)
	require.NoError(t, err)
	require.NoError(t, goose.UpTo(conn, ".", 13))
	assert.Len(t, domains("nebo"), 1)
}
