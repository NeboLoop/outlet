package config

import (
	"encoding/hex"
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
	SMTP       SMTPConfig
	Validation ValidationConfig `json:",optional"`
	SignalNet  SignalNetConfig  `json:",optional"`
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

	// Admin logins are signed with these; an empty secret means anyone can
	// forge a login
	if c.Auth.AccessSecret == "" {
		errors = append(errors, "JWT_SECRET is required (try: openssl rand -hex 32)")
	}
	if c.Auth.RefreshSecret == "" {
		errors = append(errors, "JWT_REFRESH_SECRET is required (try: openssl rand -hex 32)")
	}

	// Email provider credentials are stored encrypted with this key
	if key, err := hex.DecodeString(c.Encryption.Key); err != nil || len(key) != 32 {
		errors = append(errors, "ENCRYPTION_KEY is required: 64 hex characters (try: openssl rand -hex 32)")
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

	if c.Auth.AccessSecret != "" && len(c.Auth.AccessSecret) < 32 {
		warnings = append(warnings, "JWT_SECRET is shorter than 32 characters; use a longer random value (openssl rand -hex 32)")
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

// ValidationConfig configures email validation probing.
type ValidationConfig struct {
	Proxies      []string `json:",optional"`             // socks5:// URLs of dedicated probe hosts; empty = probe directly
	HeloHostname string   `json:",optional"`             // EHLO name; must match the probe IPs' reverse DNS
	FromDomain   string   `json:",optional"`             // MAIL FROM domain (defaults to HeloHostname)
	SMTPCheck    bool     `json:",optional"`             // enable mailbox probing (needs outbound port 25 or Proxies)
	Concurrency  int      `json:",default=20"`           // domains probed in parallel by the job worker
	CacheDays    int      `json:",default=30"`           // reuse a probe result for this many days
}

// SignalNetConfig configures the Outlet Signal Network: pooled, hashed deliverability
// signals shared across instances. The hub is run by outlet.sh; an instance key is the subscription.
type SignalNetConfig struct {
	HubURL      string `json:",optional"` // e.g. https://signals.outlet.sh; empty = disabled
	InstanceKey string `json:",optional"` // subscription key issued by the hub
	Hub         bool   `json:",optional"` // run this instance as the hub (serves /network/v1)
}
