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

// Migration 0028 turns each mailbox pool's one service into a connection:
// a Maildoso pool keeps its key on a Maildoso connection whose mailboxes
// default to 15 a day warming up from 5 by 1, a manual pool gets a manual connection, and each
// pool's outreach domains belong to its connection. Down puts the service
// and key back on the pool.
func TestMigration0028(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db"))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	goose.SetBaseFS(embedMigrations)
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.UpTo(conn, ".", 27))

	exec := func(q string, args ...any) {
		t.Helper()
		_, err := conn.Exec(q, args...)
		require.NoError(t, err)
	}
	exec(`INSERT INTO organizations (id, name, slug, api_key) VALUES ('o1', 'One', 'one', 'k1'), ('o2', 'Two', 'two', 'k2')`)
	exec(`INSERT INTO sending_pathways (id, org_id, name, provider, carries_outreach, mailbox_service, mailbox_service_key_enc)
		VALUES ('pool-md', 'o1', 'Mailboxes', 'mailboxes', 1, 'maildoso', x'0a0b0c'), ('pool-hand', 'o2', 'Mailboxes', 'mailboxes', 1, 'manual', NULL)`)
	exec(`INSERT INTO domain_identities (id, org_id, domain, purpose, pathway_id) VALUES
		('d-md', 'o1', 'getmd.example', 'outreach', 'pool-md'),
		('d-hand', 'o2', 'gethand.example', 'outreach', 'pool-hand'),
		('d-send', 'o1', 'one.example', 'sending', NULL)`)
	exec(`INSERT INTO mailbox_service_quotes (id, org_id, pathway_id, service, kind, request, expires_at)
		VALUES ('q1', 'o1', 'pool-md', 'maildoso', 'domain', '{}', '2099-01-01 00:00:00')`)

	require.NoError(t, goose.UpTo(conn, ".", 28))

	type connection struct {
		id, service, name  string
		key                []byte
		limit, start, step sql.NullInt64
	}
	of := func(pool string) []connection {
		rows, err := conn.Query(`SELECT id, service, name, api_key_enc, daily_limit, warmup_start, warmup_step FROM mailbox_connections WHERE pathway_id = ?`, pool)
		require.NoError(t, err)
		defer rows.Close()
		var out []connection
		for rows.Next() {
			var c connection
			require.NoError(t, rows.Scan(&c.id, &c.service, &c.name, &c.key, &c.limit, &c.start, &c.step))
			out = append(out, c)
		}
		return out
	}
	domainConn := func(id string) sql.NullString {
		var c sql.NullString
		require.NoError(t, conn.QueryRow(`SELECT connection_id FROM domain_identities WHERE id = ?`, id).Scan(&c))
		return c
	}

	md := of("pool-md")
	require.Len(t, md, 1)
	assert.Equal(t, "maildoso", md[0].service)
	assert.Equal(t, "Maildoso", md[0].name)
	assert.Equal(t, []byte{0x0a, 0x0b, 0x0c}, md[0].key, "the same encrypted key")
	assert.Equal(t, sql.NullInt64{Int64: 15, Valid: true}, md[0].limit)
	assert.Equal(t, sql.NullInt64{Int64: 5, Valid: true}, md[0].start, "Maildoso warms up from 5")
	assert.Equal(t, sql.NullInt64{Int64: 1, Valid: true}, md[0].step, "by 1 a day")
	assert.Equal(t, md[0].id, domainConn("d-md").String)

	hand := of("pool-hand")
	require.Len(t, hand, 1)
	assert.Equal(t, "manual", hand[0].service)
	assert.Equal(t, "Added by hand", hand[0].name)
	assert.Nil(t, hand[0].key)
	assert.False(t, hand[0].limit.Valid || hand[0].start.Valid || hand[0].step.Valid, "a manual connection uses the pool's limits")
	assert.Equal(t, hand[0].id, domainConn("d-hand").String)
	assert.False(t, domainConn("d-send").Valid, "a sending domain has no connection")

	var n int
	require.NoError(t, conn.QueryRow(`SELECT COUNT(*) FROM mailbox_service_quotes`).Scan(&n))
	assert.Zero(t, n, "pool-level quotes don't carry over")
	_, err = conn.Exec(`SELECT mailbox_service FROM sending_pathways`)
	assert.Error(t, err, "the pool's own service column is gone")

	require.NoError(t, goose.DownTo(conn, ".", 26))
	var service string
	var key []byte
	require.NoError(t, conn.QueryRow(`SELECT mailbox_service, mailbox_service_key_enc FROM sending_pathways WHERE id = 'pool-md'`).Scan(&service, &key))
	assert.Equal(t, "maildoso", service)
	assert.Equal(t, []byte{0x0a, 0x0b, 0x0c}, key)
}
