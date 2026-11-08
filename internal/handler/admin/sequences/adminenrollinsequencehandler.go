package sequences

import (
	"net/http"

	"github.com/outlet-sh/outlet/internal/logic/admin/sequences"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// Enroll a contact of the brand in one of its sequences, as the signed-in member (the same enrollment as the SDK route)
func AdminEnrollInSequenceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.EnrollSequenceRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := sequences.NewAdminEnrollInSequenceLogic(r.Context(), svcCtx)
		resp, err := l.AdminEnrollInSequence(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
