package imports

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetImportJobLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetImportJobLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetImportJobLogic {
	return &GetImportJobLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetImportJobLogic) GetImportJob(req *types.GetImportJobRequest) (resp *types.ImportJobInfo, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	job, err := l.svcCtx.DB.GetImportJob(l.ctx, db.GetImportJobParams{
		ID:    req.Id,
		OrgID: orgID,
	})
	if err != nil {
		l.Errorf("Failed to get import job: %v", err)
		return nil, err
	}

	info := jobInfo(job)
	return &info, nil
}
