package lists

import (
	"context"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListListsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListListsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListListsLogic {
	return &ListListsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListListsLogic) ListLists() (resp *types.ListListsResponse, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	rows, err := l.svcCtx.DB.ListEmailLists(l.ctx, orgID)
	if err != nil {
		l.Errorf("Failed to list lists: %v", err)
		return nil, err
	}

	lists := make([]types.ListInfo, 0, len(rows))
	for _, row := range rows {
		subscriberCount, err := l.svcCtx.DB.CountListSubscribers(l.ctx, db.CountListSubscribersParams{
			ListID:       row.ID,
			FilterStatus: "active",
		})
		if err != nil {
			l.Errorf("Failed to count subscribers for list %d: %v", row.ID, err)
			subscriberCount = 0
		}

		lists = append(lists, types.ListInfo{
			Id:              strconv.FormatInt(row.ID, 10),
			PublicId:        row.PublicID,
			OrgId:           row.OrgID,
			Name:            row.Name,
			Slug:            row.Slug,
			Description:     row.Description.String,
			DoubleOptin:     row.DoubleOptin.Int64 == 1,
			SubscriberCount: int(subscriberCount),
			CreatedAt:       utils.FormatNullString(row.CreatedAt),
			UpdatedAt:       utils.FormatNullString(row.UpdatedAt),
		})
	}

	return &types.ListListsResponse{
		Lists: lists,
	}, nil
}
