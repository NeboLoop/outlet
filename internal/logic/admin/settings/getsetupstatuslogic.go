package settings

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSetupStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetSetupStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSetupStatusLogic {
	return &GetSetupStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSetupStatusLogic) GetSetupStatus() (resp *types.SetupStatusResponse, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return nil, err
	}

	status, err := l.svcCtx.EmailService.PlatformStatus(l.ctx)
	if err != nil {
		return nil, err
	}
	return &types.SetupStatusResponse{
		HasAdmin:           true,
		HasAws:             status.HasSES,
		HasSmtp:            status.HasSMTP,
		PlatformConfigured: status.Configured(),
		MissingSettings:    status.Missing,
	}, nil
}
