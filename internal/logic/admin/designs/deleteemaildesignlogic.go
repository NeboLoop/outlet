package designs

import (
	"context"
	"errors"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteEmailDesignLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteEmailDesignLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteEmailDesignLogic {
	return &DeleteEmailDesignLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteEmailDesignLogic) DeleteEmailDesign(req *types.DeleteEmailDesignRequest) (resp *types.AnalyticsResponse, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	designID, err := strconv.ParseInt(req.Id, 10, 64)
	if err != nil {
		return nil, errors.New("invalid design ID")
	}

	err = l.svcCtx.DB.DeleteEmailDesign(l.ctx, db.DeleteEmailDesignParams{
		ID:    designID,
		OrgID: orgID,
	})
	if err != nil {
		l.Errorf("Failed to delete email design: %v", err)
		return nil, err
	}

	return &types.AnalyticsResponse{
		Success: true,
		Message: "Email design deleted successfully",
	}, nil
}
