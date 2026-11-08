package workers

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"

	_ "modernc.org/sqlite"
)

// A subscriber import starts the list's sign-up emails for the people it adds
// when the import asks to, and not when it doesn't (an old list).
func TestImportSubscribers_StartsSignupOnlyWhenAsked(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "i.db")+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := migrations.Run(conn, nil); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('org-a', 'A', 'a', 'key-a')`,
		`INSERT INTO email_lists (id, public_id, org_id, name, slug) VALUES (1, 'news', 'org-a', 'News', 'news')`,
	} {
		if _, err := conn.Exec(q); err != nil {
			t.Fatal(q, err)
		}
	}
	store := db.NewStore(conn)
	var started []string
	store.OnSubscriptionActivated(func(ctx context.Context, contactID string, listID int64) error {
		c, err := store.GetContact(ctx, contactID)
		if err != nil {
			return err
		}
		started = append(started, c.Email)
		return nil
	})

	dir := t.TempDir()
	w := NewImportWorker(store, ImportWorkerConfig{UploadDir: dir, BatchSize: 10})
	run := func(id, csv string, startSignup bool) {
		t.Helper()
		if err := os.WriteFile(ImportFile(dir, id), []byte(csv), 0o600); err != nil {
			t.Fatal(err)
		}
		opts, _ := json.Marshal(ImportOptions{StartSignup: startSignup})
		job, err := store.CreateImportJob(context.Background(), db.CreateImportJobParams{
			ID: id, OrgID: "org-a", ListID: sql.NullInt64{Int64: 1, Valid: true}, Type: "subscribers",
			Filename: id + ".csv", Options: sql.NullString{String: string(opts), Valid: true},
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := w.processImportJob(job); err != nil {
			t.Fatal(err)
		}
	}

	run("old-list", "email,name\nold@example.com,Old\n", false)
	run("new-list", "email,name\nnew@example.com,New\n", true)

	if !slices.Equal(started, []string{"new@example.com"}) {
		t.Fatalf("sign-up emails started for %v, want only new@example.com", started)
	}
}
