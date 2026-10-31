package organizations

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAllOrganizationsStatsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetAllOrganizationsStatsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAllOrganizationsStatsLogic {
	return &GetAllOrganizationsStatsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAllOrganizationsStatsLogic) GetAllOrganizationsStats() (resp *types.OrgBulkStatsResponse, err error) {
	user, ok := access.User(l.ctx)
	if !ok {
		return nil, errorx.NewUnauthorizedError("Sign in first.")
	}
	// Stats only for the brands this person can reach.
	brands, err := access.Brands(l.ctx, l.svcCtx.DB, user.ID)
	if err != nil {
		return nil, err
	}
	reach := make(map[string]bool, len(brands))
	for _, b := range brands {
		reach[b.ID] = true
	}

	resp = &types.OrgBulkStatsResponse{
		Stats: make(map[string]types.OrgStats),
	}

	// Get contact counts for all orgs
	contactStats, err := l.svcCtx.DB.GetAllOrgsContactStats(l.ctx)
	if err != nil {
		l.Logger.Errorf("Failed to get contact stats: %v", err)
	} else {
		for _, stat := range contactStats {
			if !stat.OrgID.Valid {
				continue
			}
			orgID := stat.OrgID.String
			orgStats := resp.Stats[orgID]
			orgStats.TotalContacts = stat.TotalContacts
			resp.Stats[orgID] = orgStats
		}
	}

	// Get email stats for last 30 days for all orgs
	emailStats, err := l.svcCtx.DB.GetAllOrgsEmailStats30Days(l.ctx)
	if err != nil {
		l.Logger.Errorf("Failed to get email stats: %v", err)
	} else {
		for _, stat := range emailStats {
			orgStats := resp.Stats[stat.OrgID]
			orgStats.EmailsSent30d = toInt64(stat.EmailsSent)
			resp.Stats[stat.OrgID] = orgStats
		}
	}

	// Get list counts for all orgs
	listStats, err := l.svcCtx.DB.GetAllOrgsListCounts(l.ctx)
	if err != nil {
		l.Logger.Errorf("Failed to get list stats: %v", err)
	} else {
		for _, stat := range listStats {
			orgStats := resp.Stats[stat.OrgID]
			orgStats.ListCount = stat.ListCount
			resp.Stats[stat.OrgID] = orgStats
		}
	}

	for id := range resp.Stats {
		if !reach[id] {
			delete(resp.Stats, id)
		}
	}
	return resp, nil
}
