package workers

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/logic/admin/campaigns"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// A campaign made in the admin UI is sent: the scheduler reads the lists the
// UI stored, queues one send per subscriber, and queues them once. A list of
// another brand never gives recipients, even if a campaign names it.
func TestProcessCampaign_SendsAUICampaignOnceToItsOwnBrand(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	store := db.NewStore(conn)
	ctx := context.Background()

	_, err = store.CreateUser(ctx, db.CreateUserParams{ID: "root", Email: "root@example.com", PasswordHash: "x", Name: "R", Role: "super_admin", Status: "active", EmailVerified: int64(1)})
	require.NoError(t, err)
	lists := map[string]int64{}
	for _, b := range []string{"alpha", "bravo"} {
		_, err := store.CreateOrganization(ctx, db.CreateOrganizationParams{ID: b + "-id", Name: b, Slug: b, ApiKey: "key-" + b})
		require.NoError(t, err)
		l, err := store.CreateEmailList(ctx, db.CreateEmailListParams{PublicID: "p-" + b, OrgID: b + "-id", Name: b, Slug: b})
		require.NoError(t, err)
		lists[b] = l.ID
	}
	subscribe := func(brand, contactID string) {
		_, err := store.CreateContact(ctx, db.CreateContactParams{ID: contactID, OrgID: sql.NullString{String: brand + "-id", Valid: true}, Name: contactID, Email: contactID + "@example.com", Status: "active"})
		require.NoError(t, err)
		_, err = store.SubscribeToList(ctx, db.SubscribeToListParams{ID: "s-" + contactID, ListID: lists[brand], ContactID: contactID})
		require.NoError(t, err)
	}
	subscribe("alpha", "a1")
	subscribe("alpha", "a2")
	subscribe("bravo", "b1")

	user, err := access.Active(ctx, store, "root")
	require.NoError(t, err)
	asRoot := access.WithBrand(access.WithUser(ctx, user), "alpha-id")
	s := &svc.ServiceContext{DB: store}

	made, err := campaigns.NewCreateCampaignLogic(asRoot, s).CreateCampaign(&types.CreateCampaignRequest{
		Name: "News", Subject: "Hi", HtmlBody: "<p>Hi</p>", ListIds: []string{strconv.FormatInt(lists["alpha"], 10)},
	})
	require.NoError(t, err)
	_, err = campaigns.NewSendCampaignNowLogic(asRoot, s).SendCampaignNow(&types.SendCampaignNowRequest{Id: made.Id})
	require.NoError(t, err)

	// A campaign that names bravo's list, written straight to the database.
	_, err = store.CreateCampaign(ctx, db.CreateCampaignParams{
		ID: "stray", OrgID: "alpha-id", Name: "stray", Subject: "Hi", HtmlBody: "<p>Hi</p>",
		ListIds: sql.NullString{String: db.EncodeListIDs([]int64{lists["bravo"]}), Valid: true},
		Status:  sql.NullString{String: "scheduled", Valid: true}, ScheduledAt: sql.NullString{String: "2020-01-01 00:00:00", Valid: true},
	})
	require.NoError(t, err)

	// A campaign from a domain alpha doesn't hold, written straight to the
	// database.
	_, err = store.CreateCampaign(ctx, db.CreateCampaignParams{
		ID: "spoof", OrgID: "alpha-id", Name: "spoof", Subject: "Hi", HtmlBody: "<p>Hi</p>",
		FromEmail: sql.NullString{String: "security@neboai.com", Valid: true},
		ListIds:   sql.NullString{String: db.EncodeListIDs([]int64{lists["alpha"]}), Valid: true},
		Status:    sql.NullString{String: "scheduled", Valid: true}, ScheduledAt: sql.NullString{String: "2020-01-01 00:00:00", Valid: true},
	})
	require.NoError(t, err)

	due, err := store.GetScheduledCampaigns(ctx)
	require.NoError(t, err)
	require.Len(t, due, 3)

	sched := NewCampaignScheduler(store, nil, DefaultCampaignSchedulerConfig())
	for _, c := range due {
		require.NoError(t, sched.processCampaign(c))
	}
	// Running a campaign again queues no second send.
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
	assert.Equal(t, []string{"a1", "a2"}, sends(made.Id))
	assert.Empty(t, sends("stray"), "another brand's list gives no recipients")
	assert.Empty(t, sends("spoof"), "a campaign sends only from its brand's domains")
	spoof, err := store.GetCampaign(ctx, db.GetCampaignParams{ID: "spoof", OrgID: "alpha-id"})
	require.NoError(t, err)
	assert.Equal(t, "paused", spoof.Status.String)

	// The workers pick the queued sends up.
	require.NoError(t, store.ClaimPendingCampaignSends(ctx, db.ClaimPendingCampaignSendsParams{ClaimToken: sql.NullString{String: "t1", Valid: true}, LimitCount: 100}))
	pending, err := store.GetPendingCampaignSends(ctx, sql.NullString{String: "t1", Valid: true})
	require.NoError(t, err)
	assert.Len(t, pending, 2)

	got, err := campaigns.NewGetCampaignLogic(asRoot, s).GetCampaign(&types.GetCampaignRequest{Id: made.Id})
	require.NoError(t, err)
	assert.Equal(t, "sending", got.Status)
	assert.Equal(t, []string{strconv.FormatInt(lists["alpha"], 10)}, got.ListIds)
}

// Creating or changing a campaign refuses a list of another brand.
func TestCampaignLists_MustBeTheBrandsOwn(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	store := db.NewStore(conn)
	ctx := context.Background()

	_, err = store.CreateUser(ctx, db.CreateUserParams{ID: "mike", Email: "mike@example.com", PasswordHash: "x", Name: "M", Role: "agent", Status: "active", EmailVerified: int64(1)})
	require.NoError(t, err)
	lists := map[string]string{}
	for _, b := range []string{"alpha", "bravo"} {
		_, err := store.CreateOrganization(ctx, db.CreateOrganizationParams{ID: b + "-id", Name: b, Slug: b, ApiKey: "key-" + b})
		require.NoError(t, err)
		l, err := store.CreateEmailList(ctx, db.CreateEmailListParams{PublicID: "p-" + b, OrgID: b + "-id", Name: b, Slug: b})
		require.NoError(t, err)
		lists[b] = strconv.FormatInt(l.ID, 10)
	}
	require.NoError(t, store.AddUserToOrganization(ctx, db.AddUserToOrganizationParams{OrgID: "alpha-id", UserID: "mike", Role: sql.NullString{String: "member", Valid: true}}))
	user, err := access.Active(ctx, store, "mike")
	require.NoError(t, err)
	asMike := access.WithBrand(access.WithUser(ctx, user), "alpha-id")
	s := &svc.ServiceContext{DB: store}

	_, err = campaigns.NewCreateCampaignLogic(asMike, s).CreateCampaign(&types.CreateCampaignRequest{
		Name: "x", Subject: "x", HtmlBody: "x", ListIds: []string{lists["bravo"]},
	})
	require.Error(t, err)
	_, err = campaigns.NewCreateCampaignLogic(asMike, s).CreateCampaign(&types.CreateCampaignRequest{
		Name: "x", Subject: "x", HtmlBody: "x", ListIds: []string{lists["alpha"]}, ExcludeListIds: []string{lists["bravo"]},
	})
	require.Error(t, err)

	made, err := campaigns.NewCreateCampaignLogic(asMike, s).CreateCampaign(&types.CreateCampaignRequest{
		Name: "x", Subject: "x", HtmlBody: "x", ListIds: []string{lists["alpha"]},
	})
	require.NoError(t, err)
	_, err = campaigns.NewUpdateCampaignLogic(asMike, s).UpdateCampaign(&types.UpdateCampaignRequest{Id: made.Id, ListIds: []string{lists["bravo"]}})
	require.Error(t, err)
}

// A send held back by its pathway's limit waits, pending and unclaimed,
// until the limit lifts; a retry held back returns to pending the same way.
// Other errors are not deferred.
func TestDeferCampaignSend_WaitsForTheLimit(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	store := db.NewStore(conn)
	ctx := context.Background()
	for _, q := range []string{
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('o', 'O', 'o', 'k')`,
		`INSERT INTO contacts (id, org_id, name, email) VALUES ('c1', 'o', 'C', 'c1@example.com'), ('c2', 'o', 'C', 'c2@example.com')`,
		`INSERT INTO email_campaigns (id, org_id, name, subject, html_body, status) VALUES ('camp', 'o', 'n', 's', 'b', 'sending')`,
		`INSERT INTO campaign_sends (id, campaign_id, contact_id, status) VALUES ('s1', 'camp', 'c1', 'pending'), ('s2', 'camp', 'c2', 'failed')`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err)
	}
	claim := func(token string) int {
		require.NoError(t, store.ClaimPendingCampaignSends(ctx, db.ClaimPendingCampaignSendsParams{ClaimToken: sql.NullString{String: token, Valid: true}, LimitCount: 100}))
		rows, err := store.GetPendingCampaignSends(ctx, sql.NullString{String: token, Valid: true})
		require.NoError(t, err)
		return len(rows)
	}
	require.Equal(t, 1, claim("t1"))

	assert.False(t, deferCampaignSend(ctx, store, "s1", errors.New("SES said no")), "a plain failure isn't deferred")
	limit := &email.DeliveryError{Temporary: true, RetryAt: time.Now().Add(time.Hour), Err: email.ErrOverLimit}
	assert.True(t, deferCampaignSend(ctx, store, "s1", limit))
	assert.True(t, deferCampaignSend(ctx, store, "s2", limit))
	assert.Equal(t, 0, claim("t2"), "both wait for the limit")

	var status string
	require.NoError(t, conn.QueryRow(`SELECT status FROM campaign_sends WHERE id = 's2'`).Scan(&status))
	assert.Equal(t, "pending", status)

	_, err = conn.Exec(`UPDATE campaign_sends SET send_after = datetime('now', '-1 minute')`)
	require.NoError(t, err)
	assert.Equal(t, 2, claim("t3"), "once the limit lifts, both go")
}
