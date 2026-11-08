package validation

import (
	"context"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AppendValidationJobEmailsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAppendValidationJobEmailsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AppendValidationJobEmailsLogic {
	return &AppendValidationJobEmailsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AppendValidationJobEmailsLogic) AppendValidationJobEmails(req *types.AppendValidationJobEmailsRequest) (resp *types.ValidationJobInfo, err error) {
	orgID, err := orgIDFrom(l.ctx)
	if err != nil {
		return nil, err
	}
	return l.svcCtx.Validation.AppendEmails(l.ctx, orgID, req.Id, req.Emails)
}
