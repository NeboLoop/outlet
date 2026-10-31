package blocklist

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type BulkBlockDomainsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBulkBlockDomainsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BulkBlockDomainsLogic {
	return &BulkBlockDomainsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BulkBlockDomainsLogic) BulkBlockDomains(req *types.BulkBlockDomainsRequest) (resp *types.BulkBlockDomainsResponse, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	reason := sql.NullString{}
	if req.Reason != "" {
		reason = sql.NullString{String: req.Reason, Valid: true}
	}

	var successCount int
	var failedCount int
	var errors []string

	for _, domain := range req.Domains {
		err := l.svcCtx.DB.BulkInsertBlockedDomains(l.ctx, db.BulkInsertBlockedDomainsParams{
			OrgID:  orgID,
			Domain: domain,
			Reason: reason,
		})
		if err != nil {
			l.Errorf("Failed to block domain %s: %v", domain, err)
			failedCount++
			errors = append(errors, fmt.Sprintf("Failed to block domain %s: %v", domain, err))
		} else {
			successCount++
		}
	}

	return &types.BulkBlockDomainsResponse{
		Success: successCount,
		Failed:  failedCount,
		Errors:  errors,
	}, nil
}
