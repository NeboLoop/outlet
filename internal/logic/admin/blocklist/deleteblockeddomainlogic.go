package blocklist

import (
	"context"
	"fmt"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteBlockedDomainLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteBlockedDomainLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteBlockedDomainLogic {
	return &DeleteBlockedDomainLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteBlockedDomainLogic) DeleteBlockedDomain(req *types.DeleteBlockedDomainRequest) (resp *types.Response, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	domainID, err := strconv.ParseInt(req.Id, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid domain ID: %w", err)
	}

	err = l.svcCtx.DB.DeleteBlockedDomainByID(l.ctx, db.DeleteBlockedDomainByIDParams{
		ID:    domainID,
		OrgID: orgID,
	})
	if err != nil {
		l.Errorf("Failed to delete blocked domain: %v", err)
		return nil, fmt.Errorf("failed to delete blocked domain: %w", err)
	}

	return &types.Response{
		Success: true,
		Message: "Blocked domain deleted successfully",
	}, nil
}
