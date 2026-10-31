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

type GetPlatformSettingsByCategoryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetPlatformSettingsByCategoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPlatformSettingsByCategoryLogic {
	return &GetPlatformSettingsByCategoryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetPlatformSettingsByCategoryLogic) GetPlatformSettingsByCategory(req *types.GetPlatformSettingsByCategoryRequest) (resp *types.GetPlatformSettingsResponse, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return nil, err
	}

	// Initialize as empty slice so JSON returns [] not null
	settings := make([]types.PlatformSettingInfo, 0)

	// For AI settings, return org-level LLM credentials instead of platform_settings
	if req.Category == "ai" {
		return l.getAISettings()
	}

	// Get settings by category from platform_settings table
	dbSettings, err := l.svcCtx.DB.GetPlatformSettingsByCategory(l.ctx, req.Category)
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

// getAISettings returns global AI/LLM settings from platform_settings table
func (l *GetPlatformSettingsByCategoryLogic) getAISettings() (*types.GetPlatformSettingsResponse, error) {
	settings := make([]types.PlatformSettingInfo, 0)

	// Get AI settings from platform_settings table
	dbSettings, err := l.svcCtx.DB.GetPlatformSettingsByCategory(l.ctx, "ai")
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Check which providers have keys configured
	var claudeEnabled, openaiEnabled bool
	var hasClaudeKey, hasOpenaiKey bool
	var customProvidersValue string

	for _, s := range dbSettings {
		switch s.Key {
		case "claude_enabled":
			if s.ValueText.Valid {
				claudeEnabled = s.ValueText.String == "true"
			}
		case "claude_api_key":
			hasClaudeKey = s.ValueEncrypted.Valid && s.ValueEncrypted.String != ""
		case "openai_enabled":
			if s.ValueText.Valid {
				openaiEnabled = s.ValueText.String == "true"
			}
		case "openai_api_key":
			hasOpenaiKey = s.ValueEncrypted.Valid && s.ValueEncrypted.String != ""
		case "custom_providers":
			if s.ValueText.Valid {
				customProvidersValue = s.ValueText.String
			}
		}
	}

	// Add Claude settings
	settings = append(settings, types.PlatformSettingInfo{
		Key:         "claude_enabled",
		Value:       boolToString(claudeEnabled),
		Category:    "ai",
		Description: "Claude (Anthropic) provider enabled",
		IsSensitive: false,
	})

	if hasClaudeKey {
		settings = append(settings, types.PlatformSettingInfo{
			Key:         "claude_api_key",
			Value:       "********", // Masked - key exists
			Category:    "ai",
			Description: "Anthropic API key",
			IsSensitive: true,
		})
	}

	// Add OpenAI settings
	settings = append(settings, types.PlatformSettingInfo{
		Key:         "openai_enabled",
		Value:       boolToString(openaiEnabled),
		Category:    "ai",
		Description: "OpenAI provider enabled",
		IsSensitive: false,
	})

	if hasOpenaiKey {
		settings = append(settings, types.PlatformSettingInfo{
			Key:         "openai_api_key",
			Value:       "********", // Masked - key exists
			Category:    "ai",
			Description: "OpenAI API key",
			IsSensitive: true,
		})
	}

	// Add custom providers if configured
	if customProvidersValue != "" {
		settings = append(settings, types.PlatformSettingInfo{
			Key:         "custom_providers",
			Value:       customProvidersValue,
			Category:    "ai",
			Description: "Custom AI providers configuration",
			IsSensitive: true,
		})
	}

	return &types.GetPlatformSettingsResponse{Settings: settings}, nil
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func (l *GetPlatformSettingsByCategoryLogic) toSettingInfo(key string, valueEncrypted, valueText, description sql.NullString, category string, isSensitive bool) types.PlatformSettingInfo {
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
