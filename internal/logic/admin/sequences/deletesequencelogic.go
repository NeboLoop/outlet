package sequences

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteSequenceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteSequenceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteSequenceLogic {
	return &DeleteSequenceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteSequenceLogic) DeleteSequence(req *types.GetSequenceRequest) (resp *types.AnalyticsResponse, err error) {
	if _, err := requireBrandOf(l.ctx, l.svcCtx, l.svcCtx.DB.GetSequenceOrgID, req.Id, access.Member); err != nil {
		return nil, err
	}

	err = l.svcCtx.DB.DeleteSequence(l.ctx, req.Id)
	if err != nil {
		l.Errorf("Failed to delete sequence %s: %v", req.Id, err)
		return nil, err
	}

	return &types.AnalyticsResponse{
		Success: true,
		Message: "Sequence deleted successfully",
	}, nil
}
