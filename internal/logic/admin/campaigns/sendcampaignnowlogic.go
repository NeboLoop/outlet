package campaigns

import (
	"context"
	"database/sql"
	"errors"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/middleware"
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
	orgID, ok := l.ctx.Value(middleware.OrgIDKey).(string)
	if !ok {
		return nil, errors.New("org_id not found in context")
	}

	// The scheduler picks it up within its check interval and creates the sends
	campaign, err := l.svcCtx.DB.SendCampaignNow(l.ctx, db.SendCampaignNowParams{
		ID:    req.Id,
		OrgID: orgID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errorx.NewBadRequestError("Only draft or scheduled campaigns can be sent")
	}
	if err != nil {
		l.Errorf("Failed to start campaign: %v", err)
		return nil, err
	}

	info := campaignToInfo(campaign)
	return &info, nil
}
