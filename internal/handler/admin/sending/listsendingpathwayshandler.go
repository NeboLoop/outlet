package sending

import (
	"net/http"

	"github.com/outlet-sh/outlet/internal/logic/admin/sending"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ListSendingPathwaysHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := sending.NewListSendingPathwaysLogic(r.Context(), svcCtx)
		resp, err := l.ListSendingPathways()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
