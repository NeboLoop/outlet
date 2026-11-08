package validation

import (
	"context"
	"database/sql"
	"slices"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ApplyValidationJobLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewApplyValidationJobLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApplyValidationJobLogic {
	return &ApplyValidationJobLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ApplyValidationJobLogic) ApplyValidationJob(req *types.ApplyValidationJobRequest) (resp *types.ApplyValidationJobResponse, err error) {
	orgID, err := orgIDFrom(l.ctx)
	if err != nil {
		return nil, err
	}
	if len(req.Statuses) == 0 {
		return nil, errorx.NewBadRequestError("statuses is required")
	}
	for _, status := range req.Statuses {
		if !slices.Contains(suppressibleStatuses, status) {
			return nil, errorx.NewBadRequestError("invalid status: " + status)
		}
	}
	if _, err := l.svcCtx.Validation.Job(l.ctx, orgID, req.Id); err != nil {
		return nil, err
	}

	n, err := l.svcCtx.DB.SuppressValidationJobResults(l.ctx, db.SuppressValidationJobResultsParams{
		JobID:    sql.NullString{String: req.Id, Valid: true},
		OrgID:    orgID,
		Statuses: req.Statuses,
	})
	if err != nil {
		return nil, err
	}
	return &types.ApplyValidationJobResponse{Suppressed: int(n)}, nil
}
