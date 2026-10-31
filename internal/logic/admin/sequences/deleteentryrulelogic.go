package sequences

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteEntryRuleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteEntryRuleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteEntryRuleLogic {
	return &DeleteEntryRuleLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteEntryRuleLogic) DeleteEntryRule(req *types.DeleteEntryRuleRequest) (resp *types.AnalyticsResponse, err error) {
	if _, err := access.RequireOwner(l.ctx, l.svcCtx.DB, access.EntryRule, req.Id, access.Member); err != nil {
		return nil, err
	}

	err = l.svcCtx.DB.DeleteEntryRule(l.ctx, req.Id)
	if err != nil {
		l.Errorf("Failed to delete entry rule: %v", err)
		return nil, err
	}

	return &types.AnalyticsResponse{
		Success: true,
		Message: "Entry rule deleted successfully",
	}, nil
}
