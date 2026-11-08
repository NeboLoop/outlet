package sending

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteSendingPathwayLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteSendingPathwayLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteSendingPathwayLogic {
	return &DeleteSendingPathwayLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteSendingPathwayLogic) DeleteSendingPathway(req *types.SendingPathwayIDRequest) (resp *types.Response, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Admin)
	if err != nil {
		return nil, err
	}
	if err := l.svcCtx.EmailService.DeletePathway(l.ctx, orgID, req.Id); err != nil {
		return nil, err
	}
	return &types.Response{Success: true, Message: "Sending pathway removed"}, nil
}
