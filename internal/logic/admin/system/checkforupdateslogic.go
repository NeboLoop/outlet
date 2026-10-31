package system

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/version"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	// UpdateServerURL is the URL of the update check server
	UpdateServerURL = "https://license.outlet.sh"
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

// UpdateAPIResponse represents the response from the update server
type UpdateAPIResponse struct {
	Version     string `json:"version"`
	ReleaseDate string `json:"release_date"`
	DownloadURL string `json:"download_url"`
	Changelog   string `json:"changelog"`
	Mandatory   bool   `json:"mandatory"`
}

func (l *CheckForUpdatesLogic) CheckForUpdates() (resp *types.UpdateCheckResponse, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return nil, err
	}

	// Build the check URL
	url := fmt.Sprintf("%s/api/v1/updates/check?version=%s&os=%s&arch=%s",
		UpdateServerURL, version.Version, runtime.GOOS, runtime.GOARCH)

	// Create HTTP client with timeout
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequestWithContext(l.ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", fmt.Sprintf("Outlet/%s (%s/%s)", version.Version, runtime.GOOS, runtime.GOARCH))

	resp2, err := client.Do(req)
	if err != nil {
		// If we can't reach the update server, return current version info
		l.Errorf("Failed to check for updates: %v", err)
		return &types.UpdateCheckResponse{
			CurrentVersion:  version.Version,
			LatestVersion:   version.Version,
			UpdateAvailable: false,
		}, nil
	}
	defer resp2.Body.Close()

	// No update available (204 No Content)
	if resp2.StatusCode == http.StatusNoContent {
		return &types.UpdateCheckResponse{
			CurrentVersion:  version.Version,
			LatestVersion:   version.Version,
			UpdateAvailable: false,
		}, nil
	}

	// Error from server
	if resp2.StatusCode != http.StatusOK {
		l.Errorf("Update server returned status %d", resp2.StatusCode)
		return &types.UpdateCheckResponse{
			CurrentVersion:  version.Version,
			LatestVersion:   version.Version,
			UpdateAvailable: false,
		}, nil
	}

	// Parse update response
	var updateResp UpdateAPIResponse
	if err := json.NewDecoder(resp2.Body).Decode(&updateResp); err != nil {
		return nil, fmt.Errorf("failed to parse update response: %w", err)
	}

	return &types.UpdateCheckResponse{
		CurrentVersion:  version.Version,
		LatestVersion:   updateResp.Version,
		UpdateAvailable: true,
		ReleaseDate:     updateResp.ReleaseDate,
		DownloadURL:     updateResp.DownloadURL,
		Changelog:       updateResp.Changelog,
	}, nil
}
