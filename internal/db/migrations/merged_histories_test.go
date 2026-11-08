package migrations

import (
	"database/sql"
	"io/fs"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// outlet-sh/outlet and NeboLoop/outlet each numbered migrations 0010 to 0012
// before they were merged. A database from either history ends with every
// table, column and fix after Run.
func TestBothHistoriesEndComplete(t *testing.T) {
	// outlet-sh's history: its 0010 to 0012 are today's 0015 to 0017.
	outletSh := fstest.MapFS{}
	for _, f := range mustGlob(t) {
		raw, err := fs.ReadFile(embedMigrations, f)
		require.NoError(t, err)
		switch f[:4] {
		case "0015":
			outletSh["0010_email_validation.sql"] = &fstest.MapFile{Data: raw}
		case "0016":
			outletSh["0011_signal_network.sql"] = &fstest.MapFile{Data: raw}
		case "0017":
			outletSh["0012_validation_indexes.sql"] = &fstest.MapFile{Data: raw}
		default:
			if f[:4] < "0010" {
				outletSh[f] = &fstest.MapFile{Data: raw}
			}
		}
	}

	for name, history := range map[string]fs.FS{"outlet-sh": outletSh, "NeboLoop": nil} {
		t.Run(name, func(t *testing.T) {
			conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db"))
			require.NoError(t, err)
			t.Cleanup(func() { conn.Close() })
			require.NoError(t, goose.SetDialect("sqlite3"))
			if history != nil {
				goose.SetBaseFS(history)
				require.NoError(t, goose.UpTo(conn, ".", 12))
				// outlet-sh's own 0010 also added these columns.
				for _, col := range []string{"validation_status", "validation_sub_status", "validated_at"} {
					_, err = conn.Exec("ALTER TABLE contacts ADD COLUMN " + col + " TEXT")
					require.NoError(t, err)
				}
			} else {
				goose.SetBaseFS(embedMigrations)
				require.NoError(t, goose.UpTo(conn, ".", 13))
			}
			// A campaign stored in the old comma form, which 0012 rewrites.
			_, err = conn.Exec(`INSERT INTO email_campaigns (id, org_id, name, subject, html_body, list_ids) VALUES ('c', 'o', 'n', 's', 'b', '1, 2')`)
			require.NoError(t, err)

			require.NoError(t, Run(conn))

			for _, table := range []string{"mcp_audit_log", "outbound_messages", "outbound_links", "validation_jobs", "network_signals"} {
				var n int
				require.NoError(t, conn.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&n))
				assert.Equal(t, 1, n, table)
			}
			var n int
			require.NoError(t, conn.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('campaign_sends') WHERE name = 'claim_token'`).Scan(&n))
			assert.Equal(t, 1, n, "send claims")
			require.NoError(t, conn.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('contacts') WHERE name LIKE 'validat%'`).Scan(&n))
			assert.Equal(t, 3, n, "validation columns, each once")
			var lists string
			require.NoError(t, conn.QueryRow(`SELECT list_ids FROM email_campaigns WHERE id = 'c'`).Scan(&lists))
			assert.Equal(t, `["1","2"]`, lists)
		})
	}
	goose.SetBaseFS(embedMigrations)
}

func mustGlob(t *testing.T) []string {
	files, err := fs.Glob(embedMigrations, "*.sql")
	require.NoError(t, err)
	return files
}
