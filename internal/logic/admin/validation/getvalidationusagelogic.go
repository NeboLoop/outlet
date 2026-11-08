package validation

import (
	"context"
	"database/sql"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	validationsvc "github.com/outlet-sh/outlet/internal/services/validation"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetValidationUsageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetValidationUsageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetValidationUsageLogic {
	return &GetValidationUsageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetValidationUsageLogic) GetValidationUsage(req *types.ValidationUsageRequest) (resp *types.ValidationUsageResponse, err error) {
	orgID, err := orgIDFrom(l.ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if req.From != "" {
		if from, err = time.Parse(time.DateOnly, req.From); err != nil {
			return nil, errorx.NewBadRequestError("from must be YYYY-MM-DD")
		}
	}
	if req.To != "" {
		if to, err = time.Parse(time.DateOnly, req.To); err != nil {
			return nil, errorx.NewBadRequestError("to must be YYYY-MM-DD")
		}
	}
	if to.Before(from) {
		return nil, errorx.NewBadRequestError("to must not be before from")
	}

	rows, err := l.svcCtx.DB.CountValidationUsageBySource(l.ctx, db.CountValidationUsageBySourceParams{
		OrgID:     orgID,
		FromTime:  sql.NullString{String: from.Format(validationsvc.TimeLayout), Valid: true},
		UntilTime: sql.NullString{String: to.AddDate(0, 0, 1).Format(validationsvc.TimeLayout), Valid: true},
	})
	if err != nil {
		return nil, err
	}

	resp = &types.ValidationUsageResponse{
		From:     from.Format(time.DateOnly),
		To:       to.Format(time.DateOnly),
		BySource: map[string]int{"api": 0, "admin": 0, "job": 0, "mcp": 0},
	}
	for _, row := range rows {
		resp.BySource[row.Source] = int(row.Count)
		resp.Total += int(row.Count)
	}
	return resp, nil
}
