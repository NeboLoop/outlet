package version

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withReleases(t *testing.T, url, current string) {
	t.Helper()
	oldURL, oldVersion := ReleasesAPI, Version
	ReleasesAPI, Version = url, current
	t.Cleanup(func() { ReleasesAPI, Version = oldURL, oldVersion })
}

// A check that can't reach GitHub fails; it never reports "up to date".
func TestCheckForUpdate_FailureIsAnError(t *testing.T) {
	withReleases(t, "http://update-server.invalid/releases/latest", "1.0.0")
	u, err := CheckForUpdate(context.Background(), false)
	assert.Error(t, err)
	assert.Nil(t, u)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusForbidden)
	}))
	t.Cleanup(srv.Close)
	withReleases(t, srv.URL, "1.0.0")
	_, err = CheckForUpdate(context.Background(), false)
	assert.ErrorContains(t, err, "403")
}

func TestCheckForUpdate_NewerAndLatest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.0","body":"Notes","published_at":"2026-11-01T00:00:00Z","html_url":"https://github.com/outlet-sh/outlet/releases/tag/v1.2.0","assets":[]}`))
	}))
	t.Cleanup(srv.Close)

	withReleases(t, srv.URL, "1.0.0")
	u, err := CheckForUpdate(context.Background(), false)
	require.NoError(t, err)
	require.NotNil(t, u)
	assert.Equal(t, "1.2.0", u.Version)
	assert.Equal(t, "https://github.com/outlet-sh/outlet/releases/tag/v1.2.0", u.ReleaseURL)

	withReleases(t, srv.URL, "v1.2.0")
	u, err = CheckForUpdate(context.Background(), false)
	require.NoError(t, err)
	assert.Nil(t, u, "up to date")
}

func TestReleasesAPIIsTheProjectsRepository(t *testing.T) {
	assert.Equal(t, "https://api.github.com/repos/outlet-sh/outlet/releases/latest", ReleasesAPI)
}
