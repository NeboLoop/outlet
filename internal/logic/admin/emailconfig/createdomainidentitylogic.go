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
		if existing.Purpose == "outreach" {
			return nil, errorx.NewBadRequestError(domain + " is one of this brand's outreach domains; outreach domains never send through SES.")
		}
		return nil, errorx.NewBadRequestError("Domain identity already exists for this domain")
	}
	if err := email.ClaimDomain(l.ctx, l.svcCtx.DB, req.OrgId, domain); err != nil {
		return nil, err
	}

	// The domain joins one of the brand's sending pathways, and is verified
	// with that pathway's AWS account: an ses pathway's own, or the
	// platform's.
	pathway, err := l.svcCtx.EmailService.DomainPathway(l.ctx, req.OrgId, "sending", req.PathwayId)
	if err != nil {
		return nil, err
	}
	region, accessKey, secretKey, awsErr := email.AWSCredentials(l.ctx, l.svcCtx.DB, l.svcCtx.CryptoService, req.OrgId, pathway.ID)
	if awsErr != nil {
		// No SES: mail goes through an SMTP server (the platform's or the
		// pathway's own), which signs it with its own SPF and DKIM. The
		// domain is recorded so the brand may send from it; SES never sees it.
		return l.recordSMTPDomain(req.OrgId, pathway.ID, domain)
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
		PathwayID:          sql.NullString{String: pathway.ID, Valid: true},
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

// recordSMTPDomain adds a sending domain on a server that sends through SMTP
// rather than SES. There is nothing for SES to verify, so it is ready at once.
func (l *CreateDomainIdentityLogic) recordSMTPDomain(orgID, pathwayID, domain string) (*types.DomainIdentityInfo, error) {
	identity, err := l.svcCtx.DB.CreateDomainIdentity(l.ctx, db.CreateDomainIdentityParams{
		ID: uuid.New().String(), OrgID: orgID, Domain: domain, PathwayID: sql.NullString{String: pathwayID, Valid: true},
		VerificationStatus: sql.NullString{String: "success", Valid: true},
		DkimStatus:         sql.NullString{String: "success", Valid: true},
		DkimTokens:         sql.NullString{String: "[]", Valid: true},
		DnsRecords:         sql.NullString{String: "[]", Valid: true},
	})
	if err != nil {
		l.Errorf("Failed to create domain identity record: %v", err)
		return nil, errorx.NewInternalError("Failed to save domain identity")
	}
	return &types.DomainIdentityInfo{
		Id: identity.ID, OrgId: identity.OrgID, Domain: identity.Domain,
		VerificationStatus: "success", DKIMStatus: "success", MailFromStatus: "not_started",
		DNSRecords: []types.DNSRecord{}, CreatedAt: identity.CreatedAt.String,
	}, nil
}
