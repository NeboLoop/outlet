package blocklist

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddSuppressedEmailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddSuppressedEmailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddSuppressedEmailLogic {
	return &AddSuppressedEmailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddSuppressedEmailLogic) AddSuppressedEmail(req *types.AddSuppressedEmailRequest) (resp *types.SuppressedEmailInfo, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	reason := sql.NullString{}
	if req.Reason != "" {
		reason = sql.NullString{String: req.Reason, Valid: true}
	}

	suppressed, err := l.svcCtx.DB.AddToSuppressionList(l.ctx, db.AddToSuppressionListParams{
		OrgID:  orgID,
		Email:  req.Email,
		Reason: reason,
		Source: sql.NullString{String: "manual", Valid: true},
	})
	if err != nil {
		l.Errorf("Failed to add email to suppression list: %v", err)
		return nil, err
	}

	blockAttempts := 0
	if suppressed.BlockAttempts.Valid {
		blockAttempts = int(suppressed.BlockAttempts.Int64)
	}
	createdAt := ""
	if suppressed.CreatedAt.Valid {
		createdAt = suppressed.CreatedAt.String
	}

	return &types.SuppressedEmailInfo{
		Id:            strconv.FormatInt(suppressed.ID, 10),
		OrgId:         suppressed.OrgID,
		Email:         suppressed.Email,
		Reason:        suppressed.Reason.String,
		Source:        suppressed.Source.String,
		BlockAttempts: blockAttempts,
		CreatedAt:     createdAt,
	}, nil
}
