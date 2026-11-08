package version

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// ReleasesAPI is the latest Outlet release on GitHub, where the project lives.
var ReleasesAPI = "https://api.github.com/repos/outlet-sh/outlet/releases/latest"

// Update is a release newer than this build.
type Update struct {
	Version     string
	ReleaseDate string
	ReleaseURL  string // the release's page
	DownloadURL string // this platform's binary; empty when the release has none
	Checksum    string // SHA256 of the binary, from checksums.txt
	Changelog   string
}

type githubRelease struct {
	TagName     string `json:"tag_name"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
	HTMLURL     string `json:"html_url"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// CheckForUpdate asks GitHub for the latest release. It returns nil when this
// build is the latest (force returns the latest release anyway), and an error
// when the check itself failed.
func CheckForUpdate(ctx context.Context, force bool) (*Update, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ReleasesAPI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", fmt.Sprintf("Outlet/%s (%s/%s)", Version, runtime.GOOS, runtime.GOARCH))

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("couldn't reach GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // no releases yet
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("GitHub answered %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("couldn't read GitHub's answer: %w", err)
	}

	// Compare versions (simple string comparison - assumes semver)
	latest := strings.TrimPrefix(release.TagName, "v")
	if !force && latest <= strings.TrimPrefix(Version, "v") {
		return nil, nil
	}

	u := &Update{
		Version:     latest,
		ReleaseDate: release.PublishedAt,
		ReleaseURL:  release.HTMLURL,
		Changelog:   release.Body,
	}
	assetName := fmt.Sprintf("outlet_%s_%s", runtime.GOOS, runtime.GOARCH)
	var checksumURL string
	for _, a := range release.Assets {
		if strings.Contains(a.Name, assetName) && !strings.HasSuffix(a.Name, ".txt") {
			u.DownloadURL = a.BrowserDownloadURL
		}
		if a.Name == "checksums.txt" {
			checksumURL = a.BrowserDownloadURL
		}
	}
	if checksumURL != "" {
		u.Checksum = fetchChecksum(ctx, client, checksumURL, assetName)
	}
	return u, nil
}

// fetchChecksum reads the asset's line in checksums.txt: "checksum  filename".
func fetchChecksum(ctx context.Context, client *http.Client, checksumURL, assetName string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumURL, nil)
	if err != nil {
		return ""
	}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(body), "\n") {
		if strings.Contains(line, assetName) {
			if parts := strings.Fields(line); len(parts) >= 1 {
				return parts[0]
			}
		}
	}
	return ""
}
