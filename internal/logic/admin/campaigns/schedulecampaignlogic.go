package campaigns

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ScheduleCampaignLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewScheduleCampaignLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ScheduleCampaignLogic {
	return &ScheduleCampaignLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ScheduleCampaignLogic) ScheduleCampaign(req *types.ScheduleCampaignRequest) (resp *types.CampaignInfo, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	scheduledAt, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		return nil, errors.New("invalid scheduled_at format, use ISO8601")
	}

	campaign, err := l.svcCtx.DB.ScheduleCampaign(l.ctx, db.ScheduleCampaignParams{
		ID:          req.Id,
		OrgID:       orgID,
		ScheduledAt: sql.NullString{String: scheduledAt.Format(time.RFC3339), Valid: true},
	})
	if err != nil {
		l.Errorf("Failed to schedule campaign: %v", err)
		return nil, err
	}

	info := campaignToInfo(campaign)
	return &info, nil
}
