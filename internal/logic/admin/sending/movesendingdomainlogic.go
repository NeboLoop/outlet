package sending

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type MoveSendingDomainLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewMoveSendingDomainLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MoveSendingDomainLogic {
	return &MoveSendingDomainLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MoveSendingDomainLogic) MoveSendingDomain(req *types.MoveDomainRequest) (resp *types.Response, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Admin)
	if err != nil {
		return nil, err
	}
	to, err := l.svcCtx.EmailService.MoveDomain(l.ctx, orgID, req.Id, req.PathwayId)
	if err != nil {
		return nil, err
	}
	return &types.Response{Success: true, Message: "Moved to " + to.Name}, nil
}
