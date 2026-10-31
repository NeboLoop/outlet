package imports

import (
	"context"
	"strconv"

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
		listID := ""
		if job.ListID.Valid {
			listID = strconv.FormatInt(job.ListID.Int64, 10)
		}

		status := "pending"
		if job.Status.Valid {
			status = job.Status.String
		}

		jobInfos = append(jobInfos, types.ImportJobInfo{
			Id:            job.ID,
			OrgId:         job.OrgID,
			ListId:        listID,
			Type:          job.Type,
			Status:        status,
			Filename:      job.Filename,
			TotalRows:     int(job.TotalRows.Int64),
			ProcessedRows: int(job.ProcessedRows.Int64),
			SuccessCount:  int(job.SuccessCount.Int64),
			ErrorCount:    int(job.ErrorCount.Int64),
		})
	}

	return &types.ListImportJobsResponse{
		Jobs:  jobInfos,
		Total: len(jobInfos),
		Page:  page,
		Limit: limit,
	}, nil
}
