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

// reverseCipher stands in for the server's key in tests.
type reverseCipher struct{}

func (reverseCipher) EncryptString(s string) ([]byte, error) {
	b := []byte(s)
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return b, nil
}

// Migration 0021 keeps every brand's mail flowing: a platform pathway, or an
// ses pathway with the brand's own AWS keys (moved out of its settings,
// encrypted), carrying transactional and marketing; a mailbox pool carrying
// outreach for a brand with mailboxes or outreach domains; each domain on
// the pathway it serves. Brands made afterwards start on a platform pathway.
func TestMigration0021(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db"))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	goose.SetBaseFS(embedMigrations)
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.UpTo(conn, ".", 20))

	exec := func(q string, args ...any) {
		t.Helper()
		_, err := conn.Exec(q, args...)
		require.NoError(t, err)
	}
	exec(`INSERT INTO organizations (id, name, slug, api_key, settings) VALUES
		('plain', 'Plain', 'plain', 'k1', '{}'),
		('own', 'Own SES', 'own', 'k2', '{"email":{"aws_region":"eu-west-1","aws_access_key":"AKIA1","aws_secret_key":"s3cret","ses_rate_limit":5}}'),
		('cold', 'Cold', 'cold', 'k3', NULL)`)
	exec(`INSERT INTO domain_identities (id, org_id, domain) VALUES ('d-plain', 'plain', 'plain.example'), ('d-own', 'own', 'own.example'), ('d-cold', 'cold', 'cold.example')`)
	exec(`INSERT INTO domain_identities (id, org_id, domain, purpose) VALUES ('d-out', 'cold', 'getcold.example', 'outreach')`)
	exec(`INSERT INTO mailboxes (id, org_id, domain_id, email, from_name, smtp_host, smtp_port, smtp_username, smtp_password_enc, imap_host, imap_port, imap_username, imap_password_enc)
		VALUES ('mb', 'cold', 'd-out', 'sam@getcold.example', 'Sam', 'h', 587, 'sam', x'00', 'h', 993, 'sam', x'00')`)

	// Without the key, keys can't move: the migration fails rather than lose them.
	credentialCipher = nil
	require.Error(t, goose.UpTo(conn, ".", 21))
	credentialCipher = reverseCipher{}
	t.Cleanup(func() { credentialCipher = nil })
	require.NoError(t, goose.UpTo(conn, ".", 21))

	type pathway struct {
		id, provider             string
		transactional, marketing int
		outreach, dailyLimit     int
		warmupStart, minGap      int
		region                   string
		accessKey, secretKey     []byte
	}
	pathways := func(org string) []pathway {
		rows, err := conn.Query(`SELECT id, provider, carries_transactional, carries_marketing, carries_outreach, daily_limit, warmup_start, min_gap_seconds, aws_region, aws_access_key_enc, aws_secret_key_enc
			FROM sending_pathways WHERE org_id = ? ORDER BY provider`, org)
		require.NoError(t, err)
		defer rows.Close()
		var out []pathway
		for rows.Next() {
			var p pathway
			require.NoError(t, rows.Scan(&p.id, &p.provider, &p.transactional, &p.marketing, &p.outreach, &p.dailyLimit, &p.warmupStart, &p.minGap, &p.region, &p.accessKey, &p.secretKey))
			out = append(out, p)
		}
		return out
	}
	domainPathway := func(id string) string {
		var p sql.NullString
		require.NoError(t, conn.QueryRow(`SELECT pathway_id FROM domain_identities WHERE id = ?`, id).Scan(&p))
		return p.String
	}

	plain := pathways("plain")
	require.Len(t, plain, 1)
	assert.Equal(t, "platform", plain[0].provider)
	assert.Equal(t, []int{1, 1, 0}, []int{plain[0].transactional, plain[0].marketing, plain[0].outreach})
	assert.Equal(t, plain[0].id, domainPathway("d-plain"))

	own := pathways("own")
	require.Len(t, own, 1)
	assert.Equal(t, "ses", own[0].provider)
	assert.Equal(t, []int{1, 1, 0}, []int{own[0].transactional, own[0].marketing, own[0].outreach})
	assert.Equal(t, "eu-west-1", own[0].region)
	assert.Equal(t, "1AIKA", string(own[0].accessKey), "encrypted with the server's key")
	assert.Equal(t, "terc3s", string(own[0].secretKey))
	assert.Equal(t, own[0].id, domainPathway("d-own"))
	var settings string
	require.NoError(t, conn.QueryRow(`SELECT settings FROM organizations WHERE id = 'own'`).Scan(&settings))
	assert.NotContains(t, settings, "AKIA1", "the plain-text keys leave the settings")
	assert.Contains(t, settings, "ses_rate_limit", "the rest of the settings stay")

	cold := pathways("cold")
	require.Len(t, cold, 2)
	pool, platform := cold[0], cold[1]
	assert.Equal(t, "mailboxes", pool.provider)
	assert.Equal(t, []int{0, 0, 1}, []int{pool.transactional, pool.marketing, pool.outreach})
	assert.Equal(t, []int{30, 5, 240}, []int{pool.dailyLimit, pool.warmupStart, pool.minGap}, "the pool's defaults")
	assert.Equal(t, "platform", platform.provider)
	assert.Equal(t, platform.id, domainPathway("d-cold"))
	assert.Equal(t, pool.id, domainPathway("d-out"))

	// A brand made afterwards starts on the platform pathway.
	exec(`INSERT INTO organizations (id, name, slug, api_key) VALUES ('new', 'New', 'new', 'k4')`)
	fresh := pathways("new")
	require.Len(t, fresh, 1)
	assert.Equal(t, "platform", fresh[0].provider)
	assert.Equal(t, []int{1, 1, 0}, []int{fresh[0].transactional, fresh[0].marketing, fresh[0].outreach})

	// One pathway per kind per brand.
	_, err = conn.Exec(`INSERT INTO sending_pathways (id, org_id, name, provider, carries_marketing) VALUES ('dup', 'new', 'Dup', 'smtp', 1)`)
	assert.Error(t, err, "marketing already has a pathway")
}
