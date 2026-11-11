package outbound

import (
	"net/http"

	"github.com/outlet-sh/outlet/internal/logic/admin/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func UpdateOutboundMaterialHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UpdateOutboundMaterialRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := outbound.NewUpdateOutboundMaterialLogic(r.Context(), svcCtx)
		resp, err := l.UpdateOutboundMaterial(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
