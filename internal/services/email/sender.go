package email

import (
	"context"
	"fmt"
	"net/mail"
	"net/smtp"
	"strconv"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/crypto"
)

// SMTPConfig holds SMTP configuration loaded from database
type SMTPConfig struct {
	Host        string
	Port        int
	User        string
	Password    string
	FromAddress string
	FromName    string
	ReplyTo     string
}

// Service handles email sending via SMTP or AWS SES
// Configuration is loaded dynamically from the database (platform_settings + org settings)
type Service struct {
	db           *db.Store
	baseURL      string
	cryptoSvc    *crypto.Service

	// Connection pool for high-throughput sending (optional)
	pool        *SMTPPool
	poolEnabled bool
}

// NewService creates a new email service that loads SMTP config from database
func NewService(store *db.Store, cryptoSvc *crypto.Service) *Service {
	return &Service{
		db:        store,
		cryptoSvc: cryptoSvc,
		baseURL:   "", // Set via SetBaseURL from config
	}
}

// getSESConfig loads AWS SES configuration from platform_settings
// AWS credentials are in category 'aws', email settings in category 'email'
func (s *Service) getSESConfig(ctx context.Context) (*SESConfig, error) {
	config := &SESConfig{}

	// Get AWS credentials from 'aws' category
	awsSettings, err := s.db.GetPlatformSettingsByCategory(ctx, "aws")
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS settings: %w", err)
	}

	for _, setting := range awsSettings {
		switch setting.Key {
		case "aws_access_key":
			if setting.ValueEncrypted.Valid && setting.ValueEncrypted.String != "" && s.cryptoSvc != nil {
				decrypted, err := s.cryptoSvc.DecryptString([]byte(setting.ValueEncrypted.String))
				if err == nil {
					config.AccessKey = decrypted
				}
			}
		case "aws_secret_key":
			if setting.ValueEncrypted.Valid && setting.ValueEncrypted.String != "" && s.cryptoSvc != nil {
				decrypted, err := s.cryptoSvc.DecryptString([]byte(setting.ValueEncrypted.String))
				if err == nil {
					config.SecretKey = decrypted
				}
			}
		case "aws_region":
			if setting.ValueText.Valid {
				config.Region = setting.ValueText.String
			}
		}
	}

	// Get email settings from 'email' category (from_email, from_name, reply_to)
	emailSettings, _ := s.db.GetPlatformSettingsByCategory(ctx, "email")
	for _, setting := range emailSettings {
		switch setting.Key {
		case "from_email":
			if setting.ValueText.Valid {
				config.FromAddress = setting.ValueText.String
			}
		case "from_name":
			if setting.ValueText.Valid {
				config.FromName = setting.ValueText.String
			}
		case "reply_to":
			if setting.ValueText.Valid {
				config.ReplyTo = setting.ValueText.String
			}
		}
	}

	return config, nil
}

// hasSESConfig checks if AWS SES credentials are configured
// Note: FromAddress is typically provided by the caller or org settings, not required here
func (s *Service) hasSESConfig(sesConfig *SESConfig) bool {
	return sesConfig.AccessKey != "" && sesConfig.SecretKey != ""
}

// SetBaseURL sets the base URL for tracking links
func (s *Service) SetBaseURL(url string) {
	s.baseURL = url
}

// GetBaseURL returns the base URL for tracking links
func (s *Service) GetBaseURL() string {
	return s.baseURL
}

// getGlobalSMTPConfig loads SMTP configuration from platform_settings (category: 'email')
func (s *Service) getGlobalSMTPConfig(ctx context.Context) (*SMTPConfig, error) {
	settings, err := s.db.GetPlatformSettingsByCategory(ctx, "email")
	if err != nil {
		return nil, fmt.Errorf("failed to get email settings: %w", err)
	}

	config := &SMTPConfig{}
	for _, setting := range settings {
		// Use value_text for non-sensitive, value_encrypted needs decryption (handled at API layer)
		value := ""
		if setting.ValueText.Valid {
			value = setting.ValueText.String
		}

		switch setting.Key {
		case "smtp_host":
			config.Host = value
		case "smtp_port":
			if port, err := strconv.Atoi(value); err == nil {
				config.Port = port
			}
		case "smtp_user":
			config.User = value
		case "smtp_password":
			// Password should be in value_encrypted, but for simplicity use value_text if available
			if setting.ValueEncrypted.Valid && setting.ValueEncrypted.String != "" {
				config.Password = setting.ValueEncrypted.String
			} else {
				config.Password = value
			}
		case "from_email":
			config.FromAddress = value
		case "from_name":
			config.FromName = value
		case "reply_to":
			config.ReplyTo = value
		}
	}

	return config, nil
}

// EnablePool enables SMTP connection pooling for high-throughput sending
// Must be called after platform settings are configured in the database
func (s *Service) EnablePool(ctx context.Context, poolSize int) error {
	config, err := s.getGlobalSMTPConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get SMTP config: %w", err)
	}

	if config.Host == "" {
		return fmt.Errorf("SMTP not configured - set smtp_host in platform settings")
	}

	poolConfig := SMTPPoolConfig{
		Host:     config.Host,
		Port:     config.Port,
		User:     config.User,
		Pass:     config.Password,
		UseTLS:   config.Port == 465 || config.Port == 587,
		PoolSize: poolSize,
		MaxIdle:  5 * time.Minute,
	}

	s.pool = NewSMTPPool(poolConfig)
	s.poolEnabled = true
	return nil
}

// ClosePool closes the connection pool
func (s *Service) ClosePool() {
	if s.pool != nil {
		s.pool.Close()
		s.pool = nil
		s.poolEnabled = false
	}
}

// PoolStats returns connection pool statistics
func (s *Service) PoolStats() (pooled, created int) {
	if s.pool != nil {
		return s.pool.Stats()
	}
	return 0, 0
}

// sendEmail sends an HTML email via AWS SES (preferred) or SMTP (fallback)
// Loads config from platform_settings database
func (s *Service) sendEmail(to, subject, htmlBody string) error {
	ctx := context.Background()

	// Try AWS SES first (preferred for high-volume sending)
	sesConfig, err := s.getSESConfig(ctx)
	if err == nil && s.hasSESConfig(sesConfig) {
		return SendEmailViaSES(ctx, sesConfig, to, subject, htmlBody)
	}

	// Fall back to SMTP if SES not configured
	smtpConfig, err := s.getGlobalSMTPConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get email config: %w", err)
	}

	if smtpConfig.Host == "" || smtpConfig.User == "" || smtpConfig.Password == "" {
		// Neither SES nor SMTP configured
		return fmt.Errorf("email not configured - set AWS SES credentials or SMTP settings in platform settings")
	}

	// Build email message for SMTP
	headers := fmt.Sprintf("From: %s <%s>\r\n", smtpConfig.FromName, smtpConfig.FromAddress)
	headers += fmt.Sprintf("To: %s\r\n", to)
	if smtpConfig.ReplyTo != "" {
		headers += fmt.Sprintf("Reply-To: %s\r\n", smtpConfig.ReplyTo)
	}
	headers += fmt.Sprintf("Subject: %s\r\n", subject)
	headers += "MIME-Version: 1.0\r\n"
	headers += "Content-Type: text/html; charset=UTF-8\r\n"
	headers += "\r\n"

	message := []byte(headers + htmlBody)

	// SMTP auth
	auth := smtp.PlainAuth("", smtpConfig.User, smtpConfig.Password, smtpConfig.Host)

	addr := fmt.Sprintf("%s:%d", smtpConfig.Host, smtpConfig.Port)
	err = smtp.SendMail(addr, auth, smtpConfig.FromAddress, []string{to}, message)
	if err != nil {
		return fmt.Errorf("failed to send email via SMTP: %w", err)
	}

	return nil
}

// SendConfirmationEmail sends a meeting confirmation email to the lead
func (s *Service) SendConfirmationEmail(ctx context.Context, toEmail, toName string, meetingTime time.Time, meetURL, timezone string) error {
	subject := "Your Outlet Consultation is Confirmed"

	// Get SMTP config for from address
	config, err := s.getGlobalSMTPConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get SMTP config: %w", err)
	}

	contactEmail := config.FromAddress
	if contactEmail == "" {
		return fmt.Errorf("from_email not configured in platform settings")
	}

	// Format meeting time in the attendee's timezone
	loc, _ := time.LoadLocation(timezone)
	localTime := meetingTime.In(loc)

	body := fmt.Sprintf(`
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 30px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #f9fafb; padding: 30px; border-radius: 0 0 8px 8px; }
        .info-box { background: white; padding: 20px; border-radius: 8px; margin: 20px 0; border-left: 4px solid #667eea; }
        .button { display: inline-block; background: #667eea; color: white; padding: 12px 30px; text-decoration: none; border-radius: 6px; margin: 10px 0; }
        .footer { text-align: center; color: #6b7280; font-size: 12px; margin-top: 30px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Meeting Confirmed!</h1>
        </div>
        <div class="content">
            <p>Hi %s,</p>
            <p>Your consultation with Outlet is confirmed. We're excited to discuss how AI can transform your business!</p>

            <div class="info-box">
                <h3 style="margin-top: 0;">Meeting Details</h3>
                <p><strong>Date & Time:</strong> %s</p>
                <p><strong>Duration:</strong> 25 minutes</p>
                <p><strong>Timezone:</strong> %s</p>
                %s
            </div>

            <p><strong>What to Expect:</strong></p>
            <ul>
                <li>We'll discuss your current business challenges</li>
                <li>Explore potential AI opportunities</li>
                <li>Outline a preliminary roadmap for implementation</li>
            </ul>

            <p>A calendar invite has been sent to this email address. You'll also receive a reminder 24 hours before the meeting.</p>

            <p>If you need to reschedule, please contact us at %s.</p>

            <p>Best regards,<br>The Outlet Team</p>
        </div>
        <div class="footer">
            <p>Outlet - Turning AI Confusion into Competitive Advantage</p>
        </div>
    </div>
</body>
</html>
	`, toName,
		localTime.Format("Monday, January 2, 2006 at 3:04 PM MST"),
		timezone,
		getMeetLinkHTML(meetURL),
		contactEmail)

	return s.sendEmail(toEmail, subject, body)
}

// SendAgentNotificationEmail sends a notification to the sales agent
func (s *Service) SendAgentNotificationEmail(ctx context.Context, agentEmail, attendeeName, attendeeEmail, attendeePhone string, meetingTime time.Time, meetURL, timezone, notes string) error {
	subject := fmt.Sprintf("New Meeting Booked: %s", attendeeName)

	body := fmt.Sprintf(`
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #1f2937; color: white; padding: 20px; text-align: center; }
        .content { background: #f9fafb; padding: 30px; }
        .info-box { background: white; padding: 20px; border-radius: 8px; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>New Meeting Scheduled</h2>
        </div>
        <div class="content">
            <div class="info-box">
                <h3>Attendee Information</h3>
                <p><strong>Name:</strong> %s</p>
                <p><strong>Email:</strong> %s</p>
                <p><strong>Phone:</strong> %s</p>
            </div>

            <div class="info-box">
                <h3>Meeting Details</h3>
                <p><strong>Date & Time:</strong> %s</p>
                <p><strong>Timezone:</strong> %s</p>
                <p><strong>Duration:</strong> 25 minutes</p>
                %s
            </div>

            %s
        </div>
    </div>
</body>
</html>
	`, attendeeName,
		attendeeEmail,
		getStringOrDefault(attendeePhone, "Not provided"),
		meetingTime.Format("Monday, January 2, 2006 at 3:04 PM MST"),
		timezone,
		getMeetLinkHTML(meetURL),
		getNotesHTML(notes))

	return s.sendEmail(agentEmail, subject, body)
}

func getMeetLinkHTML(meetURL string) string {
	if meetURL != "" {
		return fmt.Sprintf(`<p><strong>Google Meet Link:</strong> <a href="%s">%s</a></p>`, meetURL, meetURL)
	}
	return ""
}

func getNotesHTML(notes string) string {
	if notes != "" {
		return fmt.Sprintf(`<div class="info-box"><h3>Additional Notes</h3><p>%s</p></div>`, notes)
	}
	return ""
}

func getStringOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

// SendEmail sends an HTML email to a single recipient
func (s *Service) SendEmail(ctx context.Context, to, subject, htmlBody string) error {
	return s.sendEmail(to, subject, htmlBody)
}

// SendEmailFrom sends an HTML email with a custom from address
// Uses AWS SES (preferred) or falls back to SMTP
func (s *Service) SendEmailFrom(ctx context.Context, fromEmail, fromName, to, subject, htmlBody string, opts ...SendOption) error {
	o := applyOpts(opts)

	// Try AWS SES first (preferred for high-volume sending)
	sesConfig, err := s.getSESConfig(ctx)
	if err == nil && s.hasSESConfig(sesConfig) {
		// Override from address if provided
		if fromEmail != "" {
			sesConfig.FromAddress = fromEmail
		}
		if fromName != "" {
			sesConfig.FromName = fromName
		}
		if o.replyTo != "" {
			sesConfig.ReplyTo = o.replyTo
		}
		if o.needsRaw() {
			o.replyTo = sesConfig.ReplyTo // the brand default when none was given
			raw, err := buildRawMessage(formatFrom(sesConfig.FromName, sesConfig.FromAddress), to, subject, htmlBody, o)
			if err != nil {
				return fmt.Errorf("build raw message: %w", err)
			}
			return SendRawEmailViaSES(ctx, sesConfig, raw)
		}
		return SendEmailViaSES(ctx, sesConfig, to, subject, htmlBody)
	}

	// Fall back to SMTP
	smtpConfig, err := s.getGlobalSMTPConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get email config: %w", err)
	}

	if smtpConfig.Host == "" || smtpConfig.User == "" || smtpConfig.Password == "" {
		return fmt.Errorf("email not configured - set AWS SES credentials or SMTP settings in platform settings")
	}

	// Use provided from or fall back to defaults
	from := fromEmail
	name := fromName
	if from == "" {
		from = smtpConfig.FromAddress
	}
	if name == "" {
		name = smtpConfig.FromName
	}

	auth := smtp.PlainAuth("", smtpConfig.User, smtpConfig.Password, smtpConfig.Host)
	addr := fmt.Sprintf("%s:%d", smtpConfig.Host, smtpConfig.Port)

	if o.needsRaw() || o.replyTo != "" {
		if o.replyTo == "" {
			o.replyTo = smtpConfig.ReplyTo
		}
		raw, err := buildRawMessage(formatFrom(name, from), to, subject, htmlBody, o)
		if err != nil {
			return fmt.Errorf("build raw message: %w", err)
		}
		return smtp.SendMail(addr, auth, from, []string{to}, raw)
	}

	headers := fmt.Sprintf("From: %s <%s>\r\n", name, from)
	headers += fmt.Sprintf("To: %s\r\n", to)
	if smtpConfig.ReplyTo != "" {
		headers += fmt.Sprintf("Reply-To: %s\r\n", smtpConfig.ReplyTo)
	}
	headers += fmt.Sprintf("Subject: %s\r\n", subject)
	headers += "MIME-Version: 1.0\r\n"
	headers += "Content-Type: text/html; charset=UTF-8\r\n"
	headers += "\r\n"

	message := []byte(headers + htmlBody)

	return smtp.SendMail(addr, auth, from, []string{to}, message)
}

// formatFrom writes a From header value, encoding a display name that
// isn't plain ASCII (RFC 2047) so a name like "Nanna · Alma's Nebo" survives.
func formatFrom(name, addr string) string {
	if name == "" {
		return addr
	}
	return (&mail.Address{Name: name, Address: addr}).String()
}

// SendCampaignEmail sends a campaign email with custom from/reply-to
// Uses AWS SES (preferred), SMTP connection pooling, or standard SMTP as fallback
func (s *Service) SendCampaignEmail(to, subject, htmlBody, fromName, fromEmail, replyTo string) error {
	ctx := context.Background()

	// Try AWS SES first (preferred for high-volume sending)
	sesConfig, err := s.getSESConfig(ctx)
	if err == nil && s.hasSESConfig(sesConfig) {
		// Override from address if provided
		if fromEmail != "" {
			sesConfig.FromAddress = fromEmail
		}
		if fromName != "" {
			sesConfig.FromName = fromName
		}
		if replyTo != "" {
			sesConfig.ReplyTo = replyTo
		}
		return SendEmailViaSES(ctx, sesConfig, to, subject, htmlBody)
	}

	// Fall back to SMTP
	smtpConfig, err := s.getGlobalSMTPConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get email config: %w", err)
	}

	if smtpConfig.Host == "" || smtpConfig.User == "" || smtpConfig.Password == "" {
		return fmt.Errorf("email not configured - set AWS SES credentials or SMTP settings in platform settings")
	}

	// Use provided or fall back to defaults
	if fromEmail == "" {
		fromEmail = smtpConfig.FromAddress
	}
	if fromName == "" {
		fromName = smtpConfig.FromName
	}
	if replyTo == "" {
		replyTo = smtpConfig.ReplyTo
	}

	headers := fmt.Sprintf("From: %s <%s>\r\n", fromName, fromEmail)
	headers += fmt.Sprintf("To: %s\r\n", to)
	if replyTo != "" {
		headers += fmt.Sprintf("Reply-To: %s\r\n", replyTo)
	}
	headers += fmt.Sprintf("Subject: %s\r\n", subject)
	headers += "MIME-Version: 1.0\r\n"
	headers += "Content-Type: text/html; charset=UTF-8\r\n"
	headers += "\r\n"

	message := []byte(headers + htmlBody)

	// Use pooled SMTP connection if available
	if s.poolEnabled && s.pool != nil {
		return s.pool.SendWithPool(fromEmail, []string{to}, message)
	}

	// Fallback to standard SMTP sending
	auth := smtp.PlainAuth("", smtpConfig.User, smtpConfig.Password, smtpConfig.Host)
	addr := fmt.Sprintf("%s:%d", smtpConfig.Host, smtpConfig.Port)

	return smtp.SendMail(addr, auth, fromEmail, []string{to}, message)
}

// GetTrackingPixelURL returns the URL for a tracking pixel
func (s *Service) GetTrackingPixelURL(trackingToken string) string {
	return fmt.Sprintf("%s/api/e/o/%s", s.baseURL, trackingToken)
}

// RewriteLinksForTracking replaces href URLs with tracked redirect URLs
func (s *Service) RewriteLinksForTracking(htmlBody, trackingToken string) string {
	// Simple link rewriting using string replacement
	// This is a basic implementation - for production, consider using a proper HTML parser

	result := htmlBody
	searchStart := 0

	for {
		// Find next href="
		hrefIdx := findNextHref(result, searchStart)
		if hrefIdx == -1 {
			break
		}

		// Find the URL start (after href=")
		urlStart := hrefIdx + 6 // len(`href="`)
		if urlStart >= len(result) {
			break
		}

		// Find the closing quote
		urlEnd := findClosingQuote(result, urlStart)
		if urlEnd == -1 {
			searchStart = urlStart
			continue
		}

		originalURL := result[urlStart:urlEnd]

		// Skip special URLs
		if shouldSkipURL(originalURL) {
			searchStart = urlEnd
			continue
		}

		// Build tracked URL
		trackedURL := fmt.Sprintf("%s/api/e/c/%s?url=%s",
			s.baseURL,
			trackingToken,
			urlEncode(originalURL),
		)

		// Replace the URL
		result = result[:urlStart] + trackedURL + result[urlEnd:]
		searchStart = urlStart + len(trackedURL)
	}

	return result
}

func findNextHref(s string, start int) int {
	for i := start; i < len(s)-5; i++ {
		if s[i:i+6] == `href="` {
			return i
		}
	}
	return -1
}

func findClosingQuote(s string, start int) int {
	for i := start; i < len(s); i++ {
		if s[i] == '"' {
			return i
		}
	}
	return -1
}

func shouldSkipURL(url string) bool {
	return len(url) > 7 && url[:7] == "mailto:" ||
		len(url) > 4 && url[:4] == "tel:" ||
		len(url) > 0 && url[0] == '#' ||
		len(url) > 2 && url[:2] == "{{" ||
		containsSubstring(url, "/api/e/c/")
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func urlEncode(s string) string {
	var result []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isAlphaNum(c) || c == '-' || c == '_' || c == '.' || c == '~' {
			result = append(result, c)
		} else {
			result = append(result, '%')
			result = append(result, hexDigit(c>>4))
			result = append(result, hexDigit(c&0xf))
		}
	}
	return string(result)
}

func isAlphaNum(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func hexDigit(n byte) byte {
	if n < 10 {
		return '0' + n
	}
	return 'A' + n - 10
}
