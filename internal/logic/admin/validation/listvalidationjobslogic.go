package validation

import (
	"context"

	"github.com/outlet-sh/outlet/internal/db"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListValidationJobsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListValidationJobsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListValidationJobsLogic {
	return &ListValidationJobsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListValidationJobsLogic) ListValidationJobs(req *types.ListValidationJobsRequest) (resp *types.ListValidationJobsResponse, err error) {
	orgID, err := orgIDFrom(l.ctx)
	if err != nil {
		return nil, err
	}
	page, limit := paging(req.Page, req.Limit, 20)

	jobs, err := l.svcCtx.DB.ListValidationJobs(l.ctx, db.ListValidationJobsParams{
		OrgID:      orgID,
		PageSize:   int64(limit),
		PageOffset: int64((page - 1) * limit),
	})
	if err != nil {
		return nil, err
	}
	total, err := l.svcCtx.DB.CountValidationJobs(l.ctx, orgID)
	if err != nil {
		return nil, err
	}

	infos := make([]types.ValidationJobInfo, 0, len(jobs))
	for _, job := range jobs {
		info, err := l.svcCtx.Validation.JobInfo(l.ctx, job)
		if err != nil {
			return nil, err
		}
		infos = append(infos, info)
	}
	return &types.ListValidationJobsResponse{Jobs: infos, Total: int(total), Page: page, Limit: limit}, nil
}
