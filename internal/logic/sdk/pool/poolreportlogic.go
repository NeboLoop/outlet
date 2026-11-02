package pool

import (
	"context"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/middleware"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PoolReportLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Take a bounce (RFC 3464 DSN) or complaint (RFC 5965 ARF) report
func NewPoolReportLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PoolReportLogic {
	return &PoolReportLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PoolReportLogic) PoolReport(req *types.PoolReportRequest) (resp *types.PoolReportResponse, err error) {
	orgID, ok := l.ctx.Value(middleware.OrgIDKey).(string)
	if !ok || orgID == "" {
		return nil, errorx.NewUnauthorizedError("API key required")
	}
	r, err := l.svcCtx.MailPool.IngestReport(l.ctx, orgID, []byte(req.Raw))
	if err != nil {
		return nil, err
	}
	return &types.PoolReportResponse{
		MessageId: r.MessageID, MailboxId: r.MailboxID, Tenant: r.Tenant, Email: r.Email, Kind: r.Kind, Hard: r.Hard,
	}, nil
}
