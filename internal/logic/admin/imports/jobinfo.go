package imports

import (
	"strconv"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/types"
)

// jobInfo is an import job as the API shows it.
func jobInfo(job db.ImportJob) types.ImportJobInfo {
	listID := ""
	if job.ListID.Valid {
		listID = strconv.FormatInt(job.ListID.Int64, 10)
	}
	status := "pending"
	if job.Status.Valid {
		status = job.Status.String
	}
	return types.ImportJobInfo{
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
		SkipCount:     int(job.SkipCount.Int64),
		Errors:        job.Errors.String,
		StartedAt:     job.StartedAt.String,
		CompletedAt:   job.CompletedAt.String,
		CreatedAt:     job.CreatedAt.String,
	}
}
