package mcpauth

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func newVerifyFixture(t *testing.T) (*Authenticator, *db.Store) {
	t.Helper()
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db"))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn))
	store := db.NewStore(conn)
	_, err = store.CreateUser(context.Background(), db.CreateUserParams{
		ID: "u1", Email: "u1@example.com", PasswordHash: "x", Name: "U", Role: "agent", Status: "active", EmailVerified: int64(1),
	})
	require.NoError(t, err)
	return NewAuthenticator(&svc.ServiceContext{DB: store}), store
}

func addKey(t *testing.T, store *db.Store, id, key, scopes string) {
	t.Helper()
	_, err := store.CreateMCPAPIKey(context.Background(), db.CreateMCPAPIKeyParams{
		ID: id, UserID: "u1", Name: id, KeyHash: HashToken(key), KeyPrefix: key[:6], Scopes: sql.NullString{String: scopes, Valid: true},
	})
	require.NoError(t, err)
}

// A revoked key used to keep working until the pod restarted, because a
// cache answered before the database. Every request now reads the database.
func TestVerifyAPIKey_RevokeTakesEffectAtOnce(t *testing.T) {
	a, store := newVerifyFixture(t)
	addKey(t, store, "k1", "lv_live_key", ScopeFull)
	verify := a.TokenVerifier()

	info, err := verify(context.Background(), "lv_live_key", nil)
	require.NoError(t, err)
	assert.Equal(t, "u1", info.Extra["user_id"])
	assert.Equal(t, []string{ScopeFull}, info.Scopes)

	n, err := store.RevokeMCPAPIKey(context.Background(), db.RevokeMCPAPIKeyParams{ID: "k1", UserID: "u1"})
	require.NoError(t, err)
	require.EqualValues(t, 1, n)

	_, err = verify(context.Background(), "lv_live_key", nil)
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestVerifyAPIKey_RefusesInactivePerson(t *testing.T) {
	a, store := newVerifyFixture(t)
	addKey(t, store, "k1", "lv_live_key", ScopeFull)
	_, err := store.GetDB().Exec(`UPDATE users SET status = 'inactive' WHERE id = 'u1'`)
	require.NoError(t, err)

	_, err = a.TokenVerifier()(context.Background(), "lv_live_key", nil)
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestVerifyAPIKey_UnknownKey(t *testing.T) {
	a, _ := newVerifyFixture(t)
	_, err := a.TokenVerifier()(context.Background(), "lv_nope", nil)
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestParseScopes(t *testing.T) {
	assert.Equal(t, []string{"mcp:full"}, ParseScopes("mcp:full"))
	assert.Equal(t, []string{"mcp:full", "offline_access"}, ParseScopes("mcp:full offline_access"))
	assert.Equal(t, []string{"mcp:full", "offline_access"}, ParseScopes("mcp:full,offline_access"))
	assert.Equal(t, []string{"mcp:full"}, ParseScopes(`["mcp:full"]`))
	assert.Empty(t, ParseScopes(""))
}
