package validation

import (
	"context"
	"database/sql"

	"github.com/outlet-sh/outlet/internal/db"
	validationsvc "github.com/outlet-sh/outlet/internal/services/validation"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListValidationResultsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListValidationResultsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListValidationResultsLogic {
	return &ListValidationResultsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListValidationResultsLogic) ListValidationResults(req *types.ListValidationResultsRequest) (resp *types.ListValidationResultsResponse, err error) {
	orgID, err := orgIDFrom(l.ctx)
	if err != nil {
		return nil, err
	}
	if _, err := l.svcCtx.Validation.Job(l.ctx, orgID, req.Id); err != nil {
		return nil, err
	}
	page, limit := paging(req.Page, req.Limit, 50)
	jobID := sql.NullString{String: req.Id, Valid: true}

	rows, err := l.svcCtx.DB.ListValidationResults(l.ctx, db.ListValidationResultsParams{
		JobID:        jobID,
		OrgID:        orgID,
		FilterStatus: req.Status,
		PageSize:     int64(limit),
		PageOffset:   int64((page - 1) * limit),
	})
	if err != nil {
		return nil, err
	}
	total, err := l.svcCtx.DB.CountValidationResults(l.ctx, db.CountValidationResultsParams{
		JobID:        jobID,
		OrgID:        orgID,
		FilterStatus: req.Status,
	})
	if err != nil {
		return nil, err
	}

	results := make([]types.ValidationResult, 0, len(rows))
	for _, row := range rows {
		results = append(results, validationsvc.ToType(row))
	}
	return &types.ListValidationResultsResponse{Results: results, Total: int(total), Page: page, Limit: limit}, nil
}
