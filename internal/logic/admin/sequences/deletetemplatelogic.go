package sequences

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteTemplateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteTemplateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteTemplateLogic {
	return &DeleteTemplateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteTemplateLogic) DeleteTemplate(req *types.DeleteTemplateRequest) (resp *types.AnalyticsResponse, err error) {
	if _, err := requireBrandOf(l.ctx, l.svcCtx, l.svcCtx.DB.GetTemplateOrgID, req.Id, access.Member); err != nil {
		return nil, err
	}

	err = l.svcCtx.DB.DeleteTemplate(l.ctx, req.Id)
	if err != nil {
		return nil, err
	}

	return &types.AnalyticsResponse{
		Success: true,
		Message: "Template deleted successfully",
	}, nil
}
