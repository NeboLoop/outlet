package lists

import (
	"context"
	"fmt"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListListSubscribersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListListSubscribersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListListSubscribersLogic {
	return &ListListSubscribersLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListListSubscribersLogic) ListListSubscribers(req *types.ListSubscribersRequest) (resp *types.ListSubscribersResponse, err error) {
	listID, err := strconv.ParseInt(req.Id, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid list ID: %w", err)
	}

	if _, err := requireList(l.ctx, l.svcCtx, listID, access.Member); err != nil {
		return nil, err
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	limit := req.Limit
	if limit < 1 {
		limit = 50
	}
	offset := (page - 1) * limit

	var statusFilter interface{}
	if req.Status != "" {
		statusFilter = req.Status
	}

	subscribers, err := l.svcCtx.DB.ListListSubscribers(l.ctx, db.ListListSubscribersParams{
		ListID:       listID,
		PageSize:     int64(limit),
		PageOffset:   int64(offset),
		FilterStatus: statusFilter,
	})
	if err != nil {
		l.Errorf("Failed to list subscribers: %v", err)
		return nil, fmt.Errorf("failed to list subscribers: %w", err)
	}

	totalCount, _ := l.svcCtx.DB.CountListSubscribers(l.ctx, db.CountListSubscribersParams{
		ListID:       listID,
		FilterStatus: statusFilter,
	})

	items := make([]types.ListSubscriberInfo, 0, len(subscribers))
	for _, sub := range subscribers {
		items = append(items, types.ListSubscriberInfo{
			Id:             sub.ID,
			ContactId:      sub.ContactID,
			Email:          sub.Email,
			Name:           sub.Name,
			Status:         sub.Status.String,
			SubscribedAt:   utils.FormatNullString(sub.SubscribedAt),
			UnsubscribedAt: utils.FormatNullString(sub.UnsubscribedAt),
		})
	}

	return &types.ListSubscribersResponse{
		Subscribers: items,
		Total:       int(totalCount),
		Page:        page,
		Limit:       limit,
	}, nil
}
