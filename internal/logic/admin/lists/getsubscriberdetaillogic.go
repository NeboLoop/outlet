package lists

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSubscriberDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetSubscriberDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSubscriberDetailLogic {
	return &GetSubscriberDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSubscriberDetailLogic) GetSubscriberDetail(req *types.GetSubscriberDetailRequest) (resp *types.SubscriberDetailResponse, err error) {
	listID, err := strconv.ParseInt(req.Id, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid list ID: %w", err)
	}
	if _, err := requireList(l.ctx, l.svcCtx, listID, access.Member); err != nil {
		return nil, err
	}

	// Get detailed subscriber info
	detail, err := l.svcCtx.DB.GetListSubscriberDetail(l.ctx, req.SubscriberId)
	if err != nil || detail.ListID != listID {
		return nil, errorx.NewNotFoundError("Subscriber not found.")
	}

	// Get custom field values for this subscriber
	customFields := make(map[string]string)
	cfValues, _ := l.svcCtx.DB.GetSubscriberCustomFieldsForMerge(l.ctx, detail.ID)
	for _, cf := range cfValues {
		if cf.Value.Valid {
			customFields[cf.FieldKey] = cf.Value.String
		}
	}

	// Get campaign activity
	campaignActivity := make([]types.CampaignActivityItem, 0)
	activity, _ := l.svcCtx.DB.GetSubscriberCampaignActivity(l.ctx, detail.ContactID)
	for _, a := range activity {
		item := types.CampaignActivityItem{
			CampaignId: a.CampaignID,
		}
		if a.Status.Valid {
			item.Status = a.Status.String
		}
		if a.OpenCount.Valid {
			item.OpenCount = a.OpenCount.Int64
		}
		if a.ClickCount.Valid {
			item.ClickCount = a.ClickCount.Int64
		}
		if a.CampaignName.Valid {
			item.CampaignName = a.CampaignName.String
		}
		if a.CampaignSubject.Valid {
			item.CampaignSubject = a.CampaignSubject.String
		}
		if a.SentAt.Valid {
			item.SentAt = a.SentAt.String
		}
		if a.OpenedAt.Valid {
			item.OpenedAt = a.OpenedAt.String
		}
		if a.ClickedAt.Valid {
			item.ClickedAt = a.ClickedAt.String
		}
		campaignActivity = append(campaignActivity, item)
	}

	// Get sequence enrollments
	sequenceEnrollments := make([]types.SequenceEnrollmentItem, 0)
	enrollments, _ := l.svcCtx.DB.GetSubscriberSequenceEnrollments(l.ctx, sql.NullString{String: detail.ContactID, Valid: true})
	for _, e := range enrollments {
		item := types.SequenceEnrollmentItem{}
		if e.SequenceID.Valid {
			item.SequenceId = e.SequenceID.String
		}
		if e.CurrentPosition.Valid {
			item.CurrentPosition = e.CurrentPosition.Int64
		}
		if e.IsActive.Valid {
			item.IsActive = e.IsActive.Int64 == 1
		}
		if e.SequenceName.Valid {
			item.SequenceName = e.SequenceName.String
		}
		if e.StartedAt.Valid {
			item.StartedAt = e.StartedAt.String
		}
		if e.CompletedAt.Valid {
			item.CompletedAt = e.CompletedAt.String
		}
		if e.PausedAt.Valid {
			item.PausedAt = e.PausedAt.String
		}
		if e.UnsubscribedAt.Valid {
			item.UnsubscribedAt = e.UnsubscribedAt.String
		}
		sequenceEnrollments = append(sequenceEnrollments, item)
	}

	resp = &types.SubscriberDetailResponse{
		Id:                  detail.ID,
		ListId:              fmt.Sprintf("%d", detail.ListID),
		ContactId:           detail.ContactID,
		Email:               detail.Email,
		Name:                detail.Name,
		EmailVerified:       detail.EmailVerified == 1,
		CustomFields:        customFields,
		CampaignActivity:    campaignActivity,
		SequenceEnrollments: sequenceEnrollments,
	}

	// Handle nullable status
	if detail.Status.Valid {
		resp.Status = detail.Status.String
	}

	// Handle nullable source
	if detail.Source.Valid {
		resp.Source = detail.Source.String
	}

	// Handle GDPR fields
	if detail.GdprConsent.Valid {
		resp.GdprConsent = detail.GdprConsent.Int64 == 1
	}

	// Handle nullable timestamps
	if detail.SubscribedAt.Valid {
		resp.SubscribedAt = detail.SubscribedAt.String
	}
	if detail.VerifiedAt.Valid {
		resp.VerifiedAt = detail.VerifiedAt.String
	}
	if detail.UnsubscribedAt.Valid {
		resp.UnsubscribedAt = detail.UnsubscribedAt.String
	}
	if detail.GdprConsentAt.Valid {
		resp.GdprConsentAt = detail.GdprConsentAt.String
	}
	if detail.BlockedAt.Valid {
		resp.BlockedAt = detail.BlockedAt.String
	}

	// Handle email stats (interface{} types from COALESCE)
	if v, ok := detail.EmailsSent.(int64); ok {
		resp.EmailsSent = v
	}
	if v, ok := detail.EmailsOpened.(int64); ok {
		resp.EmailsOpened = v
	}
	if v, ok := detail.EmailsClicked.(int64); ok {
		resp.EmailsClicked = v
	}

	// Handle last activity timestamps (interface{} types)
	if v, ok := detail.LastEmailAt.(string); ok && v != "" {
		resp.LastEmailAt = v
	}
	if v, ok := detail.LastOpenAt.(string); ok && v != "" {
		resp.LastOpenAt = v
	}
	if v, ok := detail.LastClickAt.(string); ok && v != "" {
		resp.LastClickAt = v
	}

	return resp, nil
}
