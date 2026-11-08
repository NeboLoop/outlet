package db_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"

	_ "modernc.org/sqlite"
)

func newStore(t *testing.T) *db.Store {
	t.Helper()
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "s.db")+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := migrations.Run(conn, nil); err != nil {
		t.Fatal(err)
	}
	store := db.NewStore(conn)
	if _, err := store.CreateOrganization(context.Background(), db.CreateOrganizationParams{ID: "org-1", Name: "Acme", Slug: "acme", ApiKey: "k"}); err != nil {
		t.Fatal(err)
	}
	return store
}

// A hand-added or imported address that isn't one is refused, by the one path
// both take.
func TestAddSubscriber_RefusesAnInvalidAddress(t *testing.T) {
	store := newStore(t)
	for _, bad := range []string{"not-an-email", "Ann <ann@example.com>", ""} {
		if _, err := store.AddSubscriber(context.Background(), "org-1", 0, bad, "", "import", false); !errors.Is(err, db.ErrInvalidEmail) {
			t.Errorf("%q: got %v, want ErrInvalidEmail", bad, err)
		}
	}
	if _, err := store.AddSubscriber(context.Background(), "org-1", 0, " Ann@Example.com ", "", "import", false); err != nil {
		t.Errorf("a valid address: %v", err)
	}
}

// Every import worker sees the same pending job; only one claims it.
func TestClaimImportJob_OnlyOnce(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()
	job, err := store.CreateImportJob(ctx, db.CreateImportJobParams{ID: "job-1", OrgID: "org-1", Type: "subscribers", Filename: "a.csv"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.ClaimImportJob(ctx, job.ID)
	if err != nil || first != 1 {
		t.Fatalf("first claim: %d, %v", first, err)
	}
	if second, err := store.ClaimImportJob(ctx, job.ID); err != nil || second != 0 {
		t.Fatalf("second claim: %d, %v", second, err)
	}
}
