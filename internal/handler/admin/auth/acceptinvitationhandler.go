package auth

import (
	"net/http"

	"github.com/outlet-sh/outlet/internal/logic/admin/auth"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// Accept an invitation to a brand as the signed-in person (the invitation's email must be theirs)
func AcceptInvitationHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AcceptInvitationRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := auth.NewAcceptInvitationLogic(r.Context(), svcCtx)
		resp, err := l.AcceptInvitation(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
