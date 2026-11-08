package backup

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/middleware"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/rest/router"
	_ "modernc.org/sqlite"
)

// A super admin downloads a backup's bytes through the admin auth
// middleware; a brand admin, or no one signed in, is refused.
func TestDownloadBackup_StreamsTheFileToASuperAdminOnly(t *testing.T) {
	dir := t.TempDir()
	conn, err := sql.Open("sqlite", filepath.Join(dir, "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	store := db.NewStore(conn)
	ctx := context.Background()

	for _, u := range []db.CreateUserParams{
		{ID: "root", Email: "root@example.com", PasswordHash: "x", Name: "R", Role: "super_admin", Status: "active", EmailVerified: int64(1)},
		{ID: "mike", Email: "mike@example.com", PasswordHash: "x", Name: "M", Role: "agent", Status: "active", EmailVerified: int64(1)},
	} {
		_, err := store.CreateUser(ctx, u)
		require.NoError(t, err)
	}
	_, err = store.CreateOrganization(ctx, db.CreateOrganizationParams{ID: "alpha-id", Name: "alpha", Slug: "alpha", ApiKey: "key-alpha"})
	require.NoError(t, err)
	require.NoError(t, store.AddUserToOrganization(ctx, db.AddUserToOrganizationParams{OrgID: "alpha-id", UserID: "mike", Role: sql.NullString{String: "admin", Valid: true}}))

	contents := []byte("SQLite format 3\x00 the backup's bytes")
	file := filepath.Join(dir, "outlet-backup.db")
	require.NoError(t, os.WriteFile(file, contents, 0o600))
	_, err = conn.Exec(`INSERT INTO backup_history (id, filename, file_path, file_size, backup_type, storage_type, status)
		VALUES ('b1', 'outlet-backup.db', ?, ?, 'manual', 'local', 'completed')`, file, len(contents))
	require.NoError(t, err)

	// Routed as cmd/serve.go routes it: behind the admin auth middleware,
	// with the server's JSON errors
	errorx.SetupErrorHandler()
	const secret = "test-secret"
	auth := middleware.NewAuthMiddleware(secret, store).Handle
	r := router.NewRouter()
	require.NoError(t, r.Handle(http.MethodGet, "/api/admin/backup/:id/download", auth(DownloadHandler(&svc.ServiceContext{DB: store}))))
	get := func(userID, id string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/backup/"+id+"/download", nil)
		if userID != "" {
			token, err := utils.GenerateToken(userID, userID+"@example.com", "", secret, time.Hour)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	w := get("root", "b1")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, contents, w.Body.Bytes())
	assert.Equal(t, "application/octet-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, `attachment; filename="outlet-backup.db"`, w.Header().Get("Content-Disposition"))

	w = get("mike", "b1")
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.NotContains(t, w.Body.String(), "the backup's bytes")

	w = get("", "b1")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.NotContains(t, w.Body.String(), "the backup's bytes")

	w = get("root", "missing")
	assert.Equal(t, http.StatusNotFound, w.Code)
}
