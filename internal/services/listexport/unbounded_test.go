package listexport

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/outlet-sh/outlet/internal/middleware"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A download outlives the server's write timeout: Unbounded lifts it, through
// the server's own response-writer wrappers.
func TestDownloadOutlivesWriteTimeout(t *testing.T) {
	slow := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, Unbounded(w))
		cw := &countingWriter{ResponseWriter: w}
		for i := 0; i < 4; i++ {
			io.WriteString(cw, "row\n")
			cw.Flush()
			time.Sleep(80 * time.Millisecond)
		}
	})
	srv := httptest.NewUnstartedServer(middleware.RequestLog(middleware.Gzip(slow).ServeHTTP))
	srv.Config.WriteTimeout = 100 * time.Millisecond
	srv.Start()
	t.Cleanup(srv.Close)

	res, err := http.Get(srv.URL)
	require.NoError(t, err)
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Equal(t, strings.Repeat("row\n", 4), string(body))
}
