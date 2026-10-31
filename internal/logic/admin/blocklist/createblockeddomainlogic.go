package blocklist

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateBlockedDomainLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateBlockedDomainLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateBlockedDomainLogic {
	return &CreateBlockedDomainLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateBlockedDomainLogic) CreateBlockedDomain(req *types.CreateBlockedDomainRequest) (resp *types.BlockedDomainInfo, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	reason := sql.NullString{}
	if req.Reason != "" {
		reason = sql.NullString{String: req.Reason, Valid: true}
	}

	domain, err := l.svcCtx.DB.CreateBlockedDomain(l.ctx, db.CreateBlockedDomainParams{
		OrgID:  orgID,
		Domain: req.Domain,
		Reason: reason,
	})
	if err != nil {
		l.Errorf("Failed to create blocked domain: %v", err)
		return nil, fmt.Errorf("failed to create blocked domain: %w", err)
	}

	blockAttempts := 0
	if domain.BlockAttempts.Valid {
		blockAttempts = int(domain.BlockAttempts.Int64)
	}

	return &types.BlockedDomainInfo{
		Id:            strconv.FormatInt(domain.ID, 10),
		OrgId:         domain.OrgID,
		Domain:        domain.Domain,
		Reason:        utils.FormatNullString(domain.Reason),
		BlockAttempts: blockAttempts,
		CreatedAt:     utils.FormatNullString(domain.CreatedAt),
		UpdatedAt:     utils.FormatNullString(domain.UpdatedAt),
	}, nil
}
