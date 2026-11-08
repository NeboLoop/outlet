package validation

import (
	"context"

	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/middleware"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ValidateEmailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewValidateEmailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ValidateEmailLogic {
	return &ValidateEmailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ValidateEmailLogic) ValidateEmail(req *types.ValidateEmailRequest) (resp *types.ValidationResult, err error) {
	orgID, ok := l.ctx.Value(middleware.OrgIDKey).(string)
	if !ok || orgID == "" {
		return nil, errorx.NewUnauthorizedError("invalid API key")
	}
	if req.Email == "" {
		return nil, errorx.NewBadRequestError("email is required")
	}
	return l.svcCtx.Validation.Validate(l.ctx, orgID, req.Email, "api")
}
