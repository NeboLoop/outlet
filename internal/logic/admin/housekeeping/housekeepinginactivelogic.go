package housekeeping

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type HousekeepingInactiveLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewHousekeepingInactiveLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HousekeepingInactiveLogic {
	return &HousekeepingInactiveLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *HousekeepingInactiveLogic) HousekeepingInactive(req *types.HousekeepingInactiveRequest) (resp *types.HousekeepingResponse, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Admin)
	if err != nil {
		return nil, err
	}

	// Currently uses fixed 90 day window for inactive contact cleanup
	noOpensDays := 90

	l.Infof("HousekeepingInactive called for org %s: NoOpensDays=%d, NoClicksDays=%d, DryRun=%v",
		orgID, noOpensDays, req.NoClicksDays, req.DryRun)

	if req.DryRun {
		// Dry run - just count affected contacts
		count, err := l.svcCtx.DB.CountInactiveContacts90Days(l.ctx, sql.NullString{String: orgID, Valid: true})
		if err != nil {
			l.Errorf("Failed to count inactive contacts: %v", err)
			return nil, err
		}

		return &types.HousekeepingResponse{
			AffectedCount: int(count),
			DryRun:        true,
			Message:       fmt.Sprintf("Would delete %d contacts with no opens in %d days", count, noOpensDays),
		}, nil
	}

	// Actually delete inactive contacts
	deleted, err := l.svcCtx.DB.DeleteInactiveContacts90Days(l.ctx, sql.NullString{String: orgID, Valid: true})
	if err != nil {
		l.Errorf("Failed to delete inactive contacts: %v", err)
		return nil, err
	}

	l.Infof("Deleted %d inactive contacts with no opens in %d days for org %s", deleted, noOpensDays, orgID)

	return &types.HousekeepingResponse{
		AffectedCount: int(deleted),
		DryRun:        false,
		Message:       fmt.Sprintf("Deleted %d contacts with no opens in %d days", deleted, noOpensDays),
	}, nil
}
