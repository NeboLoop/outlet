package outbound

import (
	"net/http"

	"github.com/outlet-sh/outlet/internal/logic/sdk/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func SdkOutboundOverviewHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := outbound.NewSdkOutboundOverviewLogic(r.Context(), svcCtx)
		resp, err := l.SdkOutboundOverview()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
