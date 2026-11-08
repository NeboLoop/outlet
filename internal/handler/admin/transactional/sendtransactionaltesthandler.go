package transactional

import (
	"net/http"

	"github.com/outlet-sh/outlet/internal/logic/admin/transactional"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// Send a test transactional email as the signed-in member, through the same send as the SDK route
func SendTransactionalTestHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SendEmailRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := transactional.NewSendTransactionalTestLogic(r.Context(), svcCtx)
		resp, err := l.SendTransactionalTest(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
