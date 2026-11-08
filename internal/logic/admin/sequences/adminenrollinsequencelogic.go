package sequences

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	sdksequences "github.com/outlet-sh/outlet/internal/logic/sdk/sequences"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminEnrollInSequenceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Enroll a contact of the brand in one of its sequences, as the signed-in member (the same enrollment as the SDK route)
func NewAdminEnrollInSequenceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminEnrollInSequenceLogic {
	return &AdminEnrollInSequenceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminEnrollInSequenceLogic) AdminEnrollInSequence(req *types.EnrollSequenceRequest) (resp *types.EnrollSequenceResponse, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}
	return sdksequences.Enroll(l.ctx, l.svcCtx, orgID, req)
}
