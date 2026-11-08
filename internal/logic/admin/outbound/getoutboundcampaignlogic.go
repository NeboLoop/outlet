package outbound

import (
	"context"
	"database/sql"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOutboundCampaignLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetOutboundCampaignLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOutboundCampaignLogic {
	return &GetOutboundCampaignLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetOutboundCampaignLogic) GetOutboundCampaign(req *types.OutboundIDRequest) (resp *types.OutboundCampaignResults, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	c, err := l.svcCtx.DB.GetOutreachSequence(l.ctx, db.GetOutreachSequenceParams{ID: req.Id, OrgID: sql.NullString{String: org.ID, Valid: true}})
	if err != nil {
		return nil, errorx.NewNotFoundError("No such campaign.")
	}
	r, err := l.svcCtx.Outbound.CampaignResults(l.ctx, c, true)
	if err != nil {
		return nil, err
	}
	return convert[types.OutboundCampaignResults](r)
}
