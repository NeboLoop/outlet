package validation

import (
	"context"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type StartValidationJobLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewStartValidationJobLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StartValidationJobLogic {
	return &StartValidationJobLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *StartValidationJobLogic) StartValidationJob(req *types.ValidationJobIdRequest) (resp *types.ValidationJobInfo, err error) {
	orgID, err := orgIDFrom(l.ctx)
	if err != nil {
		return nil, err
	}
	return l.svcCtx.Validation.StartJob(l.ctx, orgID, req.Id)
}
