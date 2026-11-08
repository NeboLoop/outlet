package backup

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

type downloadRequest struct {
	Id string `path:"id"`
}

// DownloadHandler serves GET /api/admin/backup/:id/download behind the
// admin auth middleware: a completed backup's file, to a super admin. It
// streams the file, which a handler generated from outlet.api can't (those
// answer with JSON), so cmd/serve.go registers it.
func DownloadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req downloadRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, errorx.NewBadRequestError(err.Error()))
			return
		}
		if _, err := access.Require(r.Context(), svcCtx.DB, "", access.SuperAdmin); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		backup, err := svcCtx.DB.GetBackup(r.Context(), req.Id)
		if errors.Is(err, sql.ErrNoRows) {
			httpx.ErrorCtx(r.Context(), w, errorx.NewNotFoundError("Backup not found."))
			return
		}
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		if backup.Status != "completed" {
			httpx.ErrorCtx(r.Context(), w, errorx.NewBadRequestError(fmt.Sprintf("The backup is not ready to download (status: %s).", backup.Status)))
			return
		}
		if !backup.FilePath.Valid || backup.FilePath.String == "" {
			httpx.ErrorCtx(r.Context(), w, errorx.NewNotFoundError("The backup file is not on this server."))
			return
		}
		file, err := os.Open(backup.FilePath.String)
		if err != nil {
			logx.WithContext(r.Context()).Errorf("Failed to open backup file: %v", err)
			httpx.ErrorCtx(r.Context(), w, errorx.NewNotFoundError("The backup file is not on this server."))
			return
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, errorx.NewInternalError("Could not read the backup file."))
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", backup.Filename))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
		w.Header().Set("Cache-Control", "no-store")
		// Flushed as it goes, so the file is never held in memory whole
		if _, err := io.CopyBuffer(flushWriter{w}, file, make([]byte, 256<<10)); err != nil {
			// The download has started, so its status can't change; it is cut off
			logx.WithContext(r.Context()).Errorf("Backup %s download cut off: %v", backup.Filename, err)
		}
	}
}

// flushWriter sends each write to the client at once.
type flushWriter struct{ w http.ResponseWriter }

func (f flushWriter) Write(p []byte) (int, error) {
	n, err := f.w.Write(p)
	if fl, ok := f.w.(http.Flusher); ok && err == nil {
		fl.Flush()
	}
	return n, err
}
