package gdpr

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/services/email"

	_ "modernc.org/sqlite"
)

// A deletion removes the contact, asks the providers when told to, and is
// recorded in the GDPR audit log without the address itself. Another brand's
// contact can't be deleted.
func TestDelete_RecordsTheRequest(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "g.db")+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := migrations.Run(conn, nil); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('org-a', 'A', 'a', 'key-a'), ('org-b', 'B', 'b', 'key-b')`,
		`INSERT INTO contacts (id, org_id, name, email) VALUES ('ann', 'org-a', 'Ann', 'Ann@X.example'), ('bea', 'org-b', 'Bea', 'bea@x.example')`,
	} {
		if _, err := conn.Exec(q); err != nil {
			t.Fatal(q, err)
		}
	}
	store := db.NewStore(conn)
	mail := email.NewService(store, nil)
	ctx := context.Background()

	if _, err := Delete(ctx, store, mail, DeleteRequest{OrgID: "org-a", ContactID: "bea", Via: "admin", RequestedBy: "u1"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("another brand's contact: %v", err)
	}

	got, err := Delete(ctx, store, mail, DeleteRequest{
		OrgID: "org-a", ContactID: "ann", ProviderLogs: true,
		RequestedBy: "u1", RequestedByEmail: "admin@a.example", Via: "admin",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Providers) != 1 || got.Providers[0].Provider != "SMTP relay" {
		t.Errorf("providers: %+v", got.Providers)
	}
	var n int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM contacts WHERE id = 'ann'`).Scan(&n); err != nil || n != 0 {
		t.Errorf("contact still there: %d %v", n, err)
	}

	row, err := store.GetGDPRDeletion(ctx, db.GetGDPRDeletionParams{ID: got.AuditID, OrgID: "org-a"})
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("ann@x.example"))
	if row.EmailSha256 != hex.EncodeToString(sum[:]) || row.RequestedBy != "u1" || row.Via != "admin" || row.ProvidersRequested != 1 ||
		!strings.Contains(row.ProviderResults, "SMTP relay") {
		t.Errorf("audit row: %+v", row)
	}
	if strings.Contains(strings.ToLower(row.ProviderResults), "ann@x.example") {
		t.Error("the audit row keeps the address")
	}
}
