package blocklist

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ClearSuppressionListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewClearSuppressionListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ClearSuppressionListLogic {
	return &ClearSuppressionListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ClearSuppressionListLogic) ClearSuppressionList() (resp *types.Response, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Admin)
	if err != nil {
		return nil, err
	}

	err = l.svcCtx.DB.ClearSuppressionList(l.ctx, orgID)
	if err != nil {
		l.Errorf("Failed to clear suppression list: %v", err)
		return nil, err
	}

	return &types.Response{
		Success: true,
		Message: "Suppression list cleared",
	}, nil
}
