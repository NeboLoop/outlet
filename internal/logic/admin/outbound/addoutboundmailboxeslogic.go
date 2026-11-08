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

type AddOutboundMailboxesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddOutboundMailboxesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddOutboundMailboxesLogic {
	return &AddOutboundMailboxesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddOutboundMailboxesLogic) AddOutboundMailboxes(req *types.OutboundAddMailboxesRequest) (resp *types.OutboundAddMailboxesResponse, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Admin)
	if err != nil {
		return nil, err
	}
	return AddOutboundMailboxes(l.ctx, l.svcCtx, org, req)
}

// AddOutboundMailboxes runs for a brand already resolved: the admin route's signed-in
// person, or the API key of /sdk/v1/outbound.
func AddOutboundMailboxes(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.OutboundAddMailboxesRequest) (*types.OutboundAddMailboxesResponse, error) {
	in, err := convertInto[[]outbound.MailboxInput](req.Mailboxes)
	if err != nil {
		return nil, err
	}
	res, err := svcCtx.Outbound.AddMailboxes(ctx, org, in)
	if err != nil {
		return nil, apiErr(err)
	}
	out, err := convert[[]types.OutboundMailboxResult](res)
	if err != nil {
		return nil, err
	}
	return &types.OutboundAddMailboxesResponse{Mailboxes: *out}, nil
}
