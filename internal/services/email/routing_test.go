package email

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

// The kind decides the route, and the routes never cross: outreach (cold
// email) leaves only through its brand's own mailbox and can never reach SES
// or the platform SMTP sender; transactional and marketing mail never use a
// mailbox or send from an outreach domain.
func TestKindDecidesTheRoute(t *testing.T) {
	s, store := newTestService(t)
	ctx := context.Background()
	exec := func(q string) {
		t.Helper()
		if _, err := store.GetDB().ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO domain_identities (id, org_id, domain, purpose) VALUES ('d-out', 'org-a', 'getexample.com', 'outreach')`)
	exec(`INSERT INTO mailboxes (id, org_id, domain_id, email, from_name, smtp_host, smtp_port, smtp_username, smtp_password_enc, imap_host, imap_port, imap_username, imap_password_enc)
	      VALUES ('mb-a', 'org-a', 'd-out', 'sam@getexample.com', 'Sam', '127.0.0.1', 1, 'sam', x'00', '127.0.0.1', 1, 'sam', x'00')`)

	// Outreach without a mailbox, or with another brand's, is refused before
	// any provider is looked up.
	for _, m := range []Message{
		{Kind: KindOutreach, OrgID: "org-a", To: "c@x.example", Subject: "s", Text: "b"},
		{Kind: KindOutreach, OrgID: "org-b", MailboxID: "mb-a", To: "c@x.example", Subject: "s", Text: "b"},
	} {
		if err := s.Send(ctx, m); !errors.Is(err, ErrInvalidMessage) {
			t.Errorf("%+v: want ErrInvalidMessage, got %v", m, err)
		}
	}

	// Outreach from a real mailbox goes to the mailbox's own SMTP login (here
	// a closed port), never to SES or the platform sender, and says so.
	err := s.Send(ctx, Message{Kind: KindOutreach, OrgID: "org-a", MailboxID: "mb-a", To: "c@x.example", Subject: "s", Text: "b"})
	if err == nil || errors.Is(err, ErrInvalidMessage) {
		t.Fatalf("want a mailbox delivery error, got %v", err)
	}

	// Nothing but outreach may name a mailbox.
	if err := s.Send(ctx, Message{Kind: KindMarketing, OrgID: "org-a", MailboxID: "mb-a", To: "c@x.example", Subject: "s", HTML: "b"}); !errors.Is(err, ErrInvalidMessage) {
		t.Errorf("marketing through a mailbox: want ErrInvalidMessage, got %v", err)
	}

	// A brand's transactional and marketing mail can't send from its
	// outreach domain: that domain is for mailboxes only.
	if err := CheckSender(ctx, store, "org-a", "hello@getexample.com"); err == nil {
		t.Error("transactional mail from an outreach domain must be refused")
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
