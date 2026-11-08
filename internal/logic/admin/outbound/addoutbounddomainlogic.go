package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
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
	in, err := convertInto[outbound.DomainInput](req)
	if err != nil {
		return nil, err
	}
	d, pending, _, err := l.svcCtx.Outbound.AddDomain(l.ctx, org, in)
	if err != nil {
		return nil, apiErr(err)
	}
	info, err := convert[types.OutboundDomainInfo](outbound.DomainInfoOf(d))
	if err != nil {
		return nil, err
	}
	recs, err := convert[[]types.OutboundDNSRecord](pending)
	if err != nil {
		return nil, err
	}
	return &types.OutboundDomainResponse{Domain: *info, RecordsToAdd: *recs}, nil
}
