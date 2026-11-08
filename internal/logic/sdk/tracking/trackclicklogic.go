package tracking

import (
	"context"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type TrackClickLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewTrackClickLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TrackClickLogic {
	return &TrackClickLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TrackClickLogic) TrackClick(req *types.TrackClickRequest) (resp *types.Response, err error) {
	if err := l.svcCtx.Tracking.RecordClick(l.ctx, req.Token, req.Url); err != nil {
		return &types.Response{Success: false, Message: "invalid token"}, nil
	}

	return &types.Response{Success: true}, nil
}
