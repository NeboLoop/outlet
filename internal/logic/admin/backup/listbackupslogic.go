package backup

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListBackupsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListBackupsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListBackupsLogic {
	return &ListBackupsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListBackupsLogic) ListBackups(req *types.ListBackupsRequest) (resp *types.ListBackupsResponse, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return nil, err
	}

	// Calculate offset
	offset := int64((req.Page - 1) * req.PageSize)

	// Get backups from database
	backups, err := l.svcCtx.DB.ListBackups(l.ctx, db.ListBackupsParams{
		OffsetVal: offset,
		LimitVal:  int64(req.PageSize),
	})
	if err != nil {
		return nil, err
	}

	// Get total count
	total, err := l.svcCtx.DB.CountBackups(l.ctx)
	if err != nil {
		return nil, err
	}

	// Convert to response types
	backupInfos := make([]types.BackupInfo, len(backups))
	for i, b := range backups {
		backupInfos[i] = dbBackupToType(b)
	}

	return &types.ListBackupsResponse{
		Backups: backupInfos,
		Total:   int(total),
		Page:    req.Page,
	}, nil
}
