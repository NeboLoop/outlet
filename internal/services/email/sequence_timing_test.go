package email

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
)

// A sequence made in the app (or over MCP) sends its first email with a 0
// hour delay right away, as the app says ("Sends immediately"), not at a
// fixed hour the next morning.
func TestNewSequenceSendsByItsDelay(t *testing.T) {
	sender, store := newTestService(t)
	seqs := NewSequenceServiceWithBaseURL(store, sender, "https://mail.example.com")
	ctx := context.Background()
	conn := store.GetDB()

	seq, err := store.CreateSequence(ctx, db.CreateSequenceParams{
		ID: "welcome", OrgID: sql.NullString{String: "org-a", Valid: true}, Slug: "welcome", Name: "Welcome",
		TriggerEvent: "manual", IsActive: sql.NullInt64{Int64: 1, Valid: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if seq.SendHour.Valid {
		t.Fatalf("a new sequence has send hour %d; want none", seq.SendHour.Int64)
	}
	for _, q := range []string{
		`INSERT INTO contacts (id, org_id, name, email, email_verified) VALUES ('ana', 'org-a', 'Ana', 'ana@x.example', 1)`,
		`INSERT INTO email_templates (id, org_id, sequence_id, position, subject, html_body, delay_hours) VALUES ('w1', 'org-a', 'welcome', 1, 'S', 'B', 0)`,
	} {
		if _, err := conn.Exec(q); err != nil {
			t.Fatal(q, err)
		}
	}
	if err := seqs.StartSequenceByID(ctx, "ana", "welcome"); err != nil {
		t.Fatal(err)
	}
	var scheduled string
	if err := conn.QueryRow(`SELECT scheduled_for FROM email_queue WHERE contact_id = 'ana'`).Scan(&scheduled); err != nil {
		t.Fatal(err)
	}
	at, err := time.Parse(time.RFC3339, scheduled)
	if err != nil {
		t.Fatal(err)
	}
	if d := time.Until(at); d > time.Minute {
		t.Fatalf("first email scheduled %s from now; want right away", d)
	}
}
