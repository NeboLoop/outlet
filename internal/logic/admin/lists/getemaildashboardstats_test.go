package lists

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

// The dashboard counts every kind of email the selected brand sent, and a
// brand member may read it.
func TestEmailDashboardStats_CountsTheBrandsEmail(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	store := db.NewStore(conn)
	ctx := context.Background()

	for _, q := range []string{
		`INSERT INTO users (id, email, password_hash, name, role, status) VALUES ('mike', 'mike@example.com', 'x', 'Mike', 'agent', 'active')`,
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('org1', 'One', 'one', 'k1'), ('org2', 'Two', 'two', 'k2')`,
		`INSERT INTO org_users (org_id, user_id, role) VALUES ('org1', 'mike', 'member')`,
		`INSERT INTO contacts (id, org_id, name, email) VALUES ('c1', 'org1', 'A', 'a@example.com'), ('c2', 'org1', 'B', 'b@example.com'), ('x1', 'org2', 'X', 'x@example.com')`,
		// Campaign: 3 sent today, one opened and clicked; one never sent
		`INSERT INTO email_campaigns (id, org_id, name, subject, html_body) VALUES ('camp', 'org1', 'C', 'S', 'B'), ('other', 'org2', 'C', 'S', 'B')`,
		`INSERT INTO email_lists (id, public_id, org_id, name, slug) VALUES (1, 'p1', 'org1', 'L', 'l')`,
		`INSERT INTO list_subscribers (id, list_id, contact_id, status, unsubscribed_at) VALUES ('ls', 1, 'c2', 'unsubscribed', datetime('now'))`,
		`INSERT INTO contacts (id, org_id, name, email) VALUES ('c3', 'org1', 'C', 'c@example.com'), ('c4', 'org1', 'D', 'd@example.com')`,
		`INSERT INTO campaign_sends (id, campaign_id, contact_id, status, sent_at, opened_at, clicked_at) VALUES
			('s1', 'camp', 'c1', 'sent', datetime('now'), datetime('now'), datetime('now')),
			('s2', 'camp', 'c2', 'sent', datetime('now'), NULL, NULL),
			('s3', 'camp', 'c3', 'delivered', datetime('now'), NULL, NULL),
			('s4', 'camp', 'c4', 'pending', NULL, NULL, NULL),
			('sx', 'other', 'x1', 'sent', datetime('now'), datetime('now'), NULL)`,
		// A sequence email sent and opened, one sent 40 days ago
		`INSERT INTO email_queue (id, contact_id, scheduled_for, status, sent_at, opened_at) VALUES
			('q1', 'c1', datetime('now'), 'sent', datetime('now'), datetime('now')),
			('q2', 'c1', datetime('now'), 'sent', datetime('now', '-40 days'), NULL)`,
		// A transactional send
		`INSERT INTO transactional_emails (id, org_id, name, slug, subject, html_body) VALUES ('t', 'org1', 'T', 't', 'S', 'B')`,
		`INSERT INTO transactional_sends (id, template_id, org_id, to_email, status, sent_at) VALUES ('ts', 't', 'org1', 'buyer@example.com', 'sent', datetime('now'))`,
		`INSERT INTO email_bounce (id, email, email_lower, bounce_type) VALUES ('b1', 'B@example.com', 'b@example.com', 'Permanent'), ('b2', 'x@example.com', 'x@example.com', 'Permanent')`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err, q)
	}

	user, err := access.Active(ctx, store, "mike")
	require.NoError(t, err)
	asMember := access.WithBrand(access.WithUser(ctx, user), "org1")

	got, err := NewGetEmailDashboardStatsLogic(asMember, &svc.ServiceContext{DB: store}).GetEmailDashboardStats(&types.EmailDashboardStatsRequest{})
	require.NoError(t, err)
	assert.Equal(t, 5, got.TotalSent, "3 campaign sends, 1 sequence email, 1 transactional send")
	assert.Equal(t, 2, got.TotalOpened)
	assert.Equal(t, 1, got.TotalClicked)
	assert.Equal(t, 1, got.TotalBounced)
	assert.Equal(t, 1, got.TotalUnsubscribed)
	assert.InDelta(t, 0.4, got.OpenRate, 0.001)
	assert.InDelta(t, 0.2, got.ClickRate, 0.001)

	// Another brand's member is refused
	_, err = NewGetEmailDashboardStatsLogic(access.WithBrand(access.WithUser(ctx, user), "org2"), &svc.ServiceContext{DB: store}).GetEmailDashboardStats(&types.EmailDashboardStatsRequest{})
	assert.Error(t, err)
}

// The dashboard's newsletter numbers leave outreach out: its lists, its
// queued email, the bounces and complaints it records and its lists'
// opt-outs are Outbound's.
func TestEmailDashboardStats_LeavesOutreachOut(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	store := db.NewStore(conn)
	ctx := context.Background()

	for _, q := range []string{
		`INSERT INTO users (id, email, password_hash, name, role, status) VALUES ('mike', 'mike@example.com', 'x', 'Mike', 'agent', 'active')`,
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('org1', 'One', 'one', 'k1')`,
		`INSERT INTO org_users (org_id, user_id, role) VALUES ('org1', 'mike', 'member')`,
		`INSERT INTO contacts (id, org_id, name, email) VALUES ('n1', 'org1', 'N', 'n1@example.com'), ('n2', 'org1', 'N', 'n2@example.com'),
			('p1', 'org1', 'P', 'p1@example.com'), ('p2', 'org1', 'P', 'p2@example.com'), ('p3', 'org1', 'P', 'p3@example.com')`,
		// A newsletter list with two subscribers, one gone; an outreach list
		// of three prospects, one opted out.
		`INSERT INTO email_lists (id, public_id, org_id, name, slug) VALUES (1, 'p1', 'org1', 'Newsletter', 'newsletter')`,
		`INSERT INTO email_lists (id, public_id, org_id, name, slug, kind) VALUES (2, 'p2', 'org1', 'Utah dental', 'utah-dental', 'outreach')`,
		`INSERT INTO list_subscribers (id, list_id, contact_id, status, unsubscribed_at) VALUES
			('ls1', 1, 'n1', 'active', NULL), ('ls2', 1, 'n2', 'unsubscribed', datetime('now')),
			('lp1', 2, 'p1', 'active', NULL), ('lp2', 2, 'p2', 'active', NULL), ('lp3', 2, 'p3', 'unsubscribed', datetime('now'))`,
		// A newsletter sequence email, opened; two outreach emails, opened and clicked
		`INSERT INTO email_queue (id, contact_id, scheduled_for, status, sent_at, opened_at, clicked_at, kind) VALUES
			('q1', 'n1', datetime('now'), 'sent', datetime('now'), datetime('now'), NULL, 'marketing'),
			('o1', 'p1', datetime('now'), 'sent', datetime('now'), datetime('now'), datetime('now'), 'outreach'),
			('o2', 'p2', datetime('now'), 'sent', datetime('now'), datetime('now'), datetime('now'), 'outreach')`,
		// A newsletter bounce; an outreach bounce and complaint
		`INSERT INTO email_bounce (id, email, email_lower, bounce_type, bounce_subtype) VALUES
			('b1', 'n2@example.com', 'n2@example.com', 'Permanent', 'General'),
			('b2', 'p2@example.com', 'p2@example.com', 'Permanent', 'outreach')`,
		`INSERT INTO email_complaint (id, email, email_lower, complaint_type) VALUES ('m1', 'p1@example.com', 'p1@example.com', 'outreach')`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err, q)
	}

	user, err := access.Active(ctx, store, "mike")
	require.NoError(t, err)
	asMember := access.WithBrand(access.WithUser(ctx, user), "org1")
	sc := &svc.ServiceContext{DB: store}

	got, err := NewGetEmailDashboardStatsLogic(asMember, sc).GetEmailDashboardStats(&types.EmailDashboardStatsRequest{})
	require.NoError(t, err)
	assert.Equal(t, 1, got.TotalSent, "the newsletter email, not the two outreach emails")
	assert.Equal(t, 1, got.TotalOpened)
	assert.Equal(t, 0, got.TotalClicked)
	assert.InDelta(t, 1.0, got.OpenRate, 0.001)
	assert.InDelta(t, 0.0, got.ClickRate, 0.001)
	assert.Equal(t, 1, got.TotalBounced, "the outreach bounce is Outbound's")
	assert.Equal(t, 0, got.TotalComplaints, "the outreach complaint is Outbound's")
	assert.Equal(t, 1, got.TotalUnsubscribed, "the outreach opt-out is Outbound's")

	// The dashboard's lists are the newsletter lists; every list stays the
	// default for the Lists page.
	newsletter, err := NewListListsLogic(asMember, sc).ListLists(&types.ListListsRequest{Kind: "subscribers"})
	require.NoError(t, err)
	require.Len(t, newsletter.Lists, 1)
	assert.Equal(t, "Newsletter", newsletter.Lists[0].Name)
	assert.Equal(t, 1, newsletter.Lists[0].SubscriberCount)
	outreach, err := NewListListsLogic(asMember, sc).ListLists(&types.ListListsRequest{Kind: "outreach"})
	require.NoError(t, err)
	require.Len(t, outreach.Lists, 1)
	assert.Equal(t, "Utah dental", outreach.Lists[0].Name)
	every, err := NewListListsLogic(asMember, sc).ListLists(&types.ListListsRequest{})
	require.NoError(t, err)
	assert.Len(t, every.Lists, 2)
	_, err = NewListListsLogic(asMember, sc).ListLists(&types.ListListsRequest{Kind: "newsletter"})
	assert.Error(t, err)
}
