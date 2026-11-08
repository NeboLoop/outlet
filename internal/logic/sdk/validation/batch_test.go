package validation

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/outlet-sh/outlet/internal/config"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/middleware"
	validationsvc "github.com/outlet-sh/outlet/internal/services/validation"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// A batch is accepted at once as a background validation job, without
// checking any address inside the request (each check can take 20 seconds);
// its ID fetches the job and a result per address, pending until checked.
// Another brand's key can't read it.
func TestValidateBatchIsAcceptedAndCheckedInTheBackground(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db"))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	_, err = conn.Exec(`INSERT INTO organizations (id, name, slug, api_key) VALUES ('org1', 'Org', 'org', 'k1'), ('org2', 'Other', 'other', 'k2')`)
	require.NoError(t, err)
	store := db.NewStore(conn)
	s := &svc.ServiceContext{DB: store, Validation: validationsvc.New(store, config.ValidationConfig{CacheDays: 30, Concurrency: 2}, nil)}
	asOrg := func(id string) context.Context { return context.WithValue(context.Background(), middleware.OrgIDKey, id) }

	job, err := NewValidateBatchLogic(asOrg("org1"), s).ValidateBatch(&types.ValidateBatchRequest{
		Emails: []string{"ann@example.com", "Bob@Example.com", "ann@example.com", "cat@example.com"},
	})
	require.NoError(t, err)
	assert.Equal(t, "processing", job.Status)
	assert.Equal(t, 3, job.Total)
	assert.Equal(t, 0, job.Processed)

	got, err := NewGetValidateBatchLogic(asOrg("org1"), s).GetValidateBatch(&types.ValidationJobIdRequest{Id: job.Id})
	require.NoError(t, err)
	assert.Equal(t, job.Id, got.Job.Id)
	require.Len(t, got.Results, 3)
	assert.Equal(t, "ann@example.com", got.Results[0].Address)
	assert.Equal(t, "bob@example.com", got.Results[1].Address)
	for _, r := range got.Results {
		assert.Equal(t, "pending", r.Status)
	}

	_, err = NewGetValidateBatchLogic(asOrg("org2"), s).GetValidateBatch(&types.ValidationJobIdRequest{Id: job.Id})
	assert.Error(t, err)

	_, err = NewValidateBatchLogic(asOrg("org1"), s).ValidateBatch(&types.ValidateBatchRequest{Emails: make101()})
	assert.Error(t, err)
}

func make101() []string {
	out := make([]string, validationsvc.MaxBatchEmails+1)
	for i := range out {
		out[i] = string(rune('a'+i%26)) + string(rune('a'+i/26)) + "@example.com"
	}
	return out
}
