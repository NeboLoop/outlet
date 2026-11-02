package pool

import (
	"context"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/middleware"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PoolVerifyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Check addresses before their first email (no SMTP probing)
func NewPoolVerifyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PoolVerifyLogic {
	return &PoolVerifyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PoolVerifyLogic) PoolVerify(req *types.PoolVerifyRequest) (resp *types.PoolVerifyResponse, err error) {
	orgID, ok := l.ctx.Value(middleware.OrgIDKey).(string)
	if !ok || orgID == "" {
		return nil, errorx.NewUnauthorizedError("API key required")
	}
	results, err := l.svcCtx.MailPool.Verify(l.ctx, req.Emails)
	if err != nil {
		return nil, err
	}
	resp = &types.PoolVerifyResponse{Results: make([]types.PoolVerifyResult, 0, len(results))}
	for _, r := range results {
		resp.Results = append(resp.Results, types.PoolVerifyResult{
			Email: r.Email, Syntax: r.Syntax, Mx: r.MX, Disposable: r.Disposable, Role: r.Role,
			Blocked: r.Blocked, Ok: r.OK, Reason: r.Reason,
		})
	}
	return resp, nil
}
