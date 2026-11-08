package gdpr

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/services/gdpr"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// The export comes back in the response, ready to save: there is no file on
// the server and no link to fetch it from.
func TestExportContactData_ReturnsTheExport(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	for _, q := range []string{
		`INSERT INTO users (id, email, password_hash, name, role, status, email_verified) VALUES ('mike', 'mike@example.com', 'x', 'M', 'agent', 'active', 1)`,
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('alpha-id', 'alpha', 'alpha', 'k-a')`,
		`INSERT INTO org_users (org_id, user_id, role) VALUES ('alpha-id', 'mike', 'member')`,
		`INSERT INTO contacts (id, org_id, name, email) VALUES ('ann-id', 'alpha-id', 'Ann', 'ann@example.com')`,
		`INSERT INTO contact_tags (contact_id, tag) VALUES ('ann-id', 'vip')`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err, q)
	}
	store := db.NewStore(conn)
	user, err := access.Active(context.Background(), store, "mike")
	require.NoError(t, err)
	ctx := access.WithBrand(access.WithUser(context.Background(), user), "alpha-id")

	got, err := NewExportContactDataLogic(ctx, &svc.ServiceContext{DB: store}).ExportContactData(&types.GDPRExportRequest{ContactId: "ann-id"})
	require.NoError(t, err)
	assert.Equal(t, "json", got.Format)
	assert.Equal(t, "gdpr-export-ann-id.json", got.Filename)
	var export gdpr.Data
	require.NoError(t, json.Unmarshal([]byte(got.Data), &export))
	assert.Equal(t, "ann@example.com", export.Contact.Email)
	assert.Equal(t, []string{"vip"}, export.Tags)

	_, err = NewExportContactDataLogic(ctx, &svc.ServiceContext{DB: store}).ExportContactData(&types.GDPRExportRequest{ContactId: "nobody"})
	assert.ErrorContains(t, err, "Contact not found")
}
