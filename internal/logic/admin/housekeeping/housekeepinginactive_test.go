package housekeeping

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// The opens and clicks windows are the ones asked for, engagement on every
// kind of email counts, and the preview is exactly what the delete removes.
func TestHousekeepingInactive_UsesBothWindowsAndEveryKindOfEmail(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	store := db.NewStore(conn)
	ctx := context.Background()

	contact := func(id string) string {
		return `INSERT INTO contacts (id, org_id, name, email, email_verified, created_at) VALUES ('` + id + `', 'org1', '` + id + `', '` + id + `@example.com', 1, datetime('now', '-400 days'))`
	}
	for _, q := range []string{
		`INSERT INTO users (id, email, password_hash, name, role, status) VALUES ('anne', 'anne@example.com', 'x', 'Anne', 'agent', 'active')`,
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('org1', 'One', 'one', 'k1')`,
		`INSERT INTO org_users (org_id, user_id, role) VALUES ('org1', 'anne', 'admin')`,
		`INSERT INTO email_campaigns (id, org_id, name, subject, html_body) VALUES ('camp', 'org1', 'C', 'S', 'B')`,
		`INSERT INTO email_sequences (id, org_id, slug, name, trigger_event) VALUES ('seq', 'org1', 's', 'S', 'manual')`,
		`INSERT INTO email_templates (id, org_id, sequence_id, position, subject, html_body) VALUES ('tpl', 'org1', 'seq', 1, 'S', 'B')`,
		`INSERT INTO transactional_emails (id, org_id, name, slug, subject, html_body) VALUES ('tx', 'org1', 'T', 't', 'S', 'B')`,
		contact("idle"),
		contact("campaign_open"),
		contact("campaign_click"),
		contact("sequence_open"),
		contact("outreach_click"),
		contact("transactional_open"),
		contact("old_open"),
		contact("click_120"),
		// Too new to have had a full window: kept.
		`INSERT INTO contacts (id, org_id, name, email, email_verified, created_at) VALUES ('newcomer', 'org1', 'N', 'newcomer@example.com', 1, datetime('now', '-5 days'))`,
		`INSERT INTO campaign_sends (id, campaign_id, contact_id, opened_at) VALUES
			('cs1', 'camp', 'campaign_open', datetime('now', '-10 days')),
			('cs2', 'camp', 'old_open', datetime('now', '-200 days'))`,
		`INSERT INTO campaign_clicks (id, campaign_send_id, campaign_id, contact_id, link_url, clicked_at) VALUES
			('cc1', 'cs2', 'camp', 'campaign_click', 'https://x', datetime('now', '-10 days'))`,
		`INSERT INTO email_queue (id, contact_id, template_id, scheduled_for, opened_at, clicked_at, kind) VALUES
			('q1', 'sequence_open', 'tpl', datetime('now'), datetime('now', '-20 days'), NULL, 'marketing'),
			('q2', 'outreach_click', 'tpl', datetime('now'), NULL, datetime('now', '-20 days'), 'outreach'),
			('q3', 'click_120', 'tpl', datetime('now'), NULL, datetime('now', '-120 days'), 'marketing')`,
		`INSERT INTO transactional_sends (id, template_id, org_id, to_email, opened_at) VALUES
			('t1', 'tx', 'org1', 'Transactional_Open@example.com', datetime('now', '-3 days'))`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err, q)
	}
	user, err := access.Active(ctx, store, "anne")
	require.NoError(t, err)
	asAdmin := access.WithBrand(access.WithUser(ctx, user), "org1")
	run := func(opens, clicks int, dryRun bool) *types.HousekeepingResponse {
		got, err := NewHousekeepingInactiveLogic(asAdmin, &svc.ServiceContext{DB: store}).HousekeepingInactive(&types.HousekeepingInactiveRequest{NoOpensDays: opens, NoClicksDays: clicks, DryRun: dryRun})
		require.NoError(t, err)
		return got
	}

	// 90 days of opens, 180 of clicks: idle and old_open go; click_120 clicked
	// inside the clicks window and stays.
	preview := run(90, 180, true)
	assert.Equal(t, 2, preview.AffectedCount)
	assert.Contains(t, preview.Message, "no opens in 90 days and no clicks in 180 days")

	// A shorter clicks window lets click_120 go too.
	assert.Equal(t, 3, run(90, 100, true).AffectedCount)

	done := run(90, 180, false)
	assert.Equal(t, preview.AffectedCount, done.AffectedCount)
	var left []string
	rows, err := conn.Query(`SELECT id FROM contacts ORDER BY id`)
	require.NoError(t, err)
	for rows.Next() {
		var id string
		require.NoError(t, rows.Scan(&id))
		left = append(left, id)
	}
	require.NoError(t, rows.Close())
	assert.Equal(t, []string{"campaign_click", "campaign_open", "click_120", "newcomer", "outreach_click", "sequence_open", "transactional_open"}, left)

	_, err = NewHousekeepingInactiveLogic(asAdmin, &svc.ServiceContext{DB: store}).HousekeepingInactive(&types.HousekeepingInactiveRequest{NoOpensDays: -1, NoClicksDays: 10, DryRun: true})
	assert.Error(t, err)
}
