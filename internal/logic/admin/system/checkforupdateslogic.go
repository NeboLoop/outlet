package system

import (
	"context"
	"net/http"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/version"

	"github.com/zeromicro/go-zero/core/logx"
)

type CheckForUpdatesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCheckForUpdatesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckForUpdatesLogic {
	return &CheckForUpdatesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// CheckForUpdates reports whether a newer Outlet release exists. A check that
// fails says so: it never passes for "up to date".
func (l *CheckForUpdatesLogic) CheckForUpdates() (resp *types.UpdateCheckResponse, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return nil, err
	}

	update, err := version.CheckForUpdate(l.ctx, false)
	if err != nil {
		l.Errorf("Failed to check for updates: %v", err)
		return nil, errorx.NewCodeError(http.StatusBadGateway, "Couldn't check for updates: "+err.Error())
	}
	if update == nil {
		return &types.UpdateCheckResponse{
			CurrentVersion:  version.Version,
			LatestVersion:   version.Version,
			UpdateAvailable: false,
		}, nil
	}
	return &types.UpdateCheckResponse{
		CurrentVersion:  version.Version,
		LatestVersion:   update.Version,
		UpdateAvailable: true,
		ReleaseDate:     update.ReleaseDate,
		DownloadURL:     update.ReleaseURL,
		Changelog:       update.Changelog,
	}, nil
}
