package imports

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListImportJobsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListImportJobsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListImportJobsLogic {
	return &ListImportJobsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListImportJobsLogic) ListImportJobs(req *types.ListImportJobsRequest) (resp *types.ListImportJobsResponse, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	limit := req.Limit
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	jobs, err := l.svcCtx.DB.ListImportJobs(l.ctx, db.ListImportJobsParams{
		OrgID:      orgID,
		PageOffset: int64(offset),
		PageSize:   int64(limit),
	})
	if err != nil {
		l.Errorf("Failed to list import jobs: %v", err)
		return nil, err
	}

	var jobInfos []types.ImportJobInfo
	for _, job := range jobs {
		jobInfos = append(jobInfos, jobInfo(job))
	}

	return &types.ListImportJobsResponse{
		Jobs:  jobInfos,
		Total: len(jobInfos),
		Page:  page,
		Limit: limit,
	}, nil
}
