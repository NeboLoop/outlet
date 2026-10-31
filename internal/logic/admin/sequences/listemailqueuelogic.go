package sequences

import (
	"context"
	"database/sql"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListEmailQueueLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListEmailQueueLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListEmailQueueLogic {
	return &ListEmailQueueLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListEmailQueueLogic) ListEmailQueue(req *types.EmailQueueListRequest) (resp *types.EmailQueueListResponse, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	params := db.ListEmailQueueByOrgParams{
		OrgID: sql.NullString{String: orgID, Valid: true},
	}

	if req.Status != "" {
		params.FilterStatus = req.Status
	}

	if req.ContactId != "" {
		params.FilterContactID = req.ContactId
	}

	queueItems, err := l.svcCtx.DB.ListEmailQueueByOrg(l.ctx, params)
	if err != nil {
		l.Errorf("Failed to list email queue: %v", err)
		return nil, err
	}

	items := make([]types.EmailQueueItem, 0, len(queueItems))
	for _, item := range queueItems {
		queueItem := types.EmailQueueItem{
			Id:           item.ID,
			ContactId:    item.ContactID.String,
			TemplateId:   item.TemplateID.String,
			Subject:      item.Subject,
			ScheduledFor: item.ScheduledFor,
			Status:       item.Status.String,
			ContactEmail: item.ContactEmail,
			ContactName:  item.ContactName,
		}

		if item.SentAt.Valid {
			queueItem.SentAt = item.SentAt.String
		}

		if item.ErrorMessage.Valid {
			queueItem.ErrorMessage = item.ErrorMessage.String
		}

		items = append(items, queueItem)
	}

	return &types.EmailQueueListResponse{
		Items: items,
		Total: len(items),
	}, nil
}
