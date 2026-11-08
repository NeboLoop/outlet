package email

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/outlet-sh/outlet/internal/db"
)

// OrgEmailConfig holds per-organization email configuration
type OrgEmailConfig struct {
	// AWS SES rate limiting
	SESRateLimit  float64 `json:"ses_rate_limit,omitempty"`  // Emails per second
	SESRateBurst  int     `json:"ses_rate_burst,omitempty"`  // Burst size
	SESDailyQuota int64   `json:"ses_daily_quota,omitempty"` // Daily sending quota

	// Worker configuration
	WorkerCount int `json:"worker_count,omitempty"` // Number of email worker goroutines
	BatchSize   int `json:"batch_size,omitempty"`   // Emails to fetch per batch

	// A brand's own AWS keys live in its ses sending pathway.

	// Override platform from/reply-to
	FromEmail string `json:"from_email,omitempty"`
	FromName  string `json:"from_name,omitempty"`
	ReplyTo   string `json:"reply_to,omitempty"`
}

// OrgSettings wraps the full org settings JSON structure
type OrgSettings struct {
	Email *OrgEmailConfig `json:"email,omitempty"`
}

// GetOrgEmailConfig retrieves email configuration for an organization
// Returns default config if org has no custom settings
func GetOrgEmailConfig(ctx context.Context, store *db.Store, orgID string) (*OrgEmailConfig, error) {
	org, err := store.GetOrganizationByID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	config := &OrgEmailConfig{
		// Defaults
		SESRateLimit:  14.0, // AWS SES default for new accounts
		SESRateBurst:  50,
		SESDailyQuota: 50000,
		WorkerCount:   10,
		BatchSize:     100,
	}

	// Parse settings JSON if present
	if org.Settings.Valid && org.Settings.String != "" {
		var settings OrgSettings
		if err := json.Unmarshal([]byte(org.Settings.String), &settings); err != nil {
			return nil, fmt.Errorf("failed to parse org settings: %w", err)
		}

		if settings.Email != nil {
			// Merge with defaults (only override non-zero values)
			if settings.Email.SESRateLimit > 0 {
				config.SESRateLimit = settings.Email.SESRateLimit
			}
			if settings.Email.SESRateBurst > 0 {
				config.SESRateBurst = settings.Email.SESRateBurst
			}
			if settings.Email.SESDailyQuota > 0 {
				config.SESDailyQuota = settings.Email.SESDailyQuota
			}
			if settings.Email.WorkerCount > 0 {
				config.WorkerCount = settings.Email.WorkerCount
			}
			if settings.Email.BatchSize > 0 {
				config.BatchSize = settings.Email.BatchSize
			}
			if settings.Email.FromEmail != "" {
				config.FromEmail = settings.Email.FromEmail
			}
			if settings.Email.FromName != "" {
				config.FromName = settings.Email.FromName
			}
			if settings.Email.ReplyTo != "" {
				config.ReplyTo = settings.Email.ReplyTo
			}
		}
	}

	// Also check org's direct from fields
	if config.FromEmail == "" && org.FromEmail.Valid {
		config.FromEmail = org.FromEmail.String
	}
	if config.FromName == "" && org.FromName.Valid {
		config.FromName = org.FromName.String
	}
	if config.ReplyTo == "" && org.ReplyTo.Valid {
		config.ReplyTo = org.ReplyTo.String
	}

	return config, nil
}

// SaveOrgEmailConfig saves email configuration for an organization
func SaveOrgEmailConfig(ctx context.Context, store *db.Store, orgID string, config *OrgEmailConfig) error {
	// Get current settings
	org, err := store.GetOrganizationByID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("failed to get organization: %w", err)
	}

	// Parse existing settings or create new
	var settings OrgSettings
	if org.Settings.Valid && org.Settings.String != "" {
		if err := json.Unmarshal([]byte(org.Settings.String), &settings); err != nil {
			// If parsing fails, start fresh
			settings = OrgSettings{}
		}
	}

	// Update email config
	settings.Email = config

	// Serialize back to JSON
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to serialize settings: %w", err)
	}

	// Update org settings
	err = store.UpdateOrgSettings(ctx, db.UpdateOrgSettingsParams{
		ID:       orgID,
		Settings: sql.NullString{String: string(settingsJSON), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to save org settings: %w", err)
	}

	return nil
}
