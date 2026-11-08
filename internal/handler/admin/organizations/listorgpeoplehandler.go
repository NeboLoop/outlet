package organizations

import (
	"net/http"

	"github.com/outlet-sh/outlet/internal/logic/admin/organizations"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// A brand's people: its members with their roles, and invitations not yet accepted with when they expire (brand admins)
func ListOrgPeopleHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ListOrgPeopleRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := organizations.NewListOrgPeopleLogic(r.Context(), svcCtx)
		resp, err := l.ListOrgPeople(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
