package email

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
)

// The kind decides the route, and the routes never cross: each kind goes
// only through the brand's pathway carrying it; a kind no pathway carries
// is refused, never sent another way; a mailbox is named only by outreach.
func TestKindDecidesTheRoute(t *testing.T) {
	s, store := newTestService(t)
	ctx := context.Background()
	exec := func(q string) {
		t.Helper()
		if _, err := store.GetDB().ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}

	// No pathway carries org-a's outreach yet: refused, and not sent through
	// the platform pathway that carries its other mail.
	err := s.Send(ctx, Message{Kind: KindOutreach, OrgID: "org-a", MailboxID: "mb-a", To: "c@x.example", Subject: "s", Text: "b"})
	if !errors.Is(err, ErrInvalidMessage) || !strings.Contains(err.Error(), "no sending pathway for outreach mail") {
		t.Fatalf("outreach with no pathway: want a refusal, got %v", err)
	}

	exec(`INSERT INTO sending_pathways (id, org_id, name, provider, carries_outreach) VALUES ('pool-a', 'org-a', 'Mailboxes', 'mailboxes', 1)`)
	exec(`INSERT INTO domain_identities (id, org_id, domain, purpose, pathway_id) VALUES ('d-out', 'org-a', 'getexample.com', 'outreach', 'pool-a')`)
	exec(`INSERT INTO mailboxes (id, org_id, domain_id, email, from_name, smtp_host, smtp_port, smtp_username, smtp_password_enc, imap_host, imap_port, imap_username, imap_password_enc)
	      VALUES ('mb-a', 'org-a', 'd-out', 'sam@getexample.com', 'Sam', '127.0.0.1', 1, 'sam', x'00', '127.0.0.1', 1, 'sam', x'00')`)

	// Outreach without a mailbox, or with another brand's, is refused before
	// anything is sent.
	for _, m := range []Message{
		{Kind: KindOutreach, OrgID: "org-a", To: "c@x.example", Subject: "s", Text: "b"},
		{Kind: KindOutreach, OrgID: "org-b", MailboxID: "mb-a", To: "c@x.example", Subject: "s", Text: "b"},
		{Kind: KindOutreach, MailboxID: "mb-a", To: "c@x.example", Subject: "s", Text: "b"},
	} {
		if err := s.Send(ctx, m); !errors.Is(err, ErrInvalidMessage) {
			t.Errorf("%+v: want ErrInvalidMessage, got %v", m, err)
		}
	}

	// Nothing but outreach may name a mailbox.
	if err := s.Send(ctx, Message{Kind: KindMarketing, OrgID: "org-a", MailboxID: "mb-a", To: "c@x.example", Subject: "s", HTML: "b"}); !errors.Is(err, ErrInvalidMessage) {
		t.Errorf("marketing naming a mailbox: want ErrInvalidMessage, got %v", err)
	}

	// A brand's transactional and marketing mail can't send from its
	// outreach domain: that domain belongs to the mailbox pool.
	if err := CheckSender(ctx, store, "org-a", KindTransactional, "hello@getexample.com"); err == nil {
		t.Error("transactional mail from an outreach domain must be refused")
	}

	// Marketing goes through the pathway carrying it. Move marketing to a
	// pathway of its own, and the platform pathway no longer takes it.
	exec(`UPDATE sending_pathways SET carries_marketing = 0 WHERE org_id = 'org-a'`)
	if err := s.Send(ctx, Message{Kind: KindMarketing, OrgID: "org-a", To: "c@x.example", Subject: "s", HTML: "b"}); !errors.Is(err, ErrInvalidMessage) || !strings.Contains(err.Error(), "marketing") {
		t.Errorf("marketing with no pathway: want a refusal, got %v", err)
	}
	// Transactional still has its own and reaches delivery.
	if err := s.Send(ctx, Message{OrgID: "org-a", To: "c@x.example", Subject: "s", HTML: "b"}); !notConfigured(err) {
		t.Errorf("transactional: want it to reach the platform, got %v", err)
	}

	// The marketing worker never claims an outreach email.
	exec(`INSERT INTO contacts (id, org_id, name, email) VALUES ('c1', 'org-a', 'C', 'c@x.example')`)
	exec(`INSERT INTO email_lists (id, public_id, org_id, name, slug) VALUES (1, 'p1', 'org-a', 'L', 'l')`)
	exec(`INSERT INTO email_sequences (id, org_id, list_id, slug, name, trigger_event) VALUES ('seq', 'org-a', 1, 's', 'S', 'manual')`)
	exec(`INSERT INTO email_templates (id, org_id, sequence_id, position, subject, html_body) VALUES ('t1', 'org-a', 'seq', 1, 's', 'b')`)
	exec(`INSERT INTO email_queue (id, contact_id, template_id, scheduled_for, kind, mailbox_id, tracking_token) VALUES ('q1', 'c1', 't1', '2000-01-01T00:00:00Z', 'outreach', 'mb-a', 'tok1')`)
	rows, err := claimDueEmails(ctx, store, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Errorf("the marketing worker claimed %d outreach emails", len(rows))
	}
	var status sql.NullString
	if err := store.GetDB().QueryRowContext(ctx, `SELECT status FROM email_queue WHERE id = 'q1'`).Scan(&status); err != nil || status.String != "pending" {
		t.Errorf("outreach row status = %v %v", status, err)
	}
}
