package emailconfig

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateDomainIdentityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateDomainIdentityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateDomainIdentityLogic {
	return &CreateDomainIdentityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateDomainIdentityLogic) CreateDomainIdentity(req *types.CreateDomainIdentityRequest) (resp *types.DomainIdentityInfo, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, req.OrgId, access.Admin); err != nil {
		return nil, err
	}

	// Get the organization to get its email settings
	org, err := l.svcCtx.DB.GetOrganizationByID(l.ctx, req.OrgId)
	if err != nil {
		return nil, errorx.NewNotFoundError("Organization not found")
	}

	// Determine the domain to verify
	domain := req.Domain
	if domain == "" {
		// Extract from org's from_email
		if !org.FromEmail.Valid || org.FromEmail.String == "" {
			return nil, errorx.NewBadRequestError("No domain provided and organization has no from_email configured")
		}
		domain = email.ExtractDomainFromEmail(org.FromEmail.String)
		if domain == "" {
			return nil, errorx.NewBadRequestError("Could not extract domain from email address")
		}
	}

	// Check if domain identity already exists for this org
	existing, err := l.svcCtx.DB.GetDomainIdentityByDomain(l.ctx, db.GetDomainIdentityByDomainParams{
		OrgID:  req.OrgId,
		Domain: domain,
	})
	if err == nil && existing.ID != "" {
		return nil, errorx.NewBadRequestError("Domain identity already exists for this domain")
	}
	if err := email.ClaimDomain(l.ctx, l.svcCtx.DB, req.OrgId, domain); err != nil {
		return nil, err
	}

	// Get AWS credentials from platform settings
	region, accessKey, secretKey, err := getAWSCredentials(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}

	// Get email config for org-specific AWS credentials
	emailConfig, err := email.GetOrgEmailConfig(l.ctx, l.svcCtx.DB, req.OrgId)
	if err == nil && emailConfig.HasOwnAWSCredentials() {
		// Use org's own credentials
		region = emailConfig.AWSRegion
		accessKey = emailConfig.AWSAccessKey
		secretKey = emailConfig.AWSSecretKey
	}

	// Determine MAIL FROM subdomain (default to "mail" if not specified)
	mailFromSubdomain := req.MailFromSubdomain
	if mailFromSubdomain == "" {
		mailFromSubdomain = "mail"
	}

	// Verify domain identity with AWS SES
	result, err := email.VerifyDomainIdentity(l.ctx, region, accessKey, secretKey, domain, mailFromSubdomain)
	if err != nil {
		l.Errorf("Failed to verify domain identity: %v", err)
		return nil, errorx.NewInternalError("Failed to verify domain identity with AWS SES: " + err.Error())
	}

	// Compute the full MAIL FROM domain
	mailFromDomain := mailFromSubdomain + "." + domain

	// Convert DNS records to JSON
	dnsRecordsJSON, err := email.DNSRecordsToJSON(result.DNSRecords)
	if err != nil {
		l.Errorf("Failed to serialize DNS records: %v", err)
		return nil, errorx.NewInternalError("Failed to serialize DNS records")
	}

	// Convert DKIM tokens to JSON
	dkimTokensJSON, err := json.Marshal(result.DKIMTokens)
	if err != nil {
		l.Errorf("Failed to serialize DKIM tokens: %v", err)
		return nil, errorx.NewInternalError("Failed to serialize DKIM tokens")
	}

	// Create domain identity record
	identity, err := l.svcCtx.DB.CreateDomainIdentity(l.ctx, db.CreateDomainIdentityParams{
		ID:                 uuid.New().String(),
		OrgID:              req.OrgId,
		Domain:             domain,
		VerificationStatus: sql.NullString{String: result.VerificationStatus, Valid: true},
		DkimStatus:         sql.NullString{String: result.DKIMStatus, Valid: true},
		VerificationToken:  sql.NullString{String: result.VerificationToken, Valid: result.VerificationToken != ""},
		DkimTokens:         sql.NullString{String: string(dkimTokensJSON), Valid: true},
		DnsRecords:         sql.NullString{String: dnsRecordsJSON, Valid: true},
		MailFromDomain:     sql.NullString{String: mailFromDomain, Valid: true},
	})
	if err != nil {
		l.Errorf("Failed to create domain identity record: %v", err)
		return nil, errorx.NewInternalError("Failed to save domain identity")
	}

	// Build response
	dnsRecords := make([]types.DNSRecord, 0, len(result.DNSRecords))
	for _, r := range result.DNSRecords {
		dnsRecords = append(dnsRecords, types.DNSRecord{
			Type:     r.Type,
			Name:     r.Name,
			Value:    r.Value,
			Priority: r.Priority,
			Purpose:  r.Purpose,
		})
	}

	return &types.DomainIdentityInfo{
		Id:                 identity.ID,
		OrgId:              identity.OrgID,
		Domain:             identity.Domain,
		VerificationStatus: identity.VerificationStatus.String,
		DKIMStatus:         identity.DkimStatus.String,
		MailFromDomain:     identity.MailFromDomain.String,
		MailFromStatus:     "not_started",
		DNSRecords:         dnsRecords,
		CreatedAt:          identity.CreatedAt.String,
	}, nil
}

// getAWSCredentials retrieves AWS credentials from platform settings
func getAWSCredentials(ctx context.Context, svcCtx *svc.ServiceContext) (region, accessKey, secretKey string, err error) {
	awsSettings, err := svcCtx.DB.GetPlatformSettingsByCategory(ctx, "aws")
	if err != nil {
		return "", "", "", errorx.NewInternalError("failed to retrieve AWS settings")
	}

	region = "us-east-1" // default

	for _, setting := range awsSettings {
		switch setting.Key {
		case "aws_access_key":
			if setting.ValueEncrypted.Valid && setting.ValueEncrypted.String != "" {
				if svcCtx.CryptoService != nil {
					decrypted, decErr := svcCtx.CryptoService.DecryptString([]byte(setting.ValueEncrypted.String))
					if decErr != nil {
						return "", "", "", errorx.NewInternalError("failed to decrypt AWS credentials")
					}
					accessKey = decrypted
				}
			}
		case "aws_secret_key":
			if setting.ValueEncrypted.Valid && setting.ValueEncrypted.String != "" {
				if svcCtx.CryptoService != nil {
					decrypted, decErr := svcCtx.CryptoService.DecryptString([]byte(setting.ValueEncrypted.String))
					if decErr != nil {
						return "", "", "", errorx.NewInternalError("failed to decrypt AWS credentials")
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
		return "", "", "", errorx.NewBadRequestError("AWS credentials not configured. Please configure AWS SES in settings.")
	}

	return region, accessKey, secretKey, nil
}
