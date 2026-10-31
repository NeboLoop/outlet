package backup

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DownloadBackupLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDownloadBackupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DownloadBackupLogic {
	return &DownloadBackupLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DownloadBackup validates the backup exists and is ready for download (legacy method)
func (l *DownloadBackupLogic) DownloadBackup(req *types.DownloadBackupRequest) error {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return err
	}

	// Get backup record
	backup, err := l.svcCtx.DB.GetBackup(l.ctx, req.Id)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("backup not found")
		}
		return err
	}

	// Check if backup is completed
	if backup.Status != "completed" {
		return fmt.Errorf("backup is not ready for download (status: %s)", backup.Status)
	}

	// Check if file exists locally
	if !backup.FilePath.Valid || backup.FilePath.String == "" {
		return fmt.Errorf("backup file not available")
	}

	// Verify file exists on disk
	if _, err := os.Stat(backup.FilePath.String); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found on disk")
	}

	// Success - the handler will stream the file
	return nil
}

// Download streams the backup file to the client
func (l *DownloadBackupLogic) Download(w http.ResponseWriter, r *http.Request, req *types.DownloadBackupRequest) error {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return err
	}

	// Get backup record
	backup, err := l.svcCtx.DB.GetBackup(l.ctx, req.Id)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("backup not found")
		}
		return err
	}

	// Check if backup is completed
	if backup.Status != "completed" {
		return fmt.Errorf("backup is not ready for download (status: %s)", backup.Status)
	}

	// Check if file exists locally
	if !backup.FilePath.Valid || backup.FilePath.String == "" {
		return fmt.Errorf("backup file not available")
	}

	// Open file
	file, err := os.Open(backup.FilePath.String)
	if err != nil {
		l.Errorf("Failed to open backup file: %v", err)
		return fmt.Errorf("backup file not found on disk")
	}
	defer file.Close()

	// Get file info for content length
	fileInfo, err := file.Stat()
	if err != nil {
		l.Errorf("Failed to stat backup file: %v", err)
		return fmt.Errorf("failed to read backup file")
	}

	// Set download headers
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", backup.Filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")

	// Stream file to client
	if _, err := io.Copy(w, file); err != nil {
		l.Errorf("Failed to stream backup file: %v", err)
		return nil // Don't return error after headers sent
	}

	l.Infof("Download completed: backup %s (%d bytes)", backup.Filename, fileInfo.Size())
	return nil
}

// GetBackupFilePath returns the file path for a backup (used by custom download handler)
func (l *DownloadBackupLogic) GetBackupFilePath(id string) (string, string, error) {
	backup, err := l.svcCtx.DB.GetBackup(l.ctx, id)
	if err != nil {
		return "", "", err
	}

	if backup.Status != "completed" {
		return "", "", fmt.Errorf("backup not ready")
	}

	if !backup.FilePath.Valid {
		return "", "", fmt.Errorf("no file path")
	}

	return backup.FilePath.String, backup.Filename, nil
}
