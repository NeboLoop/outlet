package public

import (
	"context"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPublicSetupStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Check if initial setup is required
func NewGetPublicSetupStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPublicSetupStatusLogic {
	return &GetPublicSetupStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetPublicSetupStatusLogic) GetPublicSetupStatus() (resp *types.SetupStatusResponse, err error) {
	// Check if any admin user exists
	adminCount, err := l.svcCtx.DB.CountUsers(l.ctx, db.CountUsersParams{
		FilterRole:   "super_admin",
		FilterStatus: nil,
	})
	if err != nil {
		l.Error("failed to count admin users:", err)
		return nil, err
	}
	hasAdmin := adminCount > 0

	status, err := l.svcCtx.EmailService.PlatformStatus(l.ctx)
	if err != nil {
		l.Error("failed to read email settings:", err)
		return nil, err
	}

	return &types.SetupStatusResponse{
		SetupRequired:      !hasAdmin,
		HasAdmin:           hasAdmin,
		HasAws:             status.HasSES,
		HasSmtp:            status.HasSMTP,
		PlatformConfigured: hasAdmin && status.Configured(),
		MissingSettings:    status.Missing,
	}, nil
}
