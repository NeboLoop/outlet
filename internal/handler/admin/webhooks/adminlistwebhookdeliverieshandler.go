package webhooks

import (
	"net/http"

	"github.com/outlet-sh/outlet/internal/logic/admin/webhooks"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// A webhook's deliveries, newest first: status, tries and next retry
func AdminListWebhookDeliveriesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ListWebhookDeliveriesRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := webhooks.NewAdminListWebhookDeliveriesLogic(r.Context(), svcCtx)
		resp, err := l.AdminListWebhookDeliveries(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
