package transactional

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteTransactionalEmailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteTransactionalEmailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteTransactionalEmailLogic {
	return &DeleteTransactionalEmailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteTransactionalEmailLogic) DeleteTransactionalEmail(req *types.DeleteTransactionalEmailRequest) (resp *types.AnalyticsResponse, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	err = l.svcCtx.DB.DeleteTransactionalEmail(l.ctx, db.DeleteTransactionalEmailParams{
		ID:    req.Id,
		OrgID: orgID,
	})
	if err != nil {
		l.Errorf("Failed to delete transactional email: %v", err)
		return nil, err
	}

	return &types.AnalyticsResponse{
		Success: true,
		Message: "Transactional email deleted successfully",
	}, nil
}
