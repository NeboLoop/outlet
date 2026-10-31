package campaigns

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateCampaignLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateCampaignLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateCampaignLogic {
	return &CreateCampaignLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateCampaignLogic) CreateCampaign(req *types.CreateCampaignRequest) (resp *types.CampaignInfo, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	if err := email.CheckSender(l.ctx, l.svcCtx.DB, orgID, req.FromEmail); err != nil {
		return nil, err
	}
	listIDs, err := access.BrandLists(l.ctx, l.svcCtx.DB, orgID, req.ListIds)
	if err != nil {
		return nil, err
	}
	excludeListIDs, err := access.BrandLists(l.ctx, l.svcCtx.DB, orgID, req.ExcludeListIds)
	if err != nil {
		return nil, err
	}

	var designID sql.NullInt64
	if req.DesignId != nil {
		if parsed, err := strconv.ParseInt(*req.DesignId, 10, 64); err == nil {
			designID = sql.NullInt64{Int64: parsed, Valid: true}
		}
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

	campaign, err := l.svcCtx.DB.CreateCampaign(l.ctx, db.CreateCampaignParams{
		ID:             uuid.New().String(),
		OrgID:          orgID,
		DesignID:       designID,
		Name:           req.Name,
		Subject:        req.Subject,
		PreviewText:    sql.NullString{String: req.PreviewText, Valid: req.PreviewText != ""},
		FromName:       sql.NullString{String: req.FromName, Valid: req.FromName != ""},
		FromEmail:      sql.NullString{String: req.FromEmail, Valid: req.FromEmail != ""},
		ReplyTo:        sql.NullString{String: req.ReplyTo, Valid: req.ReplyTo != ""},
		HtmlBody:       req.HtmlBody,
		PlainText:      sql.NullString{String: req.PlainText, Valid: req.PlainText != ""},
		ListIds:        sql.NullString{String: db.EncodeListIDs(listIDs), Valid: true},
		ExcludeListIds: sql.NullString{String: db.EncodeListIDs(excludeListIDs), Valid: len(excludeListIDs) > 0},
		Status:         sql.NullString{String: "draft", Valid: true},
		TrackOpens:     trackOpens,
		TrackClicks:    trackClicks,
	})
	if err != nil {
		l.Errorf("Failed to create campaign: %v", err)
		return nil, err
	}

	info := campaignToInfo(campaign)
	return &info, nil
}
