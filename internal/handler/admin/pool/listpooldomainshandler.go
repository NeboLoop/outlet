package pool

import (
	"net/http"

	"github.com/outlet-sh/outlet/internal/logic/admin/pool"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ListPoolDomainsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := pool.NewListPoolDomainsLogic(r.Context(), svcCtx)
		resp, err := l.ListPoolDomains()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
