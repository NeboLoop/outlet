package settings

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateEmailSettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateEmailSettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateEmailSettingsLogic {
	return &UpdateEmailSettingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateEmailSettingsLogic) UpdateEmailSettings(req *types.UpdateEmailSettingsRequest) (resp *types.UpdateSettingsResponse, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return nil, err
	}

	const awsCategory = "aws"
	const smtpCategory = "email"

	// Save AWS Access Key (sensitive - encrypted)
	if req.AwsAccessKey != "" {
		if l.svcCtx.CryptoService == nil {
			return nil, fmt.Errorf("encryption service not configured - cannot store sensitive data")
		}

		encrypted, err := l.svcCtx.CryptoService.EncryptString(req.AwsAccessKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt aws_access_key: %w", err)
		}

		_, err = l.svcCtx.DB.UpsertPlatformSetting(l.ctx, db.UpsertPlatformSettingParams{
			Key:            "aws_access_key",
			ValueEncrypted: sql.NullString{String: string(encrypted), Valid: true},
			Description:    sql.NullString{String: "AWS IAM Access Key ID", Valid: true},
			Category:       awsCategory,
			IsSensitive:    sql.NullInt64{Int64: 1, Valid: true},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to save aws_access_key: %w", err)
		}
	}

	// Save AWS Secret Key (sensitive - encrypted)
	if req.AwsSecretKey != "" {
		if l.svcCtx.CryptoService == nil {
			return nil, fmt.Errorf("encryption service not configured - cannot store sensitive data")
		}

		encrypted, err := l.svcCtx.CryptoService.EncryptString(req.AwsSecretKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt aws_secret_key: %w", err)
		}

		_, err = l.svcCtx.DB.UpsertPlatformSetting(l.ctx, db.UpsertPlatformSettingParams{
			Key:            "aws_secret_key",
			ValueEncrypted: sql.NullString{String: string(encrypted), Valid: true},
			Description:    sql.NullString{String: "AWS IAM Secret Access Key", Valid: true},
			Category:       awsCategory,
			IsSensitive:    sql.NullInt64{Int64: 1, Valid: true},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to save aws_secret_key: %w", err)
		}
	}

	// Save AWS Region (non-sensitive)
	if req.AwsRegion != "" {
		_, err := l.svcCtx.DB.UpsertPlatformSetting(l.ctx, db.UpsertPlatformSettingParams{
			Key:         "aws_region",
			ValueText:   sql.NullString{String: req.AwsRegion, Valid: true},
			Description: sql.NullString{String: "AWS region for SES", Valid: true},
			Category:    awsCategory,
			IsSensitive: sql.NullInt64{Int64: 0, Valid: true},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to save aws_region: %w", err)
		}
	}

	// Legacy SMTP settings (deprecated but still supported)

	// Save SMTP host (non-sensitive)
	if req.SmtpHost != "" {
		_, err := l.svcCtx.DB.UpsertPlatformSetting(l.ctx, db.UpsertPlatformSettingParams{
			Key:         "smtp_host",
			ValueText:   sql.NullString{String: req.SmtpHost, Valid: true},
			Description: sql.NullString{String: "SMTP server hostname", Valid: true},
			Category:    smtpCategory,
			IsSensitive: sql.NullInt64{Int64: 0, Valid: true},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to save smtp_host: %w", err)
		}
	}

	// Save SMTP port (non-sensitive)
	if req.SmtpPort > 0 {
		_, err := l.svcCtx.DB.UpsertPlatformSetting(l.ctx, db.UpsertPlatformSettingParams{
			Key:         "smtp_port",
			ValueText:   sql.NullString{String: fmt.Sprintf("%d", req.SmtpPort), Valid: true},
			Description: sql.NullString{String: "SMTP server port", Valid: true},
			Category:    smtpCategory,
			IsSensitive: sql.NullInt64{Int64: 0, Valid: true},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to save smtp_port: %w", err)
		}
	}

	// Save SMTP user (non-sensitive - just the username)
	if req.SmtpUser != "" {
		_, err := l.svcCtx.DB.UpsertPlatformSetting(l.ctx, db.UpsertPlatformSettingParams{
			Key:         "smtp_user",
			ValueText:   sql.NullString{String: req.SmtpUser, Valid: true},
			Description: sql.NullString{String: "SMTP username", Valid: true},
			Category:    smtpCategory,
			IsSensitive: sql.NullInt64{Int64: 0, Valid: true},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to save smtp_user: %w", err)
		}
	}

	// Save SMTP password (sensitive - encrypted)
	if req.SmtpPassword != "" {
		if l.svcCtx.CryptoService == nil {
			return nil, fmt.Errorf("encryption service not configured - cannot store sensitive data")
		}

		encrypted, err := l.svcCtx.CryptoService.EncryptString(req.SmtpPassword)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt smtp_password: %w", err)
		}

		_, err = l.svcCtx.DB.UpsertPlatformSetting(l.ctx, db.UpsertPlatformSettingParams{
			Key:            "smtp_password",
			ValueEncrypted: sql.NullString{String: string(encrypted), Valid: true},
			Description:    sql.NullString{String: "SMTP password", Valid: true},
			Category:       smtpCategory,
			IsSensitive:    sql.NullInt64{Int64: 1, Valid: true},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to save smtp_password: %w", err)
		}
	}

	// Save From Email (non-sensitive)
	if req.FromEmail != "" {
		_, err := l.svcCtx.DB.UpsertPlatformSetting(l.ctx, db.UpsertPlatformSettingParams{
			Key:         "from_email",
			ValueText:   sql.NullString{String: req.FromEmail, Valid: true},
			Description: sql.NullString{String: "Default from email address", Valid: true},
			Category:    smtpCategory,
			IsSensitive: sql.NullInt64{Int64: 0, Valid: true},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to save from_email: %w", err)
		}
	}

	// Save From Name (non-sensitive)
	if req.FromName != "" {
		_, err := l.svcCtx.DB.UpsertPlatformSetting(l.ctx, db.UpsertPlatformSettingParams{
			Key:         "from_name",
			ValueText:   sql.NullString{String: req.FromName, Valid: true},
			Description: sql.NullString{String: "Default from name", Valid: true},
			Category:    smtpCategory,
			IsSensitive: sql.NullInt64{Int64: 0, Valid: true},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to save from_name: %w", err)
		}
	}

	// Save Reply-To (non-sensitive)
	if req.ReplyTo != "" {
		_, err := l.svcCtx.DB.UpsertPlatformSetting(l.ctx, db.UpsertPlatformSettingParams{
			Key:         "reply_to",
			ValueText:   sql.NullString{String: req.ReplyTo, Valid: true},
			Description: sql.NullString{String: "Default reply-to email address", Valid: true},
			Category:    smtpCategory,
			IsSensitive: sql.NullInt64{Int64: 0, Valid: true},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to save reply_to: %w", err)
		}
	}

	return &types.UpdateSettingsResponse{
		Success: true,
		Message: "Settings saved successfully",
	}, nil
}
