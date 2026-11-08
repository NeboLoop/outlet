package backup

import (
	"bytes"
	"context"
	"database/sql"
	"io"
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

// newDownloadFixture is a database with a super admin (root), a brand admin
// (mike) and a completed local backup b1 holding contents.
func newDownloadFixture(t *testing.T, contents []byte) *db.Store {
	t.Helper()
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

	file := filepath.Join(dir, "outlet-backup.db")
	require.NoError(t, os.WriteFile(file, contents, 0o600))
	_, err = conn.Exec(`INSERT INTO backup_history (id, filename, file_path, file_size, backup_type, storage_type, status)
		VALUES ('b1', 'outlet-backup.db', ?, ?, 'manual', 'local', 'completed')`, file, len(contents))
	require.NoError(t, err)
	return store
}

const testSecret = "test-secret"

// downloadRouter routes the download as cmd/serve.go does: behind the admin
// auth middleware, with the server's JSON errors.
func downloadRouter(t *testing.T, store *db.Store) http.Handler {
	t.Helper()
	errorx.SetupErrorHandler()
	auth := middleware.NewAuthMiddleware(testSecret, store).Handle
	r := router.NewRouter()
	require.NoError(t, r.Handle(http.MethodGet, "/api/admin/backup/:id/download", auth(DownloadHandler(&svc.ServiceContext{DB: store}))))
	return r
}

func signIn(t *testing.T, req *http.Request, userID string) {
	t.Helper()
	token, err := utils.GenerateToken(userID, userID+"@example.com", "", testSecret, time.Hour)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
}

// A super admin downloads a backup's bytes through the admin auth
// middleware; a brand admin, or no one signed in, is refused.
func TestDownloadBackup_StreamsTheFileToASuperAdminOnly(t *testing.T) {
	contents := []byte("SQLite format 3\x00 the backup's bytes")
	r := downloadRouter(t, newDownloadFixture(t, contents))
	get := func(userID, id string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/backup/"+id+"/download", nil)
		if userID != "" {
			signIn(t, req, userID)
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

// A download has no time limit: a backup read slower than the server's
// write timeout still arrives whole, through the server's own wrappers.
func TestDownloadBackup_OutlivesWriteTimeout(t *testing.T) {
	contents := bytes.Repeat([]byte("0123456789abcdef"), 1<<19) // 8 MB
	r := downloadRouter(t, newDownloadFixture(t, contents))
	srv := httptest.NewUnstartedServer(middleware.RequestLog(middleware.Gzip(r).ServeHTTP))
	srv.Config.WriteTimeout = 100 * time.Millisecond
	srv.Start()
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/admin/backup/b1/download", nil)
	require.NoError(t, err)
	signIn(t, req, "root")
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)

	// Read slowly, so the download takes several write timeouts
	start := time.Now()
	var got bytes.Buffer
	buf := make([]byte, 256<<10)
	for {
		n, err := res.Body.Read(buf)
		got.Write(buf[:n])
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		time.Sleep(15 * time.Millisecond)
	}
	assert.Greater(t, time.Since(start), 3*srv.Config.WriteTimeout)
	assert.True(t, bytes.Equal(contents, got.Bytes()), "got %d of %d bytes", got.Len(), len(contents))
}
