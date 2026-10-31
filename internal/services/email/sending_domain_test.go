package email

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// A domain belongs to one brand, and a brand sends only from its own
// domains: alpha can't claim or send as neboai.com, which nebo holds.
func TestSendingDomains_BelongToOneBrand(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn))
	store := db.NewStore(conn)
	ctx := context.Background()
	for _, b := range []string{"nebo", "alpha"} {
		_, err := store.CreateOrganization(ctx, db.CreateOrganizationParams{ID: b, Name: b, Slug: b, ApiKey: "key-" + b})
		require.NoError(t, err)
	}
	_, err = store.CreateDomainIdentity(ctx, db.CreateDomainIdentityParams{ID: "d1", OrgID: "nebo", Domain: "neboai.com"})
	require.NoError(t, err)

	assert.NoError(t, CheckSender(ctx, store, "nebo", "noreply@neboai.com"))
	assert.NoError(t, CheckSender(ctx, store, "nebo", "Security@NeboAI.com"))
	assert.NoError(t, CheckSender(ctx, store, "alpha", ""), "the platform's default sender")
	err = CheckSender(ctx, store, "alpha", "security@neboai.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "can't send from security@neboai.com")
	assert.Error(t, CheckSender(ctx, store, "alpha", "news@alpha.example"), "a domain no brand holds")

	assert.Error(t, ClaimDomain(ctx, store, "alpha", "NEBOAI.com"))
	assert.NoError(t, ClaimDomain(ctx, store, "alpha", "alpha.example"))
	assert.NoError(t, ClaimDomain(ctx, store, "nebo", "neboai.com"), "its own domain")

	shared, err := DomainShared(ctx, store, "nebo", "neboai.com")
	require.NoError(t, err)
	assert.False(t, shared)
}
