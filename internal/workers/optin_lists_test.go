package workers

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/logic/admin/campaigns"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// A newsletter campaign never goes to an outreach list: creating, changing,
// scheduling and sending one through the API refuse it, and the scheduler
// pauses one that names one anyway and queues nobody. Newsletter lists still
// send.
func TestNewsletterCampaign_NeverGoesToAnOutreachList(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	for _, q := range []string{
		`INSERT INTO users (id, email, password_hash, name, role, status, email_verified) VALUES ('mike', 'mike@example.com', 'x', 'M', 'agent', 'active', 1)`,
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('alpha-id', 'alpha', 'alpha', 'k-a')`,
		`INSERT INTO org_users (org_id, user_id, role) VALUES ('alpha-id', 'mike', 'admin')`,
		`INSERT INTO email_lists (id, public_id, org_id, name, slug) VALUES (1, 'p1', 'alpha-id', 'News', 'news')`,
		`INSERT INTO email_lists (id, public_id, org_id, name, slug, kind) VALUES (2, 'p2', 'alpha-id', 'Utah dental', 'utah-dental', 'outreach')`,
		`INSERT INTO contacts (id, org_id, name, email, status) VALUES ('reader', 'alpha-id', 'R', 'reader@example.com', 'active')`,
		`INSERT INTO contacts (id, org_id, name, email, status) VALUES ('dentist', 'alpha-id', 'D', 'dentist@example.com', 'active')`,
		`INSERT INTO list_subscribers (id, list_id, contact_id, status) VALUES ('s1', 1, 'reader', 'active')`,
		// Active, so only the rule keeps a campaign from reaching them.
		`INSERT INTO list_subscribers (id, list_id, contact_id, status) VALUES ('s2', 2, 'dentist', 'active')`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err, q)
	}
	store := db.NewStore(conn)
	user, err := access.Active(context.Background(), store, "mike")
	require.NoError(t, err)
	ctx := access.WithBrand(access.WithUser(context.Background(), user), "alpha-id")
	s := &svc.ServiceContext{DB: store}
	const refusal = `"Utah dental" is an outreach list`

	// Create refuses the outreach list, alone or with a newsletter list.
	for _, ids := range [][]string{{"2"}, {"1", "2"}} {
		_, err = campaigns.NewCreateCampaignLogic(ctx, s).CreateCampaign(&types.CreateCampaignRequest{
			Name: "News", Subject: "Hi", HtmlBody: "<p>Hi</p>", ListIds: ids,
		})
		require.Error(t, err, ids)
		assert.Contains(t, err.Error(), refusal)
	}

	made, err := campaigns.NewCreateCampaignLogic(ctx, s).CreateCampaign(&types.CreateCampaignRequest{
		Name: "News", Subject: "Hi", HtmlBody: "<p>Hi</p>", ListIds: []string{"1"},
	})
	require.NoError(t, err, "a newsletter list is fine")

	// Update refuses it.
	_, err = campaigns.NewUpdateCampaignLogic(ctx, s).UpdateCampaign(&types.UpdateCampaignRequest{Id: made.Id, ListIds: []string{"2"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), refusal)
	// Excluding an outreach list sends it nothing, so that stays allowed.
	_, err = campaigns.NewUpdateCampaignLogic(ctx, s).UpdateCampaign(&types.UpdateCampaignRequest{Id: made.Id, ListIds: []string{"1"}, ExcludeListIds: []string{"2"}})
	require.NoError(t, err)

	// A campaign that names the outreach list, written straight to the
	// database: schedule and send now refuse it.
	_, err = store.CreateCampaign(context.Background(), db.CreateCampaignParams{
		ID: "stray", OrgID: "alpha-id", Name: "stray", Subject: "Hi", HtmlBody: "<p>Hi</p>",
		ListIds: sql.NullString{String: db.EncodeListIDs([]int64{1, 2}), Valid: true},
		Status:  sql.NullString{String: "draft", Valid: true},
	})
	require.NoError(t, err)
	_, err = campaigns.NewScheduleCampaignLogic(ctx, s).ScheduleCampaign(&types.ScheduleCampaignRequest{
		Id: "stray", ScheduledAt: time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), refusal)
	_, err = campaigns.NewSendCampaignNowLogic(ctx, s).SendCampaignNow(&types.SendCampaignNowRequest{Id: "stray"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), refusal)
	stray, err := store.GetCampaign(context.Background(), db.GetCampaignParams{ID: "stray", OrgID: "alpha-id"})
	require.NoError(t, err)
	assert.Equal(t, "draft", stray.Status.String, "refused: still a draft")

	// The newsletter campaign is sent.
	_, err = campaigns.NewSendCampaignNowLogic(ctx, s).SendCampaignNow(&types.SendCampaignNowRequest{Id: made.Id})
	require.NoError(t, err)

	// The stray one, scheduled straight in the database: the scheduler
	// pauses it and queues nobody.
	require.NoError(t, store.UpdateCampaignStatusByID(context.Background(), db.UpdateCampaignStatusByIDParams{ID: "stray", Status: sql.NullString{String: "scheduled", Valid: true}}))
	_, err = conn.Exec(`UPDATE email_campaigns SET scheduled_at = '2020-01-01 00:00:00' WHERE id = 'stray'`)
	require.NoError(t, err)

	due, err := store.GetScheduledCampaigns(context.Background())
	require.NoError(t, err)
	require.Len(t, due, 2)
	sched := NewCampaignScheduler(store, nil, DefaultCampaignSchedulerConfig())
	for _, c := range due {
		require.NoError(t, sched.processCampaign(c))
	}

	sends := func(campaignID string) []string {
		rows, err := conn.Query(`SELECT contact_id FROM campaign_sends WHERE campaign_id = ? ORDER BY contact_id`, campaignID)
		require.NoError(t, err)
		defer rows.Close()
		var ids []string
		for rows.Next() {
			var id string
			require.NoError(t, rows.Scan(&id))
			ids = append(ids, id)
		}
		return ids
	}
	assert.Equal(t, []string{"reader"}, sends(made.Id))
	assert.Empty(t, sends("stray"), "no send to anyone when a campaign names an outreach list")
	stray, err = store.GetCampaign(context.Background(), db.GetCampaignParams{ID: "stray", OrgID: "alpha-id"})
	require.NoError(t, err)
	assert.Equal(t, "paused", stray.Status.String)
	sent, err := store.GetCampaign(context.Background(), db.GetCampaignParams{ID: made.Id, OrgID: "alpha-id"})
	require.NoError(t, err)
	assert.Equal(t, "sending", sent.Status.String)
	assert.Equal(t, []int64{1}, db.DecodeListIDs(sent.ListIds.String))
}
