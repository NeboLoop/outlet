package outbound

import (
	"net/http"

	"github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func DeleteOutboundDoNotContactListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.OutboundDNCDeleteRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := outbound.NewDeleteOutboundDoNotContactListLogic(r.Context(), svcCtx)
		resp, err := l.DeleteOutboundDoNotContactList(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
