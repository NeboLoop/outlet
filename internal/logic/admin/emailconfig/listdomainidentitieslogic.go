package emailconfig

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListDomainIdentitiesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListDomainIdentitiesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListDomainIdentitiesLogic {
	return &ListDomainIdentitiesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListDomainIdentitiesLogic) ListDomainIdentities(req *types.ListDomainIdentitiesRequest) (resp *types.ListDomainIdentitiesResponse, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, req.OrgId, access.Member); err != nil {
		return nil, err
	}

	identities, err := l.svcCtx.DB.ListDomainIdentitiesByOrg(l.ctx, req.OrgId)
	if err != nil {
		l.Errorf("Failed to list domain identities: %v", err)
		return &types.ListDomainIdentitiesResponse{Identities: []types.DomainIdentityInfo{}}, nil
	}

	result := make([]types.DomainIdentityInfo, 0, len(identities))
	for _, identity := range identities {
		// Parse DNS records from JSON
		var dnsRecords []types.DNSRecord
		if identity.DnsRecords.Valid && identity.DnsRecords.String != "" {
			records, err := email.DNSRecordsFromJSON(identity.DnsRecords.String)
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

		result = append(result, types.DomainIdentityInfo{
			Id:                 identity.ID,
			OrgId:              identity.OrgID,
			Domain:             identity.Domain,
			VerificationStatus: identity.VerificationStatus.String,
			DKIMStatus:         identity.DkimStatus.String,
			MailFromDomain:     identity.MailFromDomain.String,
			MailFromStatus:     identity.MailFromStatus.String,
			DNSRecords:         dnsRecords,
			LastCheckedAt:      identity.LastCheckedAt.String,
			CreatedAt:          identity.CreatedAt.String,
		})
	}

	return &types.ListDomainIdentitiesResponse{Identities: result}, nil
}
