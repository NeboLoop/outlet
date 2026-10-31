package campaigns

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateCampaignLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateCampaignLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCampaignLogic {
	return &UpdateCampaignLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateCampaignLogic) UpdateCampaign(req *types.UpdateCampaignRequest) (resp *types.CampaignInfo, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	var listIds, excludeListIds interface{}
	if len(req.ListIds) > 0 {
		data, _ := json.Marshal(req.ListIds)
		listIds = string(data)
	}
	if len(req.ExcludeListIds) > 0 {
		data, _ := json.Marshal(req.ExcludeListIds)
		excludeListIds = string(data)
	}

	var trackOpens, trackClicks sql.NullInt64
	if req.TrackOpens {
		trackOpens = sql.NullInt64{Int64: 1, Valid: true}
	} else {
		trackOpens = sql.NullInt64{Int64: 0, Valid: true}
	}
	if req.TrackClicks {
		trackClicks = sql.NullInt64{Int64: 1, Valid: true}
	} else {
		trackClicks = sql.NullInt64{Int64: 0, Valid: true}
	}

	campaign, err := l.svcCtx.DB.UpdateCampaign(l.ctx, db.UpdateCampaignParams{
		ID:             req.Id,
		OrgID:          orgID,
		Name:           req.Name,
		Subject:        req.Subject,
		PreviewText:    req.PreviewText,
		FromName:       req.FromName,
		FromEmail:      req.FromEmail,
		ReplyTo:        req.ReplyTo,
		HtmlBody:       req.HtmlBody,
		PlainText:      req.PlainText,
		ListIds:        listIds,
		ExcludeListIds: excludeListIds,
		TrackOpens:     trackOpens,
		TrackClicks:    trackClicks,
	})
	if err != nil {
		l.Errorf("Failed to update campaign: %v", err)
		return nil, err
	}

	info := campaignToInfo(campaign)
	return &info, nil
}
