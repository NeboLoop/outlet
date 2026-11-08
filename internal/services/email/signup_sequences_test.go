package email

import (
	"context"
	"testing"
)

// Every active "On signup" autoresponder of a list starts when a subscription
// becomes active, each once per contact; an inactive one and another list's
// don't.
func TestSubscriptionActivated_StartsEverySignupSequence(t *testing.T) {
	sender, store := newTestService(t)
	seqs := NewSequenceServiceWithBaseURL(store, sender, "https://mail.example.com")
	ctx := context.Background()
	conn := store.GetDB()
	for _, q := range []string{
		`INSERT INTO email_lists (id, public_id, org_id, name, slug) VALUES (1, 'news', 'org-a', 'News', 'news'), (2, 'other', 'org-a', 'Other', 'other')`,
		`INSERT INTO contacts (id, org_id, name, email, email_verified) VALUES ('ana', 'org-a', 'Ana', 'ana@x.example', 1)`,
		`INSERT INTO email_sequences (id, org_id, list_id, slug, name, trigger_event, is_active) VALUES
			('welcome', 'org-a', 1, 'welcome', 'Welcome', 'signup', 1),
			('course', 'org-a', 1, 'course', 'Course', 'signup', 1),
			('paused', 'org-a', 1, 'paused', 'Paused', 'signup', 0),
			('elsewhere', 'org-a', 2, 'elsewhere', 'Elsewhere', 'signup', 1)`,
		`INSERT INTO email_templates (id, org_id, sequence_id, position, subject, html_body) VALUES
			('w1', 'org-a', 'welcome', 1, 'S', 'B'),
			('c1', 'org-a', 'course', 1, 'S', 'B'),
			('p1', 'org-a', 'paused', 1, 'S', 'B'),
			('e1', 'org-a', 'elsewhere', 1, 'S', 'B')`,
	} {
		if _, err := conn.Exec(q); err != nil {
			t.Fatal(q, err)
		}
	}

	for range 2 {
		if err := seqs.SubscriptionActivated(ctx, "ana", 1); err != nil {
			t.Fatal(err)
		}
	}
	for template, want := range map[string]int{"w1": 1, "c1": 1, "p1": 0, "e1": 0} {
		var n int
		if err := conn.QueryRow(`SELECT COUNT(*) FROM email_queue WHERE contact_id = 'ana' AND template_id = ?`, template).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != want {
			t.Errorf("template %s: %d queued, want %d", template, n, want)
		}
	}
}
