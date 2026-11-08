package imports

import (
	"net/http"

	"github.com/outlet-sh/outlet/internal/logic/admin/imports"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ImportListSubscribersHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ImportListSubscribersRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := imports.NewImportListSubscribersLogic(r.Context(), svcCtx)
		resp, err := l.ImportListSubscribers(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
