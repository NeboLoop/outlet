package sending

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListSendingPathwaysLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListSendingPathwaysLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListSendingPathwaysLogic {
	return &ListSendingPathwaysLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListSendingPathwaysLogic) ListSendingPathways() (resp *types.SendingPathwayList, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}
	ps, err := l.svcCtx.EmailService.ListPathways(l.ctx, orgID)
	if err != nil {
		return nil, err
	}
	out := &types.SendingPathwayList{Pathways: make([]types.SendingPathway, 0, len(ps))}
	for _, p := range ps {
		out.Pathways = append(out.Pathways, pathwayOut(p))
	}
	return out, nil
}
