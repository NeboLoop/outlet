package organizations

import (
	"net/http"

	"github.com/outlet-sh/outlet/internal/logic/admin/organizations"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// Invite a person to a brand. Outlet is invite-only: the invitation decides the brand and role.
func CreateInvitationHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CreateInvitationRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := organizations.NewCreateInvitationLogic(r.Context(), svcCtx)
		resp, err := l.CreateInvitation(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
