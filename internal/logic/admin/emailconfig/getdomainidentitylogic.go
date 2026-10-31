package emailconfig

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetDomainIdentityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetDomainIdentityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDomainIdentityLogic {
	return &GetDomainIdentityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetDomainIdentityLogic) GetDomainIdentity(req *types.GetDomainIdentityRequest) (resp *types.DomainIdentityInfo, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, req.OrgId, access.Member); err != nil {
		return nil, err
	}

	identity, err := l.svcCtx.DB.GetDomainIdentity(l.ctx, req.Id)
	if err != nil {
		return nil, errorx.NewNotFoundError("Domain identity not found")
	}

	// Verify the identity belongs to the requested org
	if identity.OrgID != req.OrgId {
		return nil, errorx.NewNotFoundError("Domain identity not found")
	}

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

	return &types.DomainIdentityInfo{
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
	}, nil
}
