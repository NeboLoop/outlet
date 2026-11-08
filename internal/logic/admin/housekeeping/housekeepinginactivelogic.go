package housekeeping

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
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

	// A contact is inactive with no open in NoOpensDays and no click in
	// NoClicksDays, across every kind of email the brand sends.
	opensDays, clicksDays := req.NoOpensDays, req.NoClicksDays
	if opensDays < 0 || clicksDays < 0 {
		return nil, errorx.NewBadRequestError("Days can't be negative.")
	}

	l.Infof("HousekeepingInactive called for org %s: NoOpensDays=%d, NoClicksDays=%d, DryRun=%v",
		orgID, opensDays, clicksDays, req.DryRun)

	org := sql.NullString{String: orgID, Valid: true}
	opensModifier := fmt.Sprintf("-%d days", opensDays)
	clicksModifier := fmt.Sprintf("-%d days", clicksDays)
	what := fmt.Sprintf("contacts with no opens in %d days and no clicks in %d days", opensDays, clicksDays)

	if req.DryRun {
		count, err := l.svcCtx.DB.CountInactiveContacts(l.ctx, db.CountInactiveContactsParams{
			OrgID: org, OpensModifier: opensModifier, ClicksModifier: clicksModifier,
		})
		if err != nil {
			l.Errorf("Failed to count inactive contacts: %v", err)
			return nil, err
		}

		return &types.HousekeepingResponse{
			AffectedCount: int(count),
			DryRun:        true,
			Message:       fmt.Sprintf("Would delete %d %s", count, what),
		}, nil
	}

	deleted, err := l.svcCtx.DB.DeleteInactiveContacts(l.ctx, db.DeleteInactiveContactsParams{
		OrgID: org, OpensModifier: opensModifier, ClicksModifier: clicksModifier,
	})
	if err != nil {
		l.Errorf("Failed to delete inactive contacts: %v", err)
		return nil, err
	}

	l.Infof("Deleted %d %s for org %s", deleted, what, orgID)

	return &types.HousekeepingResponse{
		AffectedCount: int(deleted),
		DryRun:        false,
		Message:       fmt.Sprintf("Deleted %d %s", deleted, what),
	}, nil
}
