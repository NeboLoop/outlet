package campaigns

import (
	"context"
	"database/sql"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/email"
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

	if err := email.CheckSender(l.ctx, l.svcCtx.DB, orgID, req.FromEmail); err != nil {
		return nil, err
	}
	var listIds, excludeListIds interface{}
	if len(req.ListIds) > 0 {
		ids, err := access.BrandLists(l.ctx, l.svcCtx.DB, orgID, req.ListIds)
		if err != nil {
			return nil, err
		}
		listIds = db.EncodeListIDs(ids)
	}
	if len(req.ExcludeListIds) > 0 {
		ids, err := access.BrandLists(l.ctx, l.svcCtx.DB, orgID, req.ExcludeListIds)
		if err != nil {
			return nil, err
		}
		excludeListIds = db.EncodeListIDs(ids)
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
