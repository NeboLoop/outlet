package organizations

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateOrganizationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateOrganizationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateOrganizationLogic {
	return &CreateOrganizationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateOrganizationLogic) CreateOrganization(req *types.CreateOrgRequest) (resp *types.OrgInfo, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return nil, err
	}

	// Generate API key
	apiKeyBytes := make([]byte, 32)
	if _, err := rand.Read(apiKeyBytes); err != nil {
		return nil, err
	}
	apiKey := hex.EncodeToString(apiKeyBytes)

	// Generate org ID
	orgID := uuid.New().String()

	// Only set max_contacts if explicitly provided (otherwise NULL = unlimited)
	maxContacts := sql.NullInt64{Valid: false}
	if req.MaxContacts > 0 {
		maxContacts = sql.NullInt64{Int64: int64(req.MaxContacts), Valid: true}
	}

	org, err := l.svcCtx.DB.CreateOrganization(l.ctx, db.CreateOrganizationParams{
		ID:          orgID,
		Name:        req.Name,
		Slug:        req.Slug,
		ApiKey:      apiKey,
		MaxContacts: maxContacts,
		Settings:    sql.NullString{String: "{}", Valid: true},
		AppUrl:      sql.NullString{String: req.AppUrl, Valid: req.AppUrl != ""},
	})
	if err != nil {
		return nil, err
	}

	// Update email settings if provided
	if req.FromName != "" || req.FromEmail != "" || req.ReplyTo != "" {
		org, err = l.svcCtx.DB.UpdateOrgEmailSettings(l.ctx, db.UpdateOrgEmailSettingsParams{
			ID:        org.ID,
			FromName:  sql.NullString{String: req.FromName, Valid: req.FromName != ""},
			FromEmail: sql.NullString{String: req.FromEmail, Valid: req.FromEmail != ""},
			ReplyTo:   sql.NullString{String: req.ReplyTo, Valid: req.ReplyTo != ""},
		})
		if err != nil {
			l.Errorf("Failed to set email settings: %v", err)
			// Don't fail the whole operation, org was created successfully
		}
	}

	// Auto-setup domain identity if email is configured
	if org.FromEmail.Valid && org.FromEmail.String != "" {
		l.setupDomainIdentity(org.ID, org.FromEmail.String)
	}

	// Seed default email designs for the org
	l.seedDefaultDesigns(org.ID, req.Name)

	return &types.OrgInfo{
		Id:               org.ID,
		Name:             org.Name,
		Slug:             org.Slug,
		ApiKey:           org.ApiKey,
		BillingStatus:    "trial",
		Plan:             "starter",
		MaxContacts:      int(org.MaxContacts.Int64),
		StripeConfigured: false,
		FromName:         org.FromName.String,
		FromEmail:        org.FromEmail.String,
		ReplyTo:          org.ReplyTo.String,
		AppUrl:           org.AppUrl.String,
		CreatedAt:        org.CreatedAt.String,
	}, nil
}

// seedDefaultDesigns creates starter email designs for a new organization
func (l *CreateOrganizationLogic) seedDefaultDesigns(orgID string, orgName string) {
	// Simple design - minimal footer only
	simpleHTML := `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="margin: 0; padding: 0; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background-color: #f9fafb;">
  <div style="max-width: 600px; margin: 0 auto; padding: 40px 20px;">
    {{content}}

    <div style="margin-top: 40px; padding-top: 20px; border-top: 1px solid #e5e7eb; text-align: center; color: #6b7280; font-size: 12px;">
      <p style="margin: 0;">` + orgName + `</p>
      <p style="margin: 8px 0 0 0;">
        <a href="{{unsubscribe_url}}" style="color: #6b7280;">Unsubscribe</a>
      </p>
    </div>
  </div>
</body>
</html>`

	// Branded design - header + footer with branding
	brandedHTML := `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="margin: 0; padding: 0; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background-color: #f9fafb;">
  <div style="max-width: 600px; margin: 0 auto;">
    <!-- Header -->
    <div style="background-color: #1f2937; padding: 24px 20px; text-align: center;">
      <h1 style="margin: 0; color: #ffffff; font-size: 24px; font-weight: 600;">` + orgName + `</h1>
    </div>

    <!-- Content -->
    <div style="background-color: #ffffff; padding: 40px 20px;">
      {{content}}
    </div>

    <!-- Footer -->
    <div style="background-color: #f3f4f6; padding: 24px 20px; text-align: center; color: #6b7280; font-size: 12px;">
      <p style="margin: 0; font-weight: 500;">` + orgName + `</p>
      <p style="margin: 12px 0 0 0;">
        <a href="{{unsubscribe_url}}" style="color: #6b7280;">Unsubscribe</a> ·
        <a href="{{preferences_url}}" style="color: #6b7280;">Email Preferences</a>
      </p>
    </div>
  </div>
</body>
</html>`

	// Create Simple design
	_, err := l.svcCtx.DB.CreateEmailDesign(l.ctx, db.CreateEmailDesignParams{
		OrgID:       orgID,
		Name:        "Simple",
		Slug:        "simple",
		Description: sql.NullString{String: "Minimal design with footer only", Valid: true},
		Category:    sql.NullString{String: "general", Valid: true},
		HtmlBody:    simpleHTML,
		PlainText:   sql.NullString{String: "{{content}}\n\n---\n" + orgName + "\nUnsubscribe: {{unsubscribe_url}}", Valid: true},
		IsActive:    sql.NullInt64{Int64: 1, Valid: true},
	})
	if err != nil {
		l.Errorf("Failed to seed simple design: %v", err)
	}

	// Create Branded design
	_, err = l.svcCtx.DB.CreateEmailDesign(l.ctx, db.CreateEmailDesignParams{
		OrgID:       orgID,
		Name:        "Branded",
		Slug:        "branded",
		Description: sql.NullString{String: "Full branded design with header and footer", Valid: true},
		Category:    sql.NullString{String: "general", Valid: true},
		HtmlBody:    brandedHTML,
		PlainText:   sql.NullString{String: "=== " + orgName + " ===\n\n{{content}}\n\n---\n" + orgName + "\nUnsubscribe: {{unsubscribe_url}}", Valid: true},
		IsActive:    sql.NullInt64{Int64: 1, Valid: true},
	})
	if err != nil {
		l.Errorf("Failed to seed branded design: %v", err)
	}
}

// setupDomainIdentity auto-creates a domain identity in AWS SES for the organization
func (l *CreateOrganizationLogic) setupDomainIdentity(orgID string, fromEmail string) {
	// Extract domain from email
	domain := email.ExtractDomainFromEmail(fromEmail)
	if domain == "" {
		l.Errorf("Could not extract domain from email: %s", fromEmail)
		return
	}

	// Get AWS credentials from platform settings
	region, accessKey, secretKey, err := l.getAWSCredentials()
	if err != nil {
		l.Infof("AWS credentials not configured, skipping domain identity setup: %v", err)
		return
	}

	// Check if domain identity already exists
	existing, err := l.svcCtx.DB.GetDomainIdentityByDomain(l.ctx, db.GetDomainIdentityByDomainParams{
		OrgID:  orgID,
		Domain: domain,
	})
	if err == nil && existing.ID != "" {
		l.Infof("Domain identity already exists for domain %s", domain)
		return
	}

	// Verify domain identity with AWS SES
	result, err := email.VerifyDomainIdentity(l.ctx, region, accessKey, secretKey, domain)
	if err != nil {
		l.Errorf("Failed to verify domain identity with AWS SES: %v", err)
		return
	}

	// Convert DNS records to JSON
	dnsRecordsJSON, err := email.DNSRecordsToJSON(result.DNSRecords)
	if err != nil {
		l.Errorf("Failed to serialize DNS records: %v", err)
		return
	}

	// Convert DKIM tokens to JSON
	dkimTokensJSON, err := json.Marshal(result.DKIMTokens)
	if err != nil {
		l.Errorf("Failed to serialize DKIM tokens: %v", err)
		return
	}

	// Create domain identity record
	_, err = l.svcCtx.DB.CreateDomainIdentity(l.ctx, db.CreateDomainIdentityParams{
		ID:                 uuid.New().String(),
		OrgID:              orgID,
		Domain:             domain,
		VerificationStatus: sql.NullString{String: result.VerificationStatus, Valid: true},
		DkimStatus:         sql.NullString{String: result.DKIMStatus, Valid: true},
		VerificationToken:  sql.NullString{String: result.VerificationToken, Valid: result.VerificationToken != ""},
		DkimTokens:         sql.NullString{String: string(dkimTokensJSON), Valid: true},
		DnsRecords:         sql.NullString{String: dnsRecordsJSON, Valid: true},
	})
	if err != nil {
		l.Errorf("Failed to create domain identity record: %v", err)
		return
	}

	l.Infof("Domain identity created for %s", domain)
}

// getAWSCredentials retrieves AWS credentials from platform settings
func (l *CreateOrganizationLogic) getAWSCredentials() (region, accessKey, secretKey string, err error) {
	awsSettings, err := l.svcCtx.DB.GetPlatformSettingsByCategory(l.ctx, "aws")
	if err != nil {
		return "", "", "", err
	}

	region = "us-east-1" // default

	for _, setting := range awsSettings {
		switch setting.Key {
		case "aws_access_key":
			if setting.ValueEncrypted.Valid && setting.ValueEncrypted.String != "" {
				if l.svcCtx.CryptoService != nil {
					decrypted, decErr := l.svcCtx.CryptoService.DecryptString([]byte(setting.ValueEncrypted.String))
					if decErr != nil {
						return "", "", "", decErr
					}
					accessKey = decrypted
				}
			}
		case "aws_secret_key":
			if setting.ValueEncrypted.Valid && setting.ValueEncrypted.String != "" {
				if l.svcCtx.CryptoService != nil {
					decrypted, decErr := l.svcCtx.CryptoService.DecryptString([]byte(setting.ValueEncrypted.String))
					if decErr != nil {
						return "", "", "", decErr
					}
					secretKey = decrypted
				}
			}
		case "aws_region":
			if setting.ValueText.Valid && setting.ValueText.String != "" {
				region = setting.ValueText.String
			}
		}
	}

	if accessKey == "" || secretKey == "" {
		return "", "", "", sql.ErrNoRows
	}

	return region, accessKey, secretKey, nil
}

