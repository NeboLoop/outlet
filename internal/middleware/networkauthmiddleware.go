package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/services/signalnet"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// NetworkInstanceIDKey holds the authenticated Signal Network instance ID.
const NetworkInstanceIDKey contextKey = "network_instance_id"

// NetworkAuthMiddleware authenticates Signal Network hub requests by X-Instance-Key.
// With a nil hub (SignalNet.Hub disabled) every request gets a 404.
type NetworkAuthMiddleware struct {
	hub *signalnet.Hub
}

func NewNetworkAuthMiddleware(hub *signalnet.Hub) *NetworkAuthMiddleware {
	return &NetworkAuthMiddleware{hub: hub}
}

func (m *NetworkAuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if m.hub == nil {
			httpx.ErrorCtx(r.Context(), w, errorx.NewNotFoundError("not found"))
			return
		}
		inst, err := m.hub.Authenticate(r.Context(), r.Header.Get("X-Instance-Key"))
		if err != nil {
			if !errors.Is(err, signalnet.ErrUnauthorized) {
				logx.WithContext(r.Context()).Errorf("signalnet: instance auth failed: %v", err)
				httpx.ErrorCtx(r.Context(), w, errorx.NewInternalError("authentication failed"))
				return
			}
			httpx.ErrorCtx(r.Context(), w, errorx.NewUnauthorizedError("invalid instance key"))
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), NetworkInstanceIDKey, inst.ID)))
	}
}
