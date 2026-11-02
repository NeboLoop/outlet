package pool_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/outlet-sh/outlet/internal/config"
	"github.com/outlet-sh/outlet/internal/errorx"
	sdkpool "github.com/outlet-sh/outlet/internal/handler/sdk/pool"
	"github.com/outlet-sh/outlet/internal/middleware"
	"github.com/outlet-sh/outlet/internal/services/mailpool"
	"github.com/outlet-sh/outlet/internal/svc"
)

// With Pool.Enabled off (the shipped default) every pool route answers 503.
func TestPoolRoutesAnswer503WhenOff(t *testing.T) {
	errorx.SetupErrorHandler()
	svcCtx := &svc.ServiceContext{MailPool: mailpool.New(config.PoolConfig{}, nil, nil, nil)}

	for _, tc := range []struct {
		name    string
		handler http.HandlerFunc
		method  string
		target  string
		body    string
	}{
		{"send", sdkpool.PoolSendHandler(svcCtx), http.MethodPost, "/sdk/v1/pool/send",
			`{"tenant":"a","to":"x@example.com","subject":"s","text":"t","list_unsubscribe_url":"https://o.test/u","idempotency_key":"k"}`},
		{"capacity", sdkpool.PoolCapacityHandler(svcCtx), http.MethodGet, "/sdk/v1/pool/capacity?tenant=a", ""},
		{"assign", sdkpool.PoolAssignHandler(svcCtx), http.MethodPost, "/sdk/v1/pool/assign", `{"tenant":"a","count":1}`},
		{"release", sdkpool.PoolReleaseHandler(svcCtx), http.MethodPost, "/sdk/v1/pool/release", `{"tenant":"a"}`},
		{"verify", sdkpool.PoolVerifyHandler(svcCtx), http.MethodPost, "/sdk/v1/pool/verify", `{"emails":["x@example.com"]}`},
		{"bounces", sdkpool.PoolReportHandler(svcCtx), http.MethodPost, "/sdk/v1/pool/bounces", `{"raw":"x"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.target, strings.NewReader(tc.body))
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			req = req.WithContext(context.WithValue(req.Context(), middleware.OrgIDKey, "org-hub"))
			rec := httptest.NewRecorder()
			tc.handler(rec, req)
			assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
			var body errorx.ErrorResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, "pool is off", body.Message)
		})
	}
}
