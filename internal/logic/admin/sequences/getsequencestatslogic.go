package sequences

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSequenceStatsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetSequenceStatsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSequenceStatsLogic {
	return &GetSequenceStatsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSequenceStatsLogic) GetSequenceStats(req *types.SequenceStatsRequest) (resp *types.SequenceStatsResponse, err error) {
	if _, err := access.RequireOwner(l.ctx, l.svcCtx.DB, access.Sequence, req.Id, access.Member); err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(req.Id, 10, 64)
	if err != nil {
		l.Errorf("Invalid sequence ID %s: %v", req.Id, err)
		return nil, err
	}

	stats, err := l.svcCtx.DB.GetSequenceStats(l.ctx, sql.NullString{String: req.Id, Valid: true})
	if err != nil {
		l.Errorf("Failed to get sequence stats for %d: %v", id, err)
		return &types.SequenceStatsResponse{
			TotalSubscribers: 0,
			Completed:        0,
			Unsubscribed:     0,
			EmailsSent:       0,
			EmailsPending:    0,
		}, nil
	}

	return &types.SequenceStatsResponse{
		TotalSubscribers: int(stats.TotalSubscribers),
		Completed:        int(stats.Completed),
		Unsubscribed:     int(stats.Unsubscribed),
		EmailsSent:       int(stats.EmailsSent),
		EmailsPending:    int(stats.EmailsPending),
	}, nil
}
