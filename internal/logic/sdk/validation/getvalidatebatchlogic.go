package validation

import (
	"context"

	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/middleware"
	validationsvc "github.com/outlet-sh/outlet/internal/services/validation"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetValidateBatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetValidateBatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetValidateBatchLogic {
	return &GetValidateBatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetValidateBatch returns a batch's job and its results so far.
func (l *GetValidateBatchLogic) GetValidateBatch(req *types.ValidationJobIdRequest) (resp *types.ValidateBatchResponse, err error) {
	orgID, ok := l.ctx.Value(middleware.OrgIDKey).(string)
	if !ok || orgID == "" {
		return nil, errorx.NewUnauthorizedError("invalid API key")
	}
	job, err := l.svcCtx.Validation.Job(l.ctx, orgID, req.Id)
	if err != nil {
		return nil, err
	}
	results, _, err := l.svcCtx.Validation.Results(l.ctx, orgID, req.Id, "", validationsvc.MaxBatchEmails, 0)
	if err != nil {
		return nil, err
	}
	return &types.ValidateBatchResponse{Job: *job, Results: results}, nil
}
