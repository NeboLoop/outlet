package backup

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/robfig/cron/v3"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetBackupSettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetBackupSettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetBackupSettingsLogic {
	return &GetBackupSettingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetBackupSettingsLogic) GetBackupSettings() (resp *types.BackupSettingsResponse, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return nil, err
	}

	// Get settings from platform_settings table
	settings, err := l.svcCtx.DB.GetPlatformSettingsByCategory(l.ctx, "backup")
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Build response from settings
	// By default, scheduled backups are enabled at 3 AM daily with 30 day retention
	resp = &types.BackupSettingsResponse{
		S3Enabled:       false,
		S3Bucket:        "",
		S3Region:        "",
		S3Prefix:        "",
		HasS3Creds:      false,
		ScheduleEnabled: true,                // Enabled by default for data safety
		ScheduleCron:    "0 3 * * *",         // Default: 3 AM daily
		RetentionDays:   30,                  // Default: 30 days
	}

	for _, s := range settings {
		value := s.ValueText.String
		switch s.Key {
		case "backup.s3_enabled":
			resp.S3Enabled = value == "true"
		case "backup.s3_bucket":
			resp.S3Bucket = value
		case "backup.s3_region":
			resp.S3Region = value
		case "backup.s3_prefix":
			resp.S3Prefix = value
		case "backup.s3_access_key":
			// Just check if credentials exist (don't return actual value)
			resp.HasS3Creds = s.ValueEncrypted.Valid && s.ValueEncrypted.String != ""
		case "backup.schedule_enabled":
			resp.ScheduleEnabled = value == "true"
		case "backup.schedule_cron":
			if value != "" {
				resp.ScheduleCron = value
			}
		case "backup.retention_days":
			if days, err := strconv.Atoi(value); err == nil {
				resp.RetentionDays = days
			}
		}
	}

	// Get last backup time
	lastBackup, err := l.svcCtx.DB.GetLatestBackup(l.ctx)
	if err == nil {
		resp.LastBackupAt = lastBackup.CompletedAt.String
	}

	// Calculate next backup time from cron expression
	if resp.ScheduleEnabled && resp.ScheduleCron != "" {
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		schedule, err := parser.Parse(resp.ScheduleCron)
		if err == nil {
			nextRun := schedule.Next(time.Now())
			resp.NextBackupAt = nextRun.Format(time.RFC3339)
		}
	}

	return resp, nil
}
