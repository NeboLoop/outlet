package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddOutboundDomainLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddOutboundDomainLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddOutboundDomainLogic {
	return &AddOutboundDomainLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddOutboundDomainLogic) AddOutboundDomain(req *types.OutboundDomainRequest) (resp *types.OutboundDomainResponse, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Admin)
	if err != nil {
		return nil, err
	}
	return AddOutboundDomain(l.ctx, l.svcCtx, org, req)
}

// AddOutboundDomain runs for a brand already resolved: the admin route's signed-in
// person, or the API key of /sdk/v1/outbound.
func AddOutboundDomain(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.OutboundDomainRequest) (*types.OutboundDomainResponse, error) {
	in, err := convertInto[outbound.DomainInput](req)
	if err != nil {
		return nil, err
	}
	res, err := svcCtx.Outbound.AddDomain(ctx, org, in)
	if err != nil {
		return nil, apiErr(err)
	}
	info, err := convert[types.OutboundDomainInfo](outbound.DomainInfoOf(res.Domain))
	if err != nil {
		return nil, err
	}
	recs := make([]types.OutboundDNSRecord, 0, len(res.RecordsToAdd))
	for _, r := range res.RecordsToAdd {
		recs = append(recs, types.OutboundDNSRecord{Type: r.Type, Name: r.Name, Data: r.Data, Priority: r.Priority})
	}
	return &types.OutboundDomainResponse{Domain: *info, RecordsToAdd: recs, DNSError: res.DNSError}, nil
}
