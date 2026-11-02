package config

import (
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	Platform struct {
		Name string // Platform name for branding (e.g., "Outlet.sh")
	}
	App struct {
		BaseURL        string // Base URL for the app (e.g., https://outlet.sh)
		Domain         string // Production domain for HTTPS (e.g., outlet.sh)
		ProductionMode bool `json:",optional"` // Enable production mode (HTTPS, Let's Encrypt)
	}
	Database struct {
		Path string `json:",default=./data/outlet.db"` // SQLite database file path
	}
	Auth struct {
		AccessSecret  string
		AccessExpire  int64
		RefreshSecret string
		RefreshExpire int64
	}
	Email struct {
		// High-volume dispatcher settings (per-org SMTP config is in database)
		WorkerCount int     `json:",default=10"`  // Concurrent email workers
		RateLimit   float64 `json:",default=14"`  // Emails per second (SES limit)
		RateBurst   int     `json:",default=50"`  // Max burst size
		BatchSize   int     `json:",default=100"` // Emails per batch fetch
	}
	SMTP SMTPConfig
	Pool PoolConfig
	Encryption struct {
		Key string // 32-byte hex-encoded key for AES-256 encryption
	}
}

// Validate checks that all required configuration values are set
// Returns a list of missing/invalid configurations
func (c *Config) Validate() []string {
	var errors []string

	// Required for all environments
	if c.App.BaseURL == "" {
		errors = append(errors, "APP_BASE_URL is required")
	}

	// Validate URLs have proper format (not empty after env expansion)
	if c.App.BaseURL != "" && !strings.HasPrefix(c.App.BaseURL, "http") {
		errors = append(errors, fmt.Sprintf("APP_BASE_URL must start with http:// or https://, got: %s", c.App.BaseURL))
	}

	return errors
}

// MustValidate panics if configuration is invalid
func (c *Config) MustValidate() {
	errors := c.Validate()
	if len(errors) > 0 {
		panic(fmt.Sprintf("Configuration errors:\n  - %s", strings.Join(errors, "\n  - ")))
	}
}

// ValidateAndWarn logs warnings for missing optional config but doesn't fail
func (c *Config) ValidateAndWarn() []string {
	var warnings []string

	// Optional but recommended
	if c.Auth.AccessSecret == "" {
		warnings = append(warnings, "JWT_SECRET not set - authentication disabled")
	}

	return warnings
}

// SMTPConfig configures the SMTP ingress server
// String fields are used for env var support (empty string = default value)
type SMTPConfig struct {
	Enabled           string `json:"enabled,optional"`           // "true" to enable SMTP server
	Port              string `json:"port,optional"`              // SMTP port (default: 587)
	Domain            string `json:"domain,optional"`            // Server domain (uses App.Domain if empty)
	TLSCert           string `json:"tlscert,optional"`           // Path to TLS certificate
	TLSKey            string `json:"tlskey,optional"`            // Path to TLS private key
	MaxMessageBytes   int    `json:"maxmessagebytes,default=26214400"` // Max message size (default: 25MB)
	MaxRecipients     int    `json:"maxrecipients,default=100"`  // Max recipients per message
	AllowInsecureAuth string `json:"allowinsecureauth,optional"` // "true" to allow auth without TLS
}

// IsEnabled returns true if SMTP server should be started
func (c SMTPConfig) IsEnabled() bool {
	return strings.ToLower(c.Enabled) == "true" || c.Enabled == "1"
}

// GetPort returns the SMTP port (default: 587)
func (c SMTPConfig) GetPort() int {
	if c.Port == "" {
		return 587
	}
	port := 587
	fmt.Sscanf(c.Port, "%d", &port)
	return port
}

// IsAllowInsecureAuth returns true if insecure auth is allowed
func (c SMTPConfig) IsAllowInsecureAuth() bool {
	return strings.ToLower(c.AllowInsecureAuth) == "true" || c.AllowInsecureAuth == "1"
}

// PoolConfig configures the mailbox pool: warmed mailboxes on pool-only
// domains that a second deployment of this binary sends customers'
// outreach from. Every number lives in etc/outlet.yaml; Enabled and
// BounceDomain come from the environment so only the pool deployment
// turns it on.
type PoolConfig struct {
	Enabled      string `json:",optional"` // "true" turns the pool on (POOL_ENABLED); off, every pool route answers 503
	BounceDomain string `json:",optional"` // VERP Return-Path host: sends go out as b+<send id>@BounceDomain (POOL_BOUNCE_DOMAIN)

	// Ramp: a mailbox may send min(Ceiling, RampStart + RampStep*days since
	// its warm start) emails a UTC day, at least MinGapSeconds apart.
	RampStart     int
	RampStep      int
	Ceiling       int
	MinGapSeconds int

	// RestDays: a released mailbox goes to another tenant only after this.
	RestDays int

	// Health: a mailbox pauses when hard bounces pass BouncePausePct of its
	// last BounceWindow sends (once it has BounceMinSample), complaints pass
	// ComplaintPausePct of its last ComplaintWindow sends (once there are
	// ComplaintMinCount), receiving servers block PolicyBlocksPerDay sends in
	// a day, or its own SMTP server refuses the last ProviderFaultsToPause
	// tries. A domain pauses when DomainPauseShare of its mailboxes are
	// paused. RepeatPauses list or policy pauses in RepeatPauseDays hold the
	// tenant until an admin clears it.
	BouncePausePct        float64
	BounceWindow          int
	BounceMinSample       int
	ComplaintPausePct     float64
	ComplaintWindow       int
	ComplaintMinCount     int
	PolicyBlocksPerDay    int
	ProviderFaultsToPause int
	DomainPauseShare      float64
	RepeatPauses          int
	RepeatPauseDays       int
}

// IsEnabled reports whether the pool is on.
func (c PoolConfig) IsEnabled() bool {
	return strings.ToLower(c.Enabled) == "true" || c.Enabled == "1"
}
