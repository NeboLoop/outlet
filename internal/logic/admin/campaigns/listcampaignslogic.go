package campaigns

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListCampaignsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListCampaignsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListCampaignsLogic {
	return &ListCampaignsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListCampaignsLogic) ListCampaigns(req *types.ListCampaignsRequest) (resp *types.ListCampaignsResponse, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	var campaigns []db.EmailCampaign
	var total int64

	if req.Status != "" {
		campaigns, err = l.svcCtx.DB.ListCampaignsByStatus(l.ctx, db.ListCampaignsByStatusParams{
			OrgID:      orgID,
			Status:     sql.NullString{String: req.Status, Valid: true},
			PageSize:   int64(limit),
			PageOffset: int64(offset),
		})
		if err != nil {
			l.Errorf("Failed to list campaigns by status: %v", err)
			return nil, err
		}
		total, _ = l.svcCtx.DB.CountCampaignsByStatus(l.ctx, db.CountCampaignsByStatusParams{
			OrgID:  orgID,
			Status: sql.NullString{String: req.Status, Valid: true},
		})
	} else {
		campaigns, err = l.svcCtx.DB.ListCampaigns(l.ctx, db.ListCampaignsParams{
			OrgID:      orgID,
			PageSize:   int64(limit),
			PageOffset: int64(offset),
		})
		if err != nil {
			l.Errorf("Failed to list campaigns: %v", err)
			return nil, err
		}
		total, _ = l.svcCtx.DB.CountCampaigns(l.ctx, orgID)
	}

	result := make([]types.CampaignInfo, 0)
	for _, c := range campaigns {
		result = append(result, campaignToInfo(c))
	}

	return &types.ListCampaignsResponse{
		Campaigns: result,
		Total:     int(total),
		Page:      page,
		Limit:     limit,
	}, nil
}

func campaignToInfo(c db.EmailCampaign) types.CampaignInfo {
	var listIds []string
	if c.ListIds.Valid && c.ListIds.String != "" {
		json.Unmarshal([]byte(c.ListIds.String), &listIds)
	}
	var excludeListIds []string
	if c.ExcludeListIds.Valid && c.ExcludeListIds.String != "" {
		json.Unmarshal([]byte(c.ExcludeListIds.String), &excludeListIds)
	}

	var designId *string
	if c.DesignID.Valid {
		d := strconv.FormatInt(c.DesignID.Int64, 10)
		designId = &d
	}

	return types.CampaignInfo{
		Id:                c.ID,
		OrgId:             c.OrgID,
		DesignId:          designId,
		Name:              c.Name,
		Subject:           c.Subject,
		PreviewText:       c.PreviewText.String,
		FromName:          c.FromName.String,
		FromEmail:         c.FromEmail.String,
		ReplyTo:           c.ReplyTo.String,
		HtmlBody:          c.HtmlBody,
		PlainText:         c.PlainText.String,
		ListIds:           listIds,
		ExcludeListIds:    excludeListIds,
		Status:            c.Status.String,
		ScheduledAt:       utils.FormatNullString(c.ScheduledAt),
		StartedAt:         utils.FormatNullString(c.StartedAt),
		CompletedAt:       utils.FormatNullString(c.CompletedAt),
		TrackOpens:        c.TrackOpens.Int64 == 1,
		TrackClicks:       c.TrackClicks.Int64 == 1,
		RecipientsCount:   int(c.RecipientsCount.Int64),
		SentCount:         int(c.SentCount.Int64),
		DeliveredCount:    int(c.DeliveredCount.Int64),
		OpenedCount:       int(c.OpenedCount.Int64),
		ClickedCount:      int(c.ClickedCount.Int64),
		BouncedCount:      int(c.BouncedCount.Int64),
		ComplainedCount:   int(c.ComplainedCount.Int64),
		UnsubscribedCount: int(c.UnsubscribedCount.Int64),
		CreatedAt:         utils.FormatNullString(c.CreatedAt),
		UpdatedAt:         utils.FormatNullString(c.UpdatedAt),
	}
}
