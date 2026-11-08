package email

import (
	"context"
	"database/sql"
	"slices"
	"testing"

	"github.com/outlet-sh/outlet/internal/db"
)

// Entry rules start their sequence at the moment they name: a list
// subscription becoming active, another sequence finished, a tag added. A
// rule of another brand never enrolls the contact.
func TestEntryRulesStartTheirSequences(t *testing.T) {
	sender, store := newTestService(t)
	seqs := NewSequenceServiceWithBaseURL(store, sender, "https://mail.example.com")
	store.OnTagAdded(seqs.TagAdded)
	ctx := context.Background()
	conn := store.GetDB()

	for _, q := range []string{
		`INSERT INTO email_lists (id, public_id, org_id, name, slug) VALUES (1, 'news', 'org-a', 'News', 'news')`,
		`INSERT INTO contacts (id, org_id, name, email, email_verified) VALUES ('ana', 'org-a', 'Ana', 'ana@x.example', 1)`,
		`INSERT INTO email_sequences (id, org_id, slug, name, trigger_event) VALUES
			('on-join', 'org-a', 'j', 'J', 'manual'),
			('first', 'org-a', 'f', 'F', 'manual'),
			('after-first', 'org-a', 'af', 'AF', 'manual'),
			('on-tag', 'org-a', 't', 'T', 'manual'),
			('other-brand', 'org-b', 'o', 'O', 'manual')`,
		`INSERT INTO email_templates (id, org_id, sequence_id, position, subject, html_body) VALUES
			('j1', 'org-a', 'on-join', 1, 'S', 'B'),
			('f1', 'org-a', 'first', 1, 'S', 'B'),
			('af1', 'org-a', 'after-first', 1, 'S', 'B'),
			('t1', 'org-a', 'on-tag', 1, 'S', 'B'),
			('o1', 'org-b', 'other-brand', 1, 'S', 'B')`,
		`INSERT INTO sequence_entry_rules (id, sequence_id, trigger_type, source_id) VALUES
			('r1', 'on-join', 'list_join', '1'),
			('r2', 'after-first', 'sequence_complete', 'first'),
			('r3', 'on-tag', 'tag_added', 'vip'),
			('r4', 'other-brand', 'list_join', '1')`,
	} {
		if _, err := conn.Exec(q); err != nil {
			t.Fatal(q, err)
		}
	}
	enrolled := func() []string {
		rows, err := conn.Query(`SELECT sequence_id FROM contact_sequence_state WHERE contact_id = 'ana' ORDER BY sequence_id`)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var got []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				t.Fatal(err)
			}
			got = append(got, id)
		}
		return got
	}

	if err := seqs.SubscriptionActivated(ctx, "ana", 1); err != nil {
		t.Fatal(err)
	}
	if got := enrolled(); !slices.Equal(got, []string{"on-join"}) {
		t.Fatalf("list_join: %v", got)
	}

	if _, err := store.AddContactTag(ctx, db.AddContactTagParams{ContactID: sql.NullString{String: "ana", Valid: true}, Tag: "vip"}); err != nil {
		t.Fatal(err)
	}
	if got := enrolled(); !slices.Equal(got, []string{"on-join", "on-tag"}) {
		t.Fatalf("tag_added: %v", got)
	}

	if err := seqs.StartSequenceByID(ctx, "ana", "first"); err != nil {
		t.Fatal(err)
	}
	seqs.AdvanceAfterSend(ctx, "ana", "f1") // the last email of "first" went
	if got := enrolled(); !slices.Equal(got, []string{"after-first", "first", "on-join", "on-tag"}) {
		t.Fatalf("sequence_complete: %v", got)
	}
}

// A rule that could never run is refused when it is saved.
func TestCheckEntryRule(t *testing.T) {
	for _, trigger := range EntryRuleTriggers {
		if err := CheckEntryRule(trigger, "x"); err != nil {
			t.Errorf("%s: %v", trigger, err)
		}
	}
	if CheckEntryRule("manual", "x") == nil || CheckEntryRule("list_subscribe", "1") == nil {
		t.Error("an unknown trigger is refused")
	}
	if CheckEntryRule(TriggerTagAdded, " ") == nil {
		t.Error("a rule waits on something")
	}
}
