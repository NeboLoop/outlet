package validation

import (
	"context"
	"github.com/outlet-sh/outlet/internal/access"

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
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}
	page, limit := paging(req.Page, req.Limit, 50)
	results, total, err := l.svcCtx.Validation.Results(l.ctx, orgID, req.Id, req.Status, limit, (page-1)*limit)
	if err != nil {
		return nil, err
	}
	return &types.ListValidationResultsResponse{Results: results, Total: total, Page: page, Limit: limit}, nil
}
