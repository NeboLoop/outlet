package emailconfig

import (
	"context"
	"database/sql"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RefreshDomainIdentityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRefreshDomainIdentityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefreshDomainIdentityLogic {
	return &RefreshDomainIdentityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RefreshDomainIdentityLogic) RefreshDomainIdentity(req *types.RefreshDomainIdentityRequest) (resp *types.DomainIdentityInfo, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, req.OrgId, access.Admin); err != nil {
		return nil, err
	}

	// Get the domain identity
	identity, err := l.svcCtx.DB.GetDomainIdentity(l.ctx, req.Id)
	if err != nil {
		return nil, errorx.NewNotFoundError("Domain identity not found")
	}

	// Verify the identity belongs to the requested org
	if identity.OrgID != req.OrgId {
		return nil, errorx.NewNotFoundError("Domain identity not found")
	}

	// Get AWS credentials from platform settings
	region, accessKey, secretKey, err := getAWSCredentials(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}

	// Get email config for org-specific AWS credentials
	emailConfig, err := email.GetOrgEmailConfig(l.ctx, l.svcCtx.DB, req.OrgId)
	if err == nil && emailConfig.HasOwnAWSCredentials() {
		region = emailConfig.AWSRegion
		accessKey = emailConfig.AWSAccessKey
		secretKey = emailConfig.AWSSecretKey
	}

	// Get the current status from AWS SES
	status, err := email.GetDomainIdentityStatus(l.ctx, region, accessKey, secretKey, identity.Domain)
	if err != nil {
		l.Errorf("Failed to get domain identity status: %v", err)
		return nil, errorx.NewInternalError("Failed to check domain status with AWS SES: " + err.Error())
	}

	// If status is "not_started", the domain was never registered with SES - register it now
	if status.VerificationStatus == "not_started" {
		l.Infof("Domain %s not registered with SES, initiating verification...", identity.Domain)
		result, verifyErr := email.VerifyDomainIdentity(l.ctx, region, accessKey, secretKey, identity.Domain)
		if verifyErr != nil {
			l.Errorf("Failed to verify domain identity: %v", verifyErr)
			return nil, errorx.NewInternalError("Failed to initiate domain verification: " + verifyErr.Error())
		}

		// Update with new DNS records and pending status
		dnsRecordsJSON, _ := email.DNSRecordsToJSON(result.DNSRecords)
		updated, err := l.svcCtx.DB.UpdateDomainIdentityFull(l.ctx, db.UpdateDomainIdentityFullParams{
			ID:                 identity.ID,
			VerificationStatus: sql.NullString{String: result.VerificationStatus, Valid: true},
			DkimStatus:         sql.NullString{String: result.DKIMStatus, Valid: true},
			VerificationToken:  sql.NullString{String: result.VerificationToken, Valid: result.VerificationToken != ""},
			DnsRecords:         sql.NullString{String: dnsRecordsJSON, Valid: true},
		})
		if err != nil {
			l.Errorf("Failed to update domain identity: %v", err)
			return nil, errorx.NewInternalError("Failed to save domain verification data")
		}

		// Parse DNS records for response
		var dnsRecords []types.DNSRecord
		for _, r := range result.DNSRecords {
			dnsRecords = append(dnsRecords, types.DNSRecord{
				Type:     r.Type,
				Name:     r.Name,
				Value:    r.Value,
				Priority: r.Priority,
				Purpose:  r.Purpose,
			})
		}

		// Set up bounce notifications
		if l.svcCtx.Config.App.BaseURL != "" {
			webhookURL := l.svcCtx.Config.App.BaseURL + "/webhooks/ses/" + req.OrgId
			if notifErr := email.SetupBounceNotifications(l.ctx, region, accessKey, secretKey, identity.Domain, webhookURL); notifErr != nil {
				l.Errorf("Failed to set up bounce notifications: %v", notifErr)
				// Don't fail the request - notifications can be set up later
			}
		}

		return &types.DomainIdentityInfo{
			Id:                 updated.ID,
			OrgId:              updated.OrgID,
			Domain:             updated.Domain,
			VerificationStatus: updated.VerificationStatus.String,
			DKIMStatus:         updated.DkimStatus.String,
			MailFromDomain:     updated.MailFromDomain.String,
			MailFromStatus:     updated.MailFromStatus.String,
			DNSRecords:         dnsRecords,
			LastCheckedAt:      updated.LastCheckedAt.String,
			CreatedAt:          updated.CreatedAt.String,
		}, nil
	}

	// Update the status in the database
	updated, err := l.svcCtx.DB.UpdateDomainIdentityStatus(l.ctx, db.UpdateDomainIdentityStatusParams{
		ID:                 identity.ID,
		VerificationStatus: sql.NullString{String: status.VerificationStatus, Valid: true},
		DkimStatus:         sql.NullString{String: status.DKIMStatus, Valid: true},
	})
	if err != nil {
		l.Errorf("Failed to update domain identity status: %v", err)
		return nil, errorx.NewInternalError("Failed to save domain status")
	}

	// Parse DNS records from JSON
	var dnsRecords []types.DNSRecord
	if updated.DnsRecords.Valid && updated.DnsRecords.String != "" {
		records, err := email.DNSRecordsFromJSON(updated.DnsRecords.String)
		if err == nil {
			for _, r := range records {
				dnsRecords = append(dnsRecords, types.DNSRecord{
					Type:     r.Type,
					Name:     r.Name,
					Value:    r.Value,
					Priority: r.Priority,
					Purpose:  r.Purpose,
				})
			}
		}
	}

	return &types.DomainIdentityInfo{
		Id:                 updated.ID,
		OrgId:              updated.OrgID,
		Domain:             updated.Domain,
		VerificationStatus: updated.VerificationStatus.String,
		DKIMStatus:         updated.DkimStatus.String,
		MailFromDomain:     updated.MailFromDomain.String,
		MailFromStatus:     updated.MailFromStatus.String,
		DNSRecords:         dnsRecords,
		LastCheckedAt:      updated.LastCheckedAt.String,
		CreatedAt:          updated.CreatedAt.String,
	}, nil
}
