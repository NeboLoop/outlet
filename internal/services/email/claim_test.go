package email

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/outlet-sh/outlet/internal/db"
)

func seedSequence(t *testing.T, store *db.Store) {
	t.Helper()
	exec := func(q string, args ...any) {
		if _, err := store.GetDB().Exec(q, args...); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	exec(`INSERT INTO email_lists (id, public_id, org_id, name, slug) VALUES (1, 'pl1', 'org-a', 'L', 'l')`)
	exec(`INSERT INTO email_sequences (id, org_id, list_id, slug, name, trigger_event) VALUES ('seq', 'org-a', 1, 'welcome', 'Welcome', 'subscribe')`)
	exec(`INSERT INTO email_templates (id, org_id, sequence_id, position, subject, html_body) VALUES ('t1', 'org-a', 'seq', 1, 's', 'b')`)
	past := time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
	for i := range 10 {
		c := fmt.Sprintf("c%d", i)
		exec(`INSERT INTO contacts (id, org_id, name, email) VALUES (?, 'org-a', ?, ?)`, c, c, c+"@x.example")
		exec(`INSERT INTO email_queue (id, contact_id, template_id, scheduled_for, tracking_token) VALUES (?, ?, 't1', ?, ?)`,
			"q"+c, c, past, uuid.NewString())
	}
	// Not sendable: paused sequence, left the list, unsubscribed contact
	exec(`INSERT INTO contact_sequence_state (id, contact_id, sequence_id, is_active) VALUES ('s7', 'c7', 'seq', 0)`)
	exec(`INSERT INTO list_subscribers (list_id, contact_id, status) VALUES (1, 'c8', 'unsubscribed')`)
	exec(`UPDATE contacts SET unsubscribed_at = datetime('now') WHERE id = 'c9'`)
}

func TestClaimDueEmailsNeverHandsOutARowTwice(t *testing.T) {
	_, store := newTestService(t)
	seedSequence(t, store)
	ctx := context.Background()

	first, err := claimDueEmails(ctx, store, 4)
	if err != nil {
		t.Fatal(err)
	}
	second, err := claimDueEmails(ctx, store, 100)
	if err != nil {
		t.Fatal(err)
	}
	third, _ := claimDueEmails(ctx, store, 100)

	seen := map[string]bool{}
	for _, row := range append(first, second...) {
		if seen[row.ID] {
			t.Errorf("%s claimed twice", row.ID)
		}
		seen[row.ID] = true
		if row.OrgID != "org-a" {
			t.Errorf("%s: org = %q", row.ID, row.OrgID)
		}
	}
	if len(first) != 4 || len(seen) != 7 || len(third) != 0 {
		t.Errorf("claimed %d then %d then %d; want 4, 3, 0 (7 sendable of 10)", len(first), len(second), len(third))
	}
	for _, skipped := range []string{"qc7", "qc8", "qc9"} {
		if seen[skipped] {
			t.Errorf("%s should not be sent", skipped)
		}
	}

	// An abandoned claim (worker crashed mid-send) becomes claimable again
	if _, err := store.GetDB().Exec(`UPDATE email_queue SET claimed_at = datetime('now', '-20 minutes') WHERE id = 'qc0'`); err != nil {
		t.Fatal(err)
	}
	again, _ := claimDueEmails(ctx, store, 100)
	if len(again) != 1 || again[0].ID != "qc0" {
		t.Errorf("reclaimed %v", again)
	}
}

func TestScheduledCampaignsCompareISOTimes(t *testing.T) {
	_, store := newTestService(t)
	past := time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	for id, at := range map[string]string{"due": past, "later": future} {
		if _, err := store.GetDB().Exec(`INSERT INTO email_campaigns (id, org_id, name, subject, html_body, status, scheduled_at) VALUES (?, 'org-a', ?, 's', 'b', 'scheduled', ?)`, id, id, at); err != nil {
			t.Fatal(err)
		}
	}
	got, err := store.GetScheduledCampaigns(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "due" {
		t.Errorf("due campaigns = %v", got)
	}

	c, err := store.SendCampaignNow(context.Background(), db.SendCampaignNowParams{ID: "later", OrgID: "org-a"})
	if err != nil || c.Status.String != "scheduled" {
		t.Fatalf("send now: %v %v", c.Status, err)
	}
	got, _ = store.GetScheduledCampaigns(context.Background())
	if len(got) != 2 {
		t.Errorf("after send now, due campaigns = %d, want 2", len(got))
	}
	if _, err := store.SendCampaignNow(context.Background(), db.SendCampaignNowParams{ID: "later", OrgID: "org-b"}); err != sql.ErrNoRows {
		t.Errorf("another brand must not send it: %v", err)
	}
}
