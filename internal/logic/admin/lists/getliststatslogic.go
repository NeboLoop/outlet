package lists

import (
	"context"
	"fmt"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetListStatsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetListStatsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetListStatsLogic {
	return &GetListStatsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetListStatsLogic) GetListStats(req *types.GetListRequest) (resp *types.ListStatsResponse, err error) {
	listID, err := strconv.ParseInt(req.Id, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid list ID: %w", err)
	}

	if _, err := requireList(l.ctx, l.svcCtx, listID, access.Member); err != nil {
		return nil, err
	}

	activeCount, err := l.svcCtx.DB.CountListSubscribers(l.ctx, db.CountListSubscribersParams{
		ListID:       listID,
		FilterStatus: "active",
	})
	if err != nil {
		l.Errorf("Failed to count active subscribers: %v", err)
		activeCount = 0
	}

	bounceCount := int64(0)
	bounces, err := l.svcCtx.DB.ListRecentBounces(l.ctx, db.ListRecentBouncesParams{
		PageSize:   1000,
		PageOffset: 0,
	})
	if err == nil {
		bounceCount = int64(len(bounces))
	}

	complaintCount := int64(0)
	complaints, err := l.svcCtx.DB.ListRecentComplaints(l.ctx, db.ListRecentComplaintsParams{
		PageSize:   1000,
		PageOffset: 0,
	})
	if err == nil {
		complaintCount = int64(len(complaints))
	}

	return &types.ListStatsResponse{
		TotalSubscribers:  int(activeCount),
		ActiveSubscribers: int(activeCount),
		Unsubscribed:      0,
		Bounced:           int(bounceCount),
		Complained:        int(complaintCount),
	}, nil
}
