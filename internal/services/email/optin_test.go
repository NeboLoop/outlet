package email

import (
	"context"
	"strings"
	"testing"
)

// An autoresponder never starts for an outreach list, whatever put it there:
// its own list being an outreach list, or an entry rule waiting on one.
// The same sequences start from a newsletter list.
func TestOptInSequencesNeverStartFromAnOutreachList(t *testing.T) {
	sender, store := newTestService(t)
	seqs := NewSequenceServiceWithBaseURL(store, sender, "https://mail.example.com")
	ctx := context.Background()
	conn := store.GetDB()

	for _, q := range []string{
		`INSERT INTO email_lists (id, public_id, org_id, name, slug) VALUES (1, 'news', 'org-a', 'News', 'news')`,
		`INSERT INTO email_lists (id, public_id, org_id, name, slug, kind) VALUES (2, 'dental', 'org-a', 'Utah dental', 'utah-dental', 'outreach')`,
		`INSERT INTO contacts (id, org_id, name, email, email_verified) VALUES ('ana', 'org-a', 'Ana', 'ana@x.example', 1)`,
		`INSERT INTO email_sequences (id, org_id, list_id, slug, name, trigger_event, sequence_type) VALUES
			('on-outreach', 'org-a', 2, 'o', 'O', 'signup', 'lifecycle'),
			('by-rule', 'org-a', 1, 'r', 'R', 'manual', 'lifecycle'),
			('on-news', 'org-a', 1, 'n', 'N', 'signup', 'lifecycle')`,
		`INSERT INTO email_templates (id, org_id, sequence_id, position, subject, html_body) VALUES
			('o1', 'org-a', 'on-outreach', 1, 'S', 'B'),
			('r1', 'org-a', 'by-rule', 1, 'S', 'B'),
			('n1', 'org-a', 'on-news', 1, 'S', 'B')`,
		`INSERT INTO sequence_entry_rules (id, sequence_id, trigger_type, source_id) VALUES ('rule', 'by-rule', 'list_join', '2')`,
	} {
		if _, err := conn.Exec(q); err != nil {
			t.Fatal(q, err)
		}
	}
	enrolled := func() int {
		var n int
		if err := conn.QueryRow(`SELECT COUNT(*) FROM contact_sequence_state WHERE contact_id = 'ana'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	err := seqs.SubscriptionActivated(ctx, "ana", 2)
	if err == nil || !strings.Contains(err.Error(), `"Utah dental" is an outreach list`) {
		t.Fatalf("activation on the outreach list: %v", err)
	}
	err = seqs.StartSequenceByID(ctx, "ana", "on-outreach")
	if err == nil || !strings.Contains(err.Error(), `"Utah dental" is an outreach list`) {
		t.Fatalf("starting an autoresponder of the outreach list: %v", err)
	}
	if n := enrolled(); n != 0 {
		t.Fatalf("enrolled in %d sequences from an outreach list", n)
	}

	if err := seqs.SubscriptionActivated(ctx, "ana", 1); err != nil {
		t.Fatal(err)
	}
	if n := enrolled(); n != 1 {
		t.Fatalf("a newsletter list's signup sequence: enrolled in %d", n)
	}
}
