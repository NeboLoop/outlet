package validation

import (
	"context"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateValidationJobLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateValidationJobLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateValidationJobLogic {
	return &CreateValidationJobLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateValidationJobLogic) CreateValidationJob(req *types.CreateValidationJobRequest) (resp *types.ValidationJobInfo, err error) {
	orgID, err := orgIDFrom(l.ctx)
	if err != nil {
		return nil, err
	}
	return l.svcCtx.Validation.CreateJob(l.ctx, orgID, req.Name, req.ListId, req.Emails)
}
