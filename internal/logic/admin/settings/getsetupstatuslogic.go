package settings

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSetupStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetSetupStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSetupStatusLogic {
	return &GetSetupStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSetupStatusLogic) GetSetupStatus() (resp *types.SetupStatusResponse, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return nil, err
	}

	// Required AWS settings for platform to be fully configured
	requiredAwsSettings := []string{"aws_access_key", "aws_secret_key", "aws_region"}

	// Get all AWS-related platform settings
	awsSettings, err := l.svcCtx.DB.GetPlatformSettingsByCategory(l.ctx, "aws")
	if err != nil {
		l.Error("failed to get AWS platform settings:", err)
		return nil, err
	}

	// Build a map of configured AWS settings
	configuredSettings := make(map[string]bool)
	for _, setting := range awsSettings {
		// Check if the setting has a value (either text or encrypted)
		hasValue := (setting.ValueText.Valid && setting.ValueText.String != "") ||
			(setting.ValueEncrypted.Valid && setting.ValueEncrypted.String != "")
		if hasValue {
			configuredSettings[setting.Key] = true
		}
	}

	// Find missing AWS settings
	var missingSettings []string
	for _, required := range requiredAwsSettings {
		if !configuredSettings[required] {
			missingSettings = append(missingSettings, required)
		}
	}

	hasAws := len(missingSettings) == 0

	// Also check for legacy SMTP settings (deprecated)
	smtpSettings, _ := l.svcCtx.DB.GetPlatformSettingsByCategory(l.ctx, "email")
	hasSmtp := false
	for _, setting := range smtpSettings {
		if setting.Key == "smtp_host" {
			hasValue := (setting.ValueText.Valid && setting.ValueText.String != "") ||
				(setting.ValueEncrypted.Valid && setting.ValueEncrypted.String != "")
			if hasValue {
				hasSmtp = true
				break
			}
		}
	}

	return &types.SetupStatusResponse{
		HasAws:             hasAws,
		HasSmtp:            hasSmtp,
		PlatformConfigured: hasAws, // Platform is configured when AWS credentials are set
		MissingSettings:    missingSettings,
	}, nil
}
