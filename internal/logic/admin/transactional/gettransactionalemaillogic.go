package transactional

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTransactionalEmailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetTransactionalEmailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTransactionalEmailLogic {
	return &GetTransactionalEmailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetTransactionalEmailLogic) GetTransactionalEmail(req *types.GetTransactionalEmailRequest) (resp *types.TransactionalEmailInfo, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	email, err := l.svcCtx.DB.GetTransactionalEmail(l.ctx, db.GetTransactionalEmailParams{
		ID:    req.Id,
		OrgID: orgID,
	})
	if err != nil {
		l.Errorf("Failed to get transactional email: %v", err)
		return nil, err
	}

	info := transactionalEmailToInfo(email)
	return &info, nil
}
