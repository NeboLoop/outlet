package organizations

import (
	"context"
	"database/sql"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetDashboardStatsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetDashboardStatsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDashboardStatsLogic {
	return &GetDashboardStatsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetDashboardStatsLogic) GetDashboardStats(req *types.GetDashboardStatsRequest) (resp *types.DashboardStatsResponse, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, req.Id, access.Member); err != nil {
		return nil, err
	}

	orgID := req.Id

	resp = &types.DashboardStatsResponse{}

	// Get subscriber stats
	subscriberStats, err := l.svcCtx.DB.GetDashboardSubscriberStats(l.ctx, sql.NullString{String: orgID, Valid: true})
	if err == nil {
		resp.TotalSubscribers = subscriberStats.Total
		if subscriberStats.New30d.Valid {
			resp.NewSubscribers30d = int64(subscriberStats.New30d.Float64)
		}
		if subscriberStats.New7d.Valid {
			resp.NewSubscribers7d = int64(subscriberStats.New7d.Float64)
		}
	}

	// Get subscriber growth percentage
	growthStats, err := l.svcCtx.DB.GetDashboardSubscriberGrowth(l.ctx, sql.NullString{String: orgID, Valid: true})
	if err == nil {
		currentPeriod := toInt64(growthStats.CurrentPeriod)
		previousPeriod := toInt64(growthStats.PreviousPeriod)
		if previousPeriod > 0 {
			resp.SubscriberGrowthPct = float64(currentPeriod-previousPeriod) / float64(previousPeriod) * 100
		}
	}

	// Get email stats for last 30 days
	emailStats, err := l.svcCtx.DB.GetDashboardEmailStats30Days(l.ctx, orgID)
	if err == nil {
		resp.EmailsSent30d = toInt64(emailStats.EmailsSent)
		resp.EmailsOpened30d = toInt64(emailStats.EmailsOpened)
		resp.EmailsClicked30d = toInt64(emailStats.EmailsClicked)
		if resp.EmailsSent30d > 0 {
			resp.EmailOpenRate = float64(resp.EmailsOpened30d) / float64(resp.EmailsSent30d) * 100
		}
	}

	// Check lists
	lists, err := l.svcCtx.DB.ListEmailLists(l.ctx, orgID)
	resp.HasLists = err == nil && len(lists) > 0

	// Check if org has email configured (from_email set)
	org, err := l.svcCtx.DB.GetOrganizationByID(l.ctx, orgID)
	if err == nil {
		resp.EmailConfigured = org.FromEmail.Valid && org.FromEmail.String != ""
	}

	// MCP config check - check if user has any MCP API keys
	resp.HasMCPConfigured = false
	if user, ok := access.User(l.ctx); ok {
		mcpKeys, err := l.svcCtx.DB.ListMCPAPIKeysByUser(l.ctx, user.ID)
		if err == nil && len(mcpKeys) > 0 {
			// Check if any keys are active (not revoked)
			for _, key := range mcpKeys {
				if !key.RevokedAt.Valid {
					resp.HasMCPConfigured = true
					break
				}
			}
		}
	}

	return resp, nil
}

// toInt64 converts interface{} (from COALESCE results) to int64
func toInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case float64:
		return int64(val)
	default:
		return 0
	}
}
