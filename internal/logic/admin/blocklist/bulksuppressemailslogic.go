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

type BulkSuppressEmailsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBulkSuppressEmailsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BulkSuppressEmailsLogic {
	return &BulkSuppressEmailsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BulkSuppressEmailsLogic) BulkSuppressEmails(req *types.BulkSuppressEmailsRequest) (resp *types.BulkSuppressEmailsResponse, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	reason := sql.NullString{}
	if req.Reason != "" {
		reason = sql.NullString{String: req.Reason, Valid: true}
	}

	successCount := 0
	failedCount := 0
	var errors []string

	for _, email := range req.Emails {
		_, err := l.svcCtx.DB.AddToSuppressionList(l.ctx, db.AddToSuppressionListParams{
			OrgID:  orgID,
			Email:  email,
			Reason: reason,
			Source: sql.NullString{String: "manual", Valid: true},
		})
		if err != nil {
			failedCount++
			errors = append(errors, fmt.Sprintf("%s: %v", email, err))
			l.Errorf("Failed to suppress email %s: %v", email, err)
		} else {
			successCount++
		}
	}

	return &types.BulkSuppressEmailsResponse{
		Success: successCount,
		Failed:  failedCount,
		Errors:  errors,
	}, nil
}
