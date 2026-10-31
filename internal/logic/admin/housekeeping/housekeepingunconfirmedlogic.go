package housekeeping

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type HousekeepingUnconfirmedLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewHousekeepingUnconfirmedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HousekeepingUnconfirmedLogic {
	return &HousekeepingUnconfirmedLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *HousekeepingUnconfirmedLogic) HousekeepingUnconfirmed(req *types.HousekeepingUnconfirmedRequest) (resp *types.HousekeepingResponse, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Admin)
	if err != nil {
		return nil, err
	}

	days := req.OlderThanDays
	if days <= 0 {
		days = 30
	}

	l.Infof("HousekeepingUnconfirmed called for org %s: OlderThanDays=%d, DryRun=%v",
		orgID, days, req.DryRun)

	// Build SQLite datetime modifier (e.g., "-30 days")
	daysModifier := fmt.Sprintf("-%d days", days)

	if req.DryRun {
		// Dry run - just count affected contacts
		count, err := l.svcCtx.DB.CountUnconfirmedContactsOlderThan(l.ctx, db.CountUnconfirmedContactsOlderThanParams{
			OrgID:        sql.NullString{String: orgID, Valid: true},
			DaysModifier: daysModifier,
		})
		if err != nil {
			l.Errorf("Failed to count unconfirmed contacts: %v", err)
			return nil, err
		}

		return &types.HousekeepingResponse{
			AffectedCount: int(count),
			DryRun:        true,
			Message:       fmt.Sprintf("Would delete %d unconfirmed contacts older than %d days", count, days),
		}, nil
	}

	// Actually delete unconfirmed contacts
	deleted, err := l.svcCtx.DB.DeleteUnconfirmedContactsOlderThan(l.ctx, db.DeleteUnconfirmedContactsOlderThanParams{
		OrgID:        sql.NullString{String: orgID, Valid: true},
		DaysModifier: daysModifier,
	})
	if err != nil {
		l.Errorf("Failed to delete unconfirmed contacts: %v", err)
		return nil, err
	}

	l.Infof("Deleted %d unconfirmed contacts older than %d days for org %s", deleted, days, orgID)

	return &types.HousekeepingResponse{
		AffectedCount: int(deleted),
		DryRun:        false,
		Message:       fmt.Sprintf("Deleted %d unconfirmed contacts older than %d days", deleted, days),
	}, nil
}
