package imports

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CancelImportJobLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCancelImportJobLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelImportJobLogic {
	return &CancelImportJobLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CancelImportJobLogic) CancelImportJob(req *types.CancelImportJobRequest) (resp *types.Response, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	err = l.svcCtx.DB.CancelImportJob(l.ctx, db.CancelImportJobParams{
		ID:    req.Id,
		OrgID: orgID,
	})
	if err != nil {
		l.Errorf("Failed to cancel import job: %v", err)
		return &types.Response{Success: false, Message: "failed to cancel import job"}, nil
	}

	l.Infof("Cancelled import job: org=%s id=%s", orgID, req.Id)

	return &types.Response{Success: true, Message: "import job cancelled"}, nil
}
