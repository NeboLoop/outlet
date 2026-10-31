package campaigns

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// "Send now" used to set the status to sending, which the scheduler never
// picks up. It now schedules the campaign for now, and the scheduler sends it.
func TestSendCampaignNow_QueuesThroughTheScheduler(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn))
	store := db.NewStore(conn)
	s := &svc.ServiceContext{DB: store}
	ctx := context.Background()

	_, err = store.CreateUser(ctx, db.CreateUserParams{ID: "root", Email: "root@example.com", PasswordHash: "x", Name: "R", Role: "super_admin", Status: "active", EmailVerified: int64(1)})
	require.NoError(t, err)
	_, err = store.CreateOrganization(ctx, db.CreateOrganizationParams{ID: "alpha-id", Name: "alpha", Slug: "alpha", ApiKey: "k"})
	require.NoError(t, err)
	for _, id := range []string{"now", "later"} {
		_, err = store.CreateCampaign(ctx, db.CreateCampaignParams{
			ID: id, OrgID: "alpha-id", Name: id, Subject: "Hi", HtmlBody: "<p>Hi</p>", Status: sql.NullString{String: "draft", Valid: true},
		})
		require.NoError(t, err)
	}
	user, err := access.Active(ctx, store, "root")
	require.NoError(t, err)
	asRoot := access.WithBrand(access.WithUser(ctx, user), "alpha-id")

	info, err := NewSendCampaignNowLogic(asRoot, s).SendCampaignNow(&types.SendCampaignNowRequest{Id: "now"})
	require.NoError(t, err)
	assert.Equal(t, "scheduled", info.Status)

	// A campaign scheduled for later today (RFC 3339, as the schedule action
	// stores it) used to compare as text against datetime('now') and wait
	// for the next day. Times are compared as times now.
	_, err = NewScheduleCampaignLogic(asRoot, s).ScheduleCampaign(&types.ScheduleCampaignRequest{
		Id: "later", ScheduledAt: time.Now().UTC().Add(-time.Minute).Format(time.RFC3339),
	})
	require.NoError(t, err)

	due, err := store.GetScheduledCampaigns(ctx)
	require.NoError(t, err)
	ids := []string{}
	for _, c := range due {
		ids = append(ids, c.ID)
	}
	assert.ElementsMatch(t, []string{"now", "later"}, ids)

	// Sending twice is refused: it's no longer a draft.
	_, err = NewSendCampaignNowLogic(asRoot, s).SendCampaignNow(&types.SendCampaignNowRequest{Id: "now"})
	require.Error(t, err)
	assert.Equal(t, "Only a draft campaign can be sent.", err.Error())
}
