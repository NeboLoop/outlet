package email

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"

	_ "modernc.org/sqlite"
)

// newTestService returns a Service on a migrated temporary database with two
// brands, and no SES or SMTP configured, so a send that passes every check
// stops at "email not configured".
func newTestService(t *testing.T) (*Service, *db.Store) {
	t.Helper()
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := migrations.Run(conn); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`INSERT INTO organizations (id, name, slug, api_key, from_email) VALUES ('org-a', 'A', 'a', 'key-a', 'hello@a.example')`,
		`INSERT INTO organizations (id, name, slug, api_key, from_email) VALUES ('org-b', 'B', 'b', 'key-b', 'hello@b.example')`,
		`INSERT INTO domain_identities (id, org_id, domain) VALUES ('d-b', 'org-b', 'b.example')`,
	} {
		if _, err := conn.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	store := db.NewStore(conn)
	return NewService(store, nil), store
}

func notConfigured(err error) bool {
	return err != nil && strings.Contains(err.Error(), "email not configured")
}

func TestSendRefusesSuppressedRecipients(t *testing.T) {
	s, store := newTestService(t)
	ctx := context.Background()
	exec := func(q string) {
		if _, err := store.GetDB().Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO email_bounce (id, email, email_lower, bounce_type) VALUES ('1', 'Hard@X.example', 'hard@x.example', 'Permanent')`)
	exec(`INSERT INTO email_bounce (id, email, email_lower, bounce_type) VALUES ('2', 'soft@x.example', 'soft@x.example', 'Transient')`)
	exec(`INSERT INTO email_complaint (id, email, email_lower) VALUES ('3', 'angry@x.example', 'angry@x.example')`)
	exec(`INSERT INTO suppression_list (org_id, email, email_lower) VALUES ('org-a', 'gone@x.example', 'gone@x.example')`)
	exec(`INSERT INTO blocked_domains (org_id, domain) VALUES ('org-a', 'blocked.example')`)

	for to, wantSuppressed := range map[string]bool{
		"hard@x.example":         true,
		"angry@x.example":        true,
		"gone@x.example":         true,
		"anyone@blocked.example": true,
		"soft@x.example":         false, // transient bounces do not block
		"fine@x.example":         false,
	} {
		err := s.Send(ctx, Message{OrgID: "org-a", To: to, Subject: "s", HTML: "x"})
		if wantSuppressed && !errors.Is(err, ErrSuppressed) {
			t.Errorf("%s: want ErrSuppressed, got %v", to, err)
		}
		if !wantSuppressed && !notConfigured(err) {
			t.Errorf("%s: want it to reach delivery, got %v", to, err)
		}
	}

	// The suppression list and blocked domains belong to one brand only
	if err := s.Send(ctx, Message{OrgID: "org-b", To: "gone@x.example", Subject: "s", HTML: "x"}); !notConfigured(err) {
		t.Errorf("org-b should still reach gone@x.example, got %v", err)
	}
}

func TestSendRefusesAnotherBrandsDomain(t *testing.T) {
	s, _ := newTestService(t)
	err := s.Send(context.Background(), Message{OrgID: "org-a", FromEmail: "billing@b.example", To: "c@x.example", Subject: "s", HTML: "x"})
	if !errors.Is(err, ErrInvalidMessage) {
		t.Errorf("want ErrInvalidMessage, got %v", err)
	}
	// Brand B may use its own domain
	err = s.Send(context.Background(), Message{OrgID: "org-b", FromEmail: "billing@b.example", To: "c@x.example", Subject: "s", HTML: "x"})
	if !notConfigured(err) {
		t.Errorf("want it to reach delivery, got %v", err)
	}
}

func TestSendNeedsAFromAddress(t *testing.T) {
	s, _ := newTestService(t)
	if err := s.Send(context.Background(), Message{To: "c@x.example", Subject: "s", HTML: "x"}); !errors.Is(err, ErrInvalidMessage) {
		t.Errorf("platform mail with no From configured: want ErrInvalidMessage, got %v", err)
	}
}
