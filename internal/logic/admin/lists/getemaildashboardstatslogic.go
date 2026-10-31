package lists

import (
	"context"
	"database/sql"
	"time"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetEmailDashboardStatsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetEmailDashboardStatsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetEmailDashboardStatsLogic {
	return &GetEmailDashboardStatsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetEmailDashboardStatsLogic) GetEmailDashboardStats(req *types.EmailDashboardStatsRequest) (resp *types.EmailDashboardStatsResponse, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return nil, err
	}

	var fromTime, toTime time.Time

	if req.From != "" {
		fromTime, err = time.Parse("2006-01-02", req.From)
		if err != nil {
			fromTime = time.Now().AddDate(0, 0, -30)
		}
	} else {
		fromTime = time.Now().AddDate(0, 0, -30)
	}

	if req.To != "" {
		toTime, err = time.Parse("2006-01-02", req.To)
		if err != nil {
			toTime = time.Now()
		}
		toTime = toTime.Add(24*time.Hour - time.Second)
	} else {
		toTime = time.Now()
	}

	fromStr := fromTime.Format(time.RFC3339)
	toStr := toTime.Format(time.RFC3339)

	var totalSent, totalOpened, totalClicked int64

	stats, err := l.svcCtx.DB.GetEmailStatsInDateRange(l.ctx, db.GetEmailStatsInDateRangeParams{
		StartDate: sql.NullString{String: fromStr, Valid: true},
		EndDate:   sql.NullString{String: toStr, Valid: true},
	})
	if err != nil {
		l.Errorf("Failed to get email stats: %v", err)
	} else {
		totalSent = int64(stats.SentCount.Float64)
		totalOpened = int64(stats.OpenedCount.Float64)
		totalClicked = int64(stats.ClickedCount.Float64)
	}

	bounceCount, err := l.svcCtx.DB.CountBouncesInDateRange(l.ctx, db.CountBouncesInDateRangeParams{
		StartDate: sql.NullString{String: fromStr, Valid: true},
		EndDate:   sql.NullString{String: toStr, Valid: true},
	})
	if err != nil {
		l.Errorf("Failed to count bounces: %v", err)
		bounceCount = 0
	}

	complaintCount, err := l.svcCtx.DB.CountComplaintsInDateRange(l.ctx, db.CountComplaintsInDateRangeParams{
		StartDate: sql.NullString{String: fromStr, Valid: true},
		EndDate:   sql.NullString{String: toStr, Valid: true},
	})
	if err != nil {
		l.Errorf("Failed to count complaints: %v", err)
		complaintCount = 0
	}

	unsubscribedCount := int64(0)

	var openRate, clickRate float64
	if totalSent > 0 {
		openRate = float64(totalOpened) / float64(totalSent)
		clickRate = float64(totalClicked) / float64(totalSent)
	}

	return &types.EmailDashboardStatsResponse{
		TotalSent:         int(totalSent),
		TotalOpened:       int(totalOpened),
		TotalClicked:      int(totalClicked),
		TotalBounced:      int(bounceCount),
		TotalComplaints:   int(complaintCount),
		TotalUnsubscribed: int(unsubscribedCount),
		OpenRate:          openRate,
		ClickRate:         clickRate,
	}, nil
}
