package lists

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateListLogic {
	return &UpdateListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateListLogic) UpdateList(req *types.UpdateListRequest) (resp *types.ListInfo, err error) {
	listID, err := strconv.ParseInt(req.Id, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid list ID: %w", err)
	}

	if _, err := requireList(l.ctx, l.svcCtx, listID, access.Member); err != nil {
		return nil, err
	}

	params := db.UpdateEmailListParams{
		ID:   listID,
		Name: req.Name,
		Description: sql.NullString{
			String: req.Description,
			Valid:  req.Description != "",
		},
		ConfirmationSubject: req.ConfirmationSubject,
		ConfirmationBody:    req.ConfirmationBody,
	}

	if req.DoubleOptin != nil {
		params.DoubleOptin = sql.NullInt64{
			Int64: boolToInt64(*req.DoubleOptin),
			Valid: true,
		}
	}

	// Redirect URL settings
	if req.ThankYouUrl != nil {
		params.ThankYouUrl = *req.ThankYouUrl
	}
	if req.ConfirmRedirectUrl != nil {
		params.ConfirmRedirectUrl = *req.ConfirmRedirectUrl
	}
	if req.AlreadySubscribedUrl != nil {
		params.AlreadySubscribedUrl = *req.AlreadySubscribedUrl
	}
	if req.UnsubscribeRedirectUrl != nil {
		params.UnsubscribeRedirectUrl = *req.UnsubscribeRedirectUrl
	}

	// Thank you email settings
	if req.ThankYouEmailEnabled != nil {
		params.ThankYouEmailEnabled = sql.NullInt64{
			Int64: boolToInt64(*req.ThankYouEmailEnabled),
			Valid: true,
		}
	}
	if req.ThankYouEmailSubject != nil {
		params.ThankYouEmailSubject = *req.ThankYouEmailSubject
	}
	if req.ThankYouEmailBody != nil {
		params.ThankYouEmailBody = *req.ThankYouEmailBody
	}

	// Goodbye email settings
	if req.GoodbyeEmailEnabled != nil {
		params.GoodbyeEmailEnabled = sql.NullInt64{
			Int64: boolToInt64(*req.GoodbyeEmailEnabled),
			Valid: true,
		}
	}
	if req.GoodbyeEmailSubject != nil {
		params.GoodbyeEmailSubject = *req.GoodbyeEmailSubject
	}
	if req.GoodbyeEmailBody != nil {
		params.GoodbyeEmailBody = *req.GoodbyeEmailBody
	}

	// Unsubscribe behavior settings
	if req.UnsubscribeBehavior != nil {
		params.UnsubscribeBehavior = *req.UnsubscribeBehavior
	}
	if req.UnsubscribeScope != nil {
		params.UnsubscribeScope = *req.UnsubscribeScope
	}

	list, err := l.svcCtx.DB.UpdateEmailList(l.ctx, params)
	if err != nil {
		l.Errorf("Failed to update list: %v", err)
		return nil, fmt.Errorf("failed to update list: %w", err)
	}

	subscriberCount, _ := l.svcCtx.DB.CountListSubscribers(l.ctx, db.CountListSubscribersParams{
		ListID:       list.ID,
		FilterStatus: "active",
	})

	return &types.ListInfo{
		Id:                     strconv.FormatInt(list.ID, 10),
		PublicId:               list.PublicID,
		OrgId:                  list.OrgID,
		Name:                   list.Name,
		Slug:                   list.Slug,
		Description:            list.Description.String,
		DoubleOptin:            list.DoubleOptin.Int64 == 1,
		ConfirmationSubject:    list.ConfirmationEmailSubject.String,
		ConfirmationBody:       list.ConfirmationEmailBody.String,
		SubscriberCount:        int(subscriberCount),
		CreatedAt:              utils.FormatNullString(list.CreatedAt),
		UpdatedAt:              utils.FormatNullString(list.UpdatedAt),
		ThankYouUrl:            list.ThankYouUrl.String,
		ConfirmRedirectUrl:     list.ConfirmRedirectUrl.String,
		AlreadySubscribedUrl:   list.AlreadySubscribedUrl.String,
		UnsubscribeRedirectUrl: list.UnsubscribeRedirectUrl.String,
		ThankYouEmailEnabled:   list.ThankYouEmailEnabled.Int64 == 1,
		ThankYouEmailSubject:   list.ThankYouEmailSubject.String,
		ThankYouEmailBody:      list.ThankYouEmailBody.String,
		GoodbyeEmailEnabled:    list.GoodbyeEmailEnabled.Int64 == 1,
		GoodbyeEmailSubject:    list.GoodbyeEmailSubject.String,
		GoodbyeEmailBody:       list.GoodbyeEmailBody.String,
		UnsubscribeBehavior:    list.UnsubscribeBehavior.String,
		UnsubscribeScope:       list.UnsubscribeScope.String,
	}, nil
}
