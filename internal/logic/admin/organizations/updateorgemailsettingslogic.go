package organizations

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateOrgEmailSettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateOrgEmailSettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateOrgEmailSettingsLogic {
	return &UpdateOrgEmailSettingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateOrgEmailSettingsLogic) UpdateOrgEmailSettings(req *types.UpdateOrgEmailSettingsRequest) (resp *types.OrgInfo, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, req.Id, access.Admin); err != nil {
		return nil, err
	}

	// Update email settings (from_name, from_email, reply_to)
	org, err := l.svcCtx.DB.UpdateOrgEmailSettings(l.ctx, db.UpdateOrgEmailSettingsParams{
		ID:        req.Id,
		FromName:  req.FromName,
		FromEmail: req.FromEmail,
		ReplyTo:   req.ReplyTo,
	})
	if err != nil {
		return nil, err
	}

	// Invalidate API key middleware cache so SDK gets updated org settings
	l.svcCtx.APIKeyMiddleware.InvalidateCache(org.ApiKey)

	// Auto-create domain identity if from_email is provided and no identity exists for the domain
	if req.FromEmail != "" {
		go l.ensureDomainIdentity(req.Id, req.FromEmail)
	}

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

// ensureDomainIdentity creates a domain identity in AWS SES if one doesn't exist for the domain
func (l *UpdateOrgEmailSettingsLogic) ensureDomainIdentity(orgID, fromEmail string) {
	// Extract domain from email
	parts := strings.Split(fromEmail, "@")
	if len(parts) != 2 {
		l.Errorf("Invalid email address format: %s", fromEmail)
		return
	}
	domain := parts[1]

	// Check if domain identity already exists and is properly verified
	existing, err := l.svcCtx.DB.GetDomainIdentityByDomain(l.ctx, db.GetDomainIdentityByDomainParams{
		OrgID:  orgID,
		Domain: domain,
	})
	if err == nil && existing.ID != "" {
		// If domain exists but verification never started, we'll re-trigger it below
		// Otherwise, the domain is already being verified or is verified
		if existing.VerificationStatus.String != "not_started" && existing.VerificationStatus.String != "" {
			l.Infof("Domain identity already exists for %s (status: %s)", domain, existing.VerificationStatus.String)
			return
		}
		l.Infof("Domain identity exists but verification not started for %s, will re-trigger", domain)
	}

	// Get AWS credentials
	region, accessKey, secretKey, err := l.getAWSCredentials()
	if err != nil {
		l.Errorf("Failed to get AWS credentials for auto domain verification: %v", err)
		return
	}

	// Check for org-specific AWS credentials
	emailConfig, err := email.GetOrgEmailConfig(l.ctx, l.svcCtx.DB, orgID)
	if err == nil && emailConfig.HasOwnAWSCredentials() {
		region = emailConfig.AWSRegion
		accessKey = emailConfig.AWSAccessKey
		secretKey = emailConfig.AWSSecretKey
	}

	// Create domain identity in AWS SES
	result, err := email.VerifyDomainIdentity(l.ctx, region, accessKey, secretKey, domain)
	if err != nil {
		l.Errorf("Failed to auto-verify domain identity for %s: %v", domain, err)
		return
	}

	// Serialize DNS records and DKIM tokens
	dnsRecordsJSON, err := email.DNSRecordsToJSON(result.DNSRecords)
	if err != nil {
		l.Errorf("Failed to serialize DNS records: %v", err)
		return
	}

	dkimTokensJSON, err := json.Marshal(result.DKIMTokens)
	if err != nil {
		l.Errorf("Failed to serialize DKIM tokens: %v", err)
		return
	}

	// Create or update domain identity record
	if existing.ID != "" {
		// Update existing domain identity that was in "not_started" state
		_, err = l.svcCtx.DB.UpdateDomainIdentityFull(l.ctx, db.UpdateDomainIdentityFullParams{
			ID:                 existing.ID,
			VerificationStatus: sql.NullString{String: result.VerificationStatus, Valid: true},
			DkimStatus:         sql.NullString{String: result.DKIMStatus, Valid: true},
			VerificationToken:  sql.NullString{String: result.VerificationToken, Valid: result.VerificationToken != ""},
			DnsRecords:         sql.NullString{String: dnsRecordsJSON, Valid: true},
		})
		if err != nil {
			l.Errorf("Failed to update domain identity for %s: %v", domain, err)
			return
		}
		l.Infof("Re-triggered verification for domain identity %s", domain)
	} else {
		// Create new domain identity
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
			l.Errorf("Failed to save domain identity for %s: %v", domain, err)
			return
		}
		l.Infof("Auto-created domain identity for %s", domain)
	}

	// Set up bounce/complaint/delivery notifications via SNS -> webhook
	if l.svcCtx.Config.App.BaseURL != "" {
		webhookURL := l.svcCtx.Config.App.BaseURL + "/webhooks/ses/" + orgID
		if err := email.SetupBounceNotifications(l.ctx, region, accessKey, secretKey, domain, webhookURL); err != nil {
			l.Errorf("Failed to set up bounce notifications for %s: %v", domain, err)
			// Don't fail - notifications can be set up later
		} else {
			l.Infof("Set up bounce/complaint/delivery notifications for %s", domain)
		}
	}

	// Notify connected clients that a domain identity was created
	if l.svcCtx.WebSocketHub != nil {
		l.svcCtx.WebSocketHub.BroadcastDomainIdentityCreated(orgID, domain)
	}
}

// getAWSCredentials retrieves AWS credentials from platform settings
func (l *UpdateOrgEmailSettingsLogic) getAWSCredentials() (region, accessKey, secretKey string, err error) {
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
		return "", "", "", err
	}

	return region, accessKey, secretKey, nil
}
