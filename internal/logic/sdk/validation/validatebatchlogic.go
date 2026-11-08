package validation

import (
	"context"
	"fmt"
	"sync"

	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/middleware"
	validationsvc "github.com/outlet-sh/outlet/internal/services/validation"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ValidateBatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewValidateBatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ValidateBatchLogic {
	return &ValidateBatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ValidateBatchLogic) ValidateBatch(req *types.ValidateBatchRequest) (resp *types.ValidateBatchResponse, err error) {
	orgID, ok := l.ctx.Value(middleware.OrgIDKey).(string)
	if !ok || orgID == "" {
		return nil, errorx.NewUnauthorizedError("invalid API key")
	}
	emails := validationsvc.CleanEmails(req.Emails)
	if len(emails) == 0 {
		return nil, errorx.NewBadRequestError("emails is required")
	}
	if len(emails) > validationsvc.MaxBatchEmails {
		return nil, errorx.NewBadRequestError(fmt.Sprintf("at most %d emails per batch", validationsvc.MaxBatchEmails))
	}

	// Check up to 10 addresses at a time; results keep request order.
	results := make([]types.ValidationResult, len(emails))
	errs := make([]error, len(emails))
	sem := make(chan struct{}, 10)
	var wg sync.WaitGroup
	for i, email := range emails {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			res, err := l.svcCtx.Validation.Validate(l.ctx, orgID, email, "api")
			if err != nil {
				errs[i] = err
				return
			}
			results[i] = *res
		}()
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return &types.ValidateBatchResponse{Results: results}, nil
}
