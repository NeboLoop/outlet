package campaigns

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCampaignStatsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCampaignStatsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCampaignStatsLogic {
	return &GetCampaignStatsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCampaignStatsLogic) GetCampaignStats(req *types.GetCampaignRequest) (resp *types.CampaignStatsResponse, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	campaign, err := l.svcCtx.DB.GetCampaign(l.ctx, db.GetCampaignParams{
		ID:    req.Id,
		OrgID: orgID,
	})
	if err != nil {
		l.Errorf("Failed to get campaign: %v", err)
		return nil, err
	}

	links, err := l.svcCtx.DB.GetCampaignLinkStats(l.ctx, req.Id)
	if err != nil {
		l.Errorf("Failed to get campaign link stats: %v", err)
	}

	linkStats := make([]types.CampaignLinkStat, 0)
	for _, link := range links {
		linkStats = append(linkStats, types.CampaignLinkStat{
			Url:        link.LinkUrl,
			Name:       link.LinkName.String,
			ClickCount: int(link.ClickCount),
		})
	}

	info := campaignToInfo(campaign)
	return &types.CampaignStatsResponse{
		Campaign: info,
		Links:    linkStats,
	}, nil
}
