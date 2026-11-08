package transactional

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// A template takes one of its brand's designs, changes it, and drops it;
// another brand's design is refused.
func TestTransactionalDesign(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	for _, q := range []string{
		`INSERT INTO users (id, email, password_hash, name, role, status, email_verified) VALUES ('mike', 'mike@example.com', 'x', 'M', 'agent', 'active', 1)`,
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('alpha-id', 'alpha', 'alpha', 'k-a'), ('bravo-id', 'bravo', 'bravo', 'k-b')`,
		`INSERT INTO org_users (org_id, user_id, role) VALUES ('alpha-id', 'mike', 'member')`,
		`INSERT INTO email_designs (id, org_id, name, slug, html_body) VALUES (7, 'alpha-id', 'A', 'a', '{{content}}'), (8, 'alpha-id', 'B', 'b', '{{content}}'), (9, 'bravo-id', 'C', 'c', '{{content}}')`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err, q)
	}
	store := db.NewStore(conn)
	s := &svc.ServiceContext{DB: store}
	user, err := access.Active(context.Background(), store, "mike")
	require.NoError(t, err)
	ctx := access.WithBrand(access.WithUser(context.Background(), user), "alpha-id")
	str := func(v string) *string { return &v }

	_, err = NewCreateTransactionalEmailLogic(ctx, s).CreateTransactionalEmail(&types.CreateTransactionalEmailRequest{
		Name: "R", Slug: "r", Subject: "S", HtmlBody: "<p>x</p>", DesignId: str("9"),
	})
	assert.ErrorContains(t, err, "Choose a design", "another brand's design")

	created, err := NewCreateTransactionalEmailLogic(ctx, s).CreateTransactionalEmail(&types.CreateTransactionalEmailRequest{
		Name: "R", Slug: "r", Subject: "S", HtmlBody: "<p>x</p>", DesignId: str("7"),
	})
	require.NoError(t, err)
	require.NotNil(t, created.DesignId)
	assert.Equal(t, "7", *created.DesignId)

	update := func(design *string) *types.TransactionalEmailInfo {
		got, err := NewUpdateTransactionalEmailLogic(ctx, s).UpdateTransactionalEmail(&types.UpdateTransactionalEmailRequest{
			Id: created.Id, Name: "R", Slug: "r", Subject: "S", HtmlBody: "<p>x</p>", IsActive: true, DesignId: design,
		})
		require.NoError(t, err)
		return got
	}
	got := update(str("8"))
	require.NotNil(t, got.DesignId)
	assert.Equal(t, "8", *got.DesignId)
	got = update(nil)
	require.NotNil(t, got.DesignId, "omitted keeps it")
	assert.Equal(t, "8", *got.DesignId)
	got = update(str(""))
	assert.Nil(t, got.DesignId, `"" removes it`)

	_, err = NewUpdateTransactionalEmailLogic(ctx, s).UpdateTransactionalEmail(&types.UpdateTransactionalEmailRequest{Id: created.Id, DesignId: str("9")})
	assert.ErrorContains(t, err, "Choose a design")
}
