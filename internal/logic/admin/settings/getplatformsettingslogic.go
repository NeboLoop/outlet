package settings

import (
	"context"
	"database/sql"
	"strings"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPlatformSettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetPlatformSettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPlatformSettingsLogic {
	return &GetPlatformSettingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetPlatformSettingsLogic) GetPlatformSettings() (resp *types.GetPlatformSettingsResponse, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return nil, err
	}

	// Initialize as empty slice so JSON returns [] not null
	settings := make([]types.PlatformSettingInfo, 0)

	// Get all settings
	dbSettings, err := l.svcCtx.DB.ListPlatformSettings(l.ctx)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	for _, s := range dbSettings {
		settings = append(settings, l.toSettingInfo(s.Key, s.ValueEncrypted, s.ValueText, s.Description, s.Category, s.IsSensitive.Valid && s.IsSensitive.Int64 == 1))
	}

	return &types.GetPlatformSettingsResponse{
		Settings: settings,
	}, nil
}

func (l *GetPlatformSettingsLogic) toSettingInfo(key string, valueEncrypted, valueText, description sql.NullString, category string, isSensitive bool) types.PlatformSettingInfo {
	var value string

	if isSensitive && valueEncrypted.Valid && valueEncrypted.String != "" {
		// Decrypt and mask sensitive values
		if l.svcCtx.CryptoService != nil {
			decrypted, err := l.svcCtx.CryptoService.DecryptString([]byte(valueEncrypted.String))
			if err == nil && decrypted != "" {
				// Mask the value - show only first 4 and last 4 characters if long enough
				if len(decrypted) > 12 {
					value = decrypted[:4] + strings.Repeat("*", len(decrypted)-8) + decrypted[len(decrypted)-4:]
				} else if len(decrypted) > 0 {
					value = strings.Repeat("*", len(decrypted))
				}
			}
		}
		if value == "" {
			value = "********" // Fallback masked value
		}
	} else if valueText.Valid {
		value = valueText.String
	}

	return types.PlatformSettingInfo{
		Key:         key,
		Value:       value,
		Category:    category,
		Description: description.String,
		IsSensitive: isSensitive,
	}
}
