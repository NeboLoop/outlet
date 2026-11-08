package lists

import (
	"context"
	"database/sql"
	"time"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
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

// GetEmailDashboardStats reports what the brand's email did from req.From to
// req.To (whole UTC days, YYYY-MM-DD; the last 30 days by default).
func (l *GetEmailDashboardStatsLogic) GetEmailDashboardStats(req *types.EmailDashboardStatsRequest) (resp *types.EmailDashboardStatsResponse, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	fromTime := now.AddDate(0, 0, -30)
	toTime := now
	if req.From != "" {
		if fromTime, err = time.Parse(time.DateOnly, req.From); err != nil {
			return nil, errorx.NewBadRequestError("from is a date: YYYY-MM-DD")
		}
	}
	if req.To != "" {
		if toTime, err = time.Parse(time.DateOnly, req.To); err != nil {
			return nil, errorx.NewBadRequestError("to is a date: YYYY-MM-DD")
		}
		toTime = toTime.Add(24*time.Hour - time.Second)
	}

	// Stored times are UTC text, 'YYYY-MM-DD HH:MM:SS'
	stats, err := l.svcCtx.DB.GetBrandEmailStats(l.ctx, db.GetBrandEmailStatsParams{
		OrgID:     sql.NullString{String: orgID, Valid: true},
		StartDate: sql.NullString{String: fromTime.Format(time.DateTime), Valid: true},
		EndDate:   sql.NullString{String: toTime.Format(time.DateTime), Valid: true},
	})
	if err != nil {
		l.Errorf("Failed to get email stats: %v", err)
		return nil, errorx.NewInternalError("Couldn't load the email stats.")
	}

	var openRate, clickRate float64
	if stats.SentCount > 0 {
		openRate = float64(stats.OpenedCount) / float64(stats.SentCount)
		clickRate = float64(stats.ClickedCount) / float64(stats.SentCount)
	}

	return &types.EmailDashboardStatsResponse{
		TotalSent:         int(stats.SentCount),
		TotalOpened:       int(stats.OpenedCount),
		TotalClicked:      int(stats.ClickedCount),
		TotalBounced:      int(stats.BouncedCount),
		TotalComplaints:   int(stats.ComplaintCount),
		TotalUnsubscribed: int(stats.UnsubscribedCount),
		OpenRate:          openRate,
		ClickRate:         clickRate,
	}, nil
}
