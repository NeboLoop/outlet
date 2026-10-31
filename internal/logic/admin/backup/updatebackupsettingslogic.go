package backup

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateBackupSettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateBackupSettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateBackupSettingsLogic {
	return &UpdateBackupSettingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateBackupSettingsLogic) UpdateBackupSettings(req *types.BackupSettingsRequest) (resp *types.BackupSettingsResponse, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return nil, err
	}

	// Helper to upsert a text setting
	upsertText := func(key, value, desc string) error {
		_, err := l.svcCtx.DB.UpsertPlatformSetting(l.ctx, db.UpsertPlatformSettingParams{
			Key:         key,
			ValueText:   sql.NullString{String: value, Valid: true},
			Description: sql.NullString{String: desc, Valid: true},
			Category:    "backup",
			IsSensitive: sql.NullInt64{Int64: 0, Valid: true},
		})
		return err
	}

	// Helper to upsert a sensitive setting (encrypted)
	upsertSensitive := func(key, value, desc string) error {
		encryptedValue := value
		if l.svcCtx.CryptoService != nil {
			encrypted, err := l.svcCtx.CryptoService.EncryptString(value)
			if err != nil {
				return fmt.Errorf("failed to encrypt %s: %w", key, err)
			}
			encryptedValue = hex.EncodeToString(encrypted)
		}
		_, err := l.svcCtx.DB.UpsertPlatformSetting(l.ctx, db.UpsertPlatformSettingParams{
			Key:            key,
			ValueEncrypted: sql.NullString{String: encryptedValue, Valid: true},
			Description:    sql.NullString{String: desc, Valid: true},
			Category:       "backup",
			IsSensitive:    sql.NullInt64{Int64: 1, Valid: true},
		})
		return err
	}

	// Update S3 settings
	if err := upsertText("backup.s3_enabled", fmt.Sprintf("%t", req.S3Enabled), "Enable S3 backup storage"); err != nil {
		return nil, err
	}

	if req.S3Bucket != "" {
		if err := upsertText("backup.s3_bucket", req.S3Bucket, "S3 bucket name"); err != nil {
			return nil, err
		}
	}

	if req.S3Region != "" {
		if err := upsertText("backup.s3_region", req.S3Region, "S3 region"); err != nil {
			return nil, err
		}
	}

	if req.S3Prefix != "" {
		if err := upsertText("backup.s3_prefix", req.S3Prefix, "S3 key prefix"); err != nil {
			return nil, err
		}
	}

	if req.S3AccessKey != "" {
		if err := upsertSensitive("backup.s3_access_key", req.S3AccessKey, "S3 access key"); err != nil {
			return nil, err
		}
	}

	if req.S3SecretKey != "" {
		if err := upsertSensitive("backup.s3_secret_key", req.S3SecretKey, "S3 secret key"); err != nil {
			return nil, err
		}
	}

	// Update schedule settings
	if err := upsertText("backup.schedule_enabled", fmt.Sprintf("%t", req.ScheduleEnabled), "Enable scheduled backups"); err != nil {
		return nil, err
	}

	if req.ScheduleCron != "" {
		if err := upsertText("backup.schedule_cron", req.ScheduleCron, "Backup schedule cron expression"); err != nil {
			return nil, err
		}
	}

	if req.RetentionDays > 0 {
		if err := upsertText("backup.retention_days", strconv.Itoa(req.RetentionDays), "Backup retention period in days"); err != nil {
			return nil, err
		}
	}

	// Return updated settings
	return NewGetBackupSettingsLogic(l.ctx, l.svcCtx).GetBackupSettings()
}
