package tools

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/mcp/mcpctx"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// The campaign tool never points a newsletter campaign at an outreach list,
// the email tool never puts a sequence other than outreach on one, and
// list.list tells the kinds apart and filters by kind.
func TestTools_KeepOptInMailOffOutreachLists(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	for _, q := range []string{
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('org1', 'Brand', 'brand', 'k1')`,
		`INSERT INTO email_lists (id, public_id, org_id, name, slug) VALUES (1, 'news', 'org1', 'News', 'news')`,
		`INSERT INTO email_lists (id, public_id, org_id, name, slug, kind) VALUES (2, 'dental', 'org1', 'Utah dental', 'utah-dental', 'outreach')`,
		`INSERT INTO email_campaigns (id, org_id, name, subject, html_body, list_ids, status) VALUES ('stray', 'org1', 'stray', 'Hi', '<p>Hi</p>', '["2"]', 'draft')`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err, q)
	}
	store := db.NewStore(conn)
	brand, err := store.GetOrganizationByID(context.Background(), "org1")
	require.NoError(t, err)
	ctx := mcpctx.WithCallBrand(context.Background(), brand)
	toolCtx := mcpctx.NewToolContext(&svc.ServiceContext{DB: store, Sequences: email.NewSequenceServiceWithBaseURL(store, nil, "")}, brand, "req", "test")
	const refusal = `"Utah dental" is an outreach list`

	// campaign.create
	_, _, err = handleCampaignCreate(ctx, toolCtx, CampaignInput{Action: "create", Name: "N", Subject: "Hi", HTMLBody: "<p>Hi</p>", ListIDs: "1,2"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), refusal)
	_, out, err := handleCampaignCreate(ctx, toolCtx, CampaignInput{Action: "create", Name: "N", Subject: "Hi", HTMLBody: "<p>Hi</p>", ListIDs: "1"})
	require.NoError(t, err, "a newsletter list is fine")
	made := out.(CampaignCreateOutput).ID

	// campaign.update
	_, _, err = handleCampaignUpdate(ctx, toolCtx, CampaignInput{Action: "update", ID: made, ListIDs: "2"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), refusal)

	// campaign.schedule and campaign.send of one that names the outreach list
	_, _, err = handleCampaignSchedule(ctx, toolCtx, CampaignInput{Action: "schedule", ID: "stray", ScheduledAt: "2030-01-15T10:00:00Z"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), refusal)
	_, _, err = handleCampaignSend(ctx, toolCtx, CampaignInput{Action: "send", ID: "stray"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), refusal)
	var status string
	require.NoError(t, conn.QueryRow(`SELECT status FROM email_campaigns WHERE id = 'stray'`).Scan(&status))
	assert.Equal(t, "draft", status)

	_, _, err = handleCampaignSend(ctx, toolCtx, CampaignInput{Action: "send", ID: made})
	require.NoError(t, err, "a newsletter campaign still sends")

	// sequence.create: only an outreach sequence goes on an outreach list.
	_, _, err = handleSequenceCreate(ctx, toolCtx, EmailInput{Resource: "sequence", Action: "create", Name: "Welcome", ListID: "2"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), refusal)
	_, seqOut, err := handleSequenceCreate(ctx, toolCtx, EmailInput{Resource: "sequence", Action: "create", Name: "Welcome", ListID: "1"})
	require.NoError(t, err)
	_, cold, err := handleSequenceCreate(ctx, toolCtx, EmailInput{Resource: "sequence", Action: "create", Name: "Cold", ListID: "2", SequenceType: "outreach"})
	require.NoError(t, err, "an outreach sequence belongs on an outreach list")

	// sequence.update: an outreach sequence can't become an autoresponder.
	_, _, err = handleSequenceUpdate(ctx, toolCtx, EmailInput{Resource: "sequence", Action: "update", ID: cold.(SequenceCreateOutput).ID, SequenceType: "lifecycle"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), refusal)

	// entry_rule.create: an autoresponder never starts from an outreach list.
	_, _, err = handleEntryRuleCreate(ctx, toolCtx, EmailInput{Resource: "entry_rule", Action: "create", SequenceID: seqOut.(SequenceCreateOutput).ID, TriggerType: "list_join", SourceID: "2"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), refusal)
	_, _, err = handleEntryRuleCreate(ctx, toolCtx, EmailInput{Resource: "entry_rule", Action: "create", SequenceID: seqOut.(SequenceCreateOutput).ID, TriggerType: "list_join", SourceID: "1"})
	require.NoError(t, err)

	// list.list shows each list's kind and filters by it.
	kinds := func(kind string) map[string]string {
		_, out, err := handleListList(ctx, toolCtx, EmailInput{Resource: "list", Action: "list", Kind: kind})
		require.NoError(t, err)
		got := map[string]string{}
		for _, l := range out.(ListListOutput).Lists {
			got[l.Name] = l.Kind
		}
		return got
	}
	assert.Equal(t, map[string]string{"News": "subscribers", "Utah dental": "outreach"}, kinds(""))
	assert.Equal(t, map[string]string{"News": "subscribers"}, kinds("subscribers"))
	assert.Equal(t, map[string]string{"Utah dental": "outreach"}, kinds("outreach"))
	_, _, err = handleListList(ctx, toolCtx, EmailInput{Resource: "list", Action: "list", Kind: "newsletter"})
	require.Error(t, err)
	_, got, err := handleListGet(ctx, toolCtx, EmailInput{Resource: "list", Action: "get", ID: "2"})
	require.NoError(t, err)
	assert.Equal(t, "outreach", got.(ListGetOutput).Kind)
}
