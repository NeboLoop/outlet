package webhooks

import (
	"net/http"

	"github.com/outlet-sh/outlet/internal/logic/admin/webhooks"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// Send a delivery again now, signed the same way
func AdminResendWebhookDeliveryHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ResendWebhookDeliveryRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := webhooks.NewAdminResendWebhookDeliveryLogic(r.Context(), svcCtx)
		resp, err := l.AdminResendWebhookDelivery(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
