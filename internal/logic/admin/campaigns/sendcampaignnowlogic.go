package campaigns

import (
	"context"
	"database/sql"
	"errors"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendCampaignNowLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSendCampaignNowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendCampaignNowLogic {
	return &SendCampaignNowLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SendCampaignNowLogic) SendCampaignNow(req *types.SendCampaignNowRequest) (resp *types.CampaignInfo, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	// Schedule it for now: the scheduler is the one sending path. The
	// campaign comes back "scheduled" until the scheduler starts it.
	campaign, err := l.svcCtx.DB.ScheduleCampaign(l.ctx, db.ScheduleCampaignParams{
		ID:    req.Id,
		OrgID: orgID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errorx.NewBadRequestError("Only a draft campaign can be sent.")
	}
	if err != nil {
		l.Errorf("Failed to queue campaign: %v", err)
		return nil, err
	}

	info := campaignToInfo(campaign)
	return &info, nil
}
