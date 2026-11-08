package imports

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	listlogic "github.com/outlet-sh/outlet/internal/logic/admin/lists"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/workers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// A brand member adds people to a list by hand and by CSV import; both put
// them on the list active, and a blocked domain is refused.
func TestAddAndImportSubscribers(t *testing.T) {
	dir := t.TempDir()
	conn, err := sql.Open("sqlite", filepath.Join(dir, "outlet.db")+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	store := db.NewStore(conn)
	ctx := context.Background()
	for _, q := range []string{
		`INSERT INTO users (id, email, password_hash, name, role, status) VALUES ('mike', 'mike@example.com', 'x', 'Mike', 'agent', 'active')`,
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('org1', 'One', 'one', 'k1'), ('org2', 'Two', 'two', 'k2')`,
		`INSERT INTO org_users (org_id, user_id, role) VALUES ('org1', 'mike', 'member')`,
		`INSERT INTO email_lists (id, public_id, org_id, name, slug) VALUES (1, 'p1', 'org1', 'News', 'news'), (2, 'p2', 'org2', 'Other', 'other')`,
		`INSERT INTO blocked_domains (org_id, domain) VALUES ('org1', 'spam.example')`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err, q)
	}
	user, err := access.Active(ctx, store, "mike")
	require.NoError(t, err)
	asMember := access.WithUser(ctx, user)
	s := &svc.ServiceContext{DB: store, UploadDir: filepath.Join(dir, "uploads")}

	// By hand
	add := func(listID, address string) (*types.Response, error) {
		return listlogic.NewAddListSubscriberLogic(asMember, s).AddListSubscriber(&types.AddListSubscriberRequest{Id: listID, Email: address, Name: "Ann"})
	}
	resp, err := add("1", "ann@example.com")
	require.NoError(t, err)
	assert.Equal(t, "Subscriber added", resp.Message)
	resp, err = add("1", "ann@example.com")
	require.NoError(t, err)
	assert.Equal(t, "Already on this list", resp.Message)
	_, err = add("1", "bob@spam.example")
	assert.ErrorContains(t, err, "blocklist")
	_, err = add("1", "not an address")
	assert.Error(t, err)
	_, err = add("2", "ann@example.com")
	assert.Error(t, err, "another brand's list")

	// By CSV
	_, err = NewImportListSubscribersLogic(asMember, s).ImportListSubscribers(&types.ImportListSubscribersRequest{Id: "1", Filename: "people.csv", Csv: "name,phone\nNo,1\n"})
	assert.ErrorContains(t, err, "email column")

	job, err := NewImportListSubscribersLogic(asMember, s).ImportListSubscribers(&types.ImportListSubscribersRequest{
		Id: "1", Filename: "people.csv",
		Csv: "Email,Name\ncy@example.com,Cy\nann@example.com,Ann B\ndee@spam.example,Dee\n",
	})
	require.NoError(t, err)
	assert.Equal(t, "people.csv", job.Filename)
	assert.Equal(t, "pending", job.Status)

	w := workers.NewImportWorker(store, workers.ImportWorkerConfig{PollInterval: 10 * time.Millisecond, Workers: 1, BatchSize: 100, UploadDir: s.UploadDir})
	w.Start()
	t.Cleanup(w.Stop)
	require.Eventually(t, func() bool {
		got, err := NewGetImportJobLogic(access.WithBrand(asMember, "org1"), s).GetImportJob(&types.GetImportJobRequest{Id: job.Id})
		return err == nil && got.Status == "completed"
	}, 5*time.Second, 10*time.Millisecond)

	got, err := NewGetImportJobLogic(access.WithBrand(asMember, "org1"), s).GetImportJob(&types.GetImportJobRequest{Id: job.Id})
	require.NoError(t, err)
	assert.Equal(t, 3, got.ProcessedRows)
	assert.Equal(t, 2, got.SuccessCount)
	assert.Equal(t, 1, got.ErrorCount, "the blocked domain")

	subs, err := store.ListListSubscribers(ctx, db.ListListSubscribersParams{ListID: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Len(t, subs, 2, "ann and cy")
	// The worker removes the CSV just after it marks the job completed.
	assert.Eventually(t, func() bool {
		_, err := os.Stat(workers.ImportFile(s.UploadDir, job.Id))
		return os.IsNotExist(err)
	}, 5*time.Second, 10*time.Millisecond, "the CSV is removed once imported")
}
