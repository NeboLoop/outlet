package oauth

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/mcp/mcpauth"
	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// A disabled person used to keep getting new 30-day tokens from refresh.
func TestRefreshGrant_RefusesDisabledPerson(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn))
	store := db.NewStore(conn)
	ctx := context.Background()

	_, err = store.CreateUser(ctx, db.CreateUserParams{ID: "u1", Email: "u1@example.com", PasswordHash: "x", Name: "U", Role: "agent", Status: "active", EmailVerified: int64(1)})
	require.NoError(t, err)
	_, err = store.CreateMCPOAuthClient(ctx, db.CreateMCPOAuthClientParams{ID: "c1", ClientID: "client", ClientSecretHash: "x", Name: "Client", RedirectUris: "https://client.test/cb"})
	require.NoError(t, err)
	newToken := func(id, refresh string) {
		_, err := store.CreateMCPOAuthToken(ctx, db.CreateMCPOAuthTokenParams{
			ID: id, ClientID: "c1", UserID: "u1", AccessTokenHash: mcpauth.HashToken("access-" + id),
			RefreshTokenHash: sql.NullString{String: mcpauth.HashToken(refresh), Valid: true}, Scopes: mcpauth.ScopeFull,
			ExpiresAt:        time.Now().Add(time.Hour).Format(time.RFC3339),
			RefreshExpiresAt: sql.NullString{String: time.Now().Add(24 * time.Hour).Format(time.RFC3339), Valid: true},
		})
		require.NoError(t, err)
	}
	h := NewHandler(&svc.ServiceContext{DB: store}, "https://outlet.test")
	refresh := func(token string) (int, map[string]any) {
		form := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {token}, "client_id": {"client"}}
		req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()
		h.HandleToken(rr, req)
		var out map[string]any
		_ = json.Unmarshal(rr.Body.Bytes(), &out)
		return rr.Code, out
	}

	newToken("t1", "refresh-1")
	code, out := refresh("refresh-1")
	require.Equal(t, http.StatusOK, code, out)
	assert.NotEmpty(t, out["access_token"])

	newToken("t2", "refresh-2")
	_, err = store.GetDB().Exec(`UPDATE users SET status = 'inactive' WHERE id = 'u1'`)
	require.NoError(t, err)
	code, out = refresh("refresh-2")
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Equal(t, "invalid_grant", out["error"])
}
