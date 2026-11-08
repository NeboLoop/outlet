package imports

import (
	"context"
	"database/sql"
	"encoding/csv"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/workers"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type ImportListSubscribersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewImportListSubscribersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ImportListSubscribersLogic {
	return &ImportListSubscribersLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ImportListSubscribers saves the CSV and queues an import job; the import
// worker adds each row to the list, active at once.
func (l *ImportListSubscribersLogic) ImportListSubscribers(req *types.ImportListSubscribersRequest) (resp *types.ImportJobInfo, err error) {
	listID, err := strconv.ParseInt(req.Id, 10, 64)
	if err != nil {
		return nil, errorx.NewBadRequestError("Invalid list ID.")
	}
	orgID, err := access.RequireOwner(l.ctx, l.svcCtx.DB, access.List, req.Id, access.Member)
	if err != nil {
		return nil, err
	}

	header, err := csv.NewReader(strings.NewReader(req.Csv)).Read()
	if err != nil {
		return nil, errorx.NewBadRequestError("The file isn't a CSV with a header row.")
	}
	if !slices.ContainsFunc(header, func(col string) bool { return strings.EqualFold(strings.TrimSpace(col), "email") }) {
		return nil, errorx.NewBadRequestError("The CSV needs an email column (and may have a name column).")
	}

	jobID := uuid.NewString()
	if err := os.MkdirAll(l.svcCtx.UploadDir, 0o700); err != nil {
		l.Errorf("Failed to create upload directory: %v", err)
		return nil, errorx.NewInternalError("Couldn't save the file.")
	}
	if err := os.WriteFile(workers.ImportFile(l.svcCtx.UploadDir, jobID), []byte(req.Csv), 0o600); err != nil {
		l.Errorf("Failed to save import file: %v", err)
		return nil, errorx.NewInternalError("Couldn't save the file.")
	}

	name := filepath.Base(strings.TrimSpace(req.Filename))
	if name == "." || name == "/" || name == "" {
		name = "import.csv"
	}
	job, err := l.svcCtx.DB.CreateImportJob(l.ctx, db.CreateImportJobParams{
		ID:       jobID,
		OrgID:    orgID,
		ListID:   sql.NullInt64{Int64: listID, Valid: true},
		Type:     "subscribers",
		Filename: name,
	})
	if err != nil {
		_ = os.Remove(workers.ImportFile(l.svcCtx.UploadDir, jobID))
		l.Errorf("Failed to create import job: %v", err)
		return nil, errorx.NewInternalError("Couldn't start the import.")
	}
	info := jobInfo(job)
	return &info, nil
}
