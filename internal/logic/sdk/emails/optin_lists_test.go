package emails

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// Marketing mail sent through the API never goes to an outreach list, even
// to someone active on it.
func TestSend_MarketingRefusesAnOutreachList(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	for _, q := range []string{
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('org1', 'Brand', 'brand', 'k1')`,
		`INSERT INTO email_lists (id, public_id, org_id, name, slug, kind) VALUES (2, 'dental', 'org1', 'Utah dental', 'utah-dental', 'outreach')`,
		`INSERT INTO contacts (id, org_id, name, email, status) VALUES ('dentist', 'org1', 'D', 'dentist@example.com', 'active')`,
		`INSERT INTO list_subscribers (id, list_id, contact_id, status) VALUES ('s2', 2, 'dentist', 'active')`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err, q)
	}
	store := db.NewStore(conn)
	org, err := store.GetOrganizationByID(context.Background(), "org1")
	require.NoError(t, err)

	resp, err := Send(context.Background(), &svc.ServiceContext{DB: store}, org, &types.SendEmailRequest{
		To: "dentist@example.com", Subject: "Hi", Body: "<p>Hi</p>", Type: "marketing", ListSlug: "utah-dental",
	}, ViaAPI)
	require.Error(t, err)
	assert.True(t, errors.Is(err, email.ErrInvalidMessage))
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Message, `"Utah dental" is an outreach list`)
}
