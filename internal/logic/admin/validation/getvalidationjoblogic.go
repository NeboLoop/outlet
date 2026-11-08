package validation

import (
	"context"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetValidationJobLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetValidationJobLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetValidationJobLogic {
	return &GetValidationJobLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetValidationJobLogic) GetValidationJob(req *types.ValidationJobIdRequest) (resp *types.ValidationJobInfo, err error) {
	orgID, err := orgIDFrom(l.ctx)
	if err != nil {
		return nil, err
	}
	return l.svcCtx.Validation.Job(l.ctx, orgID, req.Id)
}
