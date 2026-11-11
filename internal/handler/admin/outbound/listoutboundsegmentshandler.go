package outbound

import (
	"net/http"

	"github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ListOutboundSegmentsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := outbound.NewListOutboundSegmentsLogic(r.Context(), svcCtx)
		resp, err := l.ListOutboundSegments()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
