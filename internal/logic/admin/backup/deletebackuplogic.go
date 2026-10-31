package backup

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/outlet-sh/outlet/internal/access"
	backupService "github.com/outlet-sh/outlet/internal/services/backup"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteBackupLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteBackupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteBackupLogic {
	return &DeleteBackupLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteBackupLogic) DeleteBackup(req *types.DeleteBackupRequest) (resp *types.Response, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return nil, err
	}

	// Get backup to find file path
	backup, err := l.svcCtx.DB.GetBackup(l.ctx, req.Id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("backup not found")
		}
		return nil, err
	}

	// Delete local file if it exists
	if backup.FilePath.Valid && backup.FilePath.String != "" {
		if err := os.Remove(backup.FilePath.String); err != nil && !os.IsNotExist(err) {
			l.Errorf("Failed to delete backup file %s: %v", backup.FilePath.String, err)
		}
	}

	// Delete from S3 if stored there
	if backup.StorageType == "s3" && backup.S3Bucket.Valid && backup.S3Key.Valid {
		s3Config := l.getS3Config()
		if s3Config != nil {
			if err := backupService.DeleteFromS3(l.ctx, *s3Config, backup.S3Key.String); err != nil {
				l.Errorf("Failed to delete backup from S3: %v", err)
				// Continue with database deletion even if S3 delete fails
			} else {
				l.Infof("Deleted backup from S3: %s/%s", backup.S3Bucket.String, backup.S3Key.String)
			}
		} else {
			l.Errorf("Cannot delete S3 backup: S3 not configured")
		}
	}

	// Delete from database
	if err := l.svcCtx.DB.DeleteBackup(l.ctx, req.Id); err != nil {
		return nil, fmt.Errorf("failed to delete backup record: %w", err)
	}

	return &types.Response{
		Success: true,
		Message: "Backup deleted successfully",
	}, nil
}

// getS3Config retrieves S3 configuration from platform settings
func (l *DeleteBackupLogic) getS3Config() *backupService.S3Config {
	settings, err := l.svcCtx.DB.GetPlatformSettingsByCategory(l.ctx, "backup")
	if err != nil {
		l.Errorf("Failed to get backup settings: %v", err)
		return nil
	}

	cfg := &backupService.S3Config{}
	var accessKeyEncrypted, secretKeyEncrypted []byte

	for _, s := range settings {
		switch s.Key {
		case "backup.s3_enabled":
			if s.ValueText.String != "true" {
				return nil
			}
		case "backup.s3_bucket":
			cfg.Bucket = s.ValueText.String
		case "backup.s3_region":
			cfg.Region = s.ValueText.String
		case "backup.s3_prefix":
			cfg.Prefix = s.ValueText.String
		case "backup.s3_access_key":
			if s.ValueEncrypted.Valid {
				var err error
				accessKeyEncrypted, err = hex.DecodeString(s.ValueEncrypted.String)
				if err != nil {
					l.Errorf("Failed to decode S3 access key: %v", err)
				}
			}
		case "backup.s3_secret_key":
			if s.ValueEncrypted.Valid {
				var err error
				secretKeyEncrypted, err = hex.DecodeString(s.ValueEncrypted.String)
				if err != nil {
					l.Errorf("Failed to decode S3 secret key: %v", err)
				}
			}
		}
	}

	// Decrypt credentials if crypto service is available
	if l.svcCtx.CryptoService != nil {
		if len(accessKeyEncrypted) > 0 {
			decrypted, err := l.svcCtx.CryptoService.DecryptString(accessKeyEncrypted)
			if err == nil {
				cfg.AccessKey = decrypted
			} else {
				l.Errorf("Failed to decrypt S3 access key: %v", err)
			}
		}
		if len(secretKeyEncrypted) > 0 {
			decrypted, err := l.svcCtx.CryptoService.DecryptString(secretKeyEncrypted)
			if err == nil {
				cfg.SecretKey = decrypted
			} else {
				l.Errorf("Failed to decrypt S3 secret key: %v", err)
			}
		}
	}

	if cfg.Bucket == "" {
		return nil
	}

	return cfg
}
