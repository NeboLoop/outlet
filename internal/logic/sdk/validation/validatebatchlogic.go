package validation

import (
	"context"
	"fmt"
	"time"

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

// ValidateBatch accepts up to MaxBatchEmails addresses at once as a
// validation job, which the validation worker checks in the background (each
// check can take its probe's 20 seconds). The job's ID fetches its results.
func (l *ValidateBatchLogic) ValidateBatch(req *types.ValidateBatchRequest) (resp *types.ValidationJobInfo, err error) {
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
	job, err := l.svcCtx.Validation.CreateJob(l.ctx, orgID, "API batch "+time.Now().UTC().Format("2006-01-02 15:04:05"), "", emails)
	if err != nil {
		return nil, err
	}
	return l.svcCtx.Validation.StartJob(l.ctx, orgID, job.Id)
}
