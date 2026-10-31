package campaigns

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteCampaignLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteCampaignLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteCampaignLogic {
	return &DeleteCampaignLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteCampaignLogic) DeleteCampaign(req *types.DeleteCampaignRequest) (resp *types.AnalyticsResponse, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	err = l.svcCtx.DB.DeleteCampaign(l.ctx, db.DeleteCampaignParams{
		ID:    req.Id,
		OrgID: orgID,
	})
	if err != nil {
		l.Errorf("Failed to delete campaign: %v", err)
		return nil, err
	}

	return &types.AnalyticsResponse{
		Success: true,
		Message: "Campaign deleted successfully",
	}, nil
}
