package campaigns

import (
	"net/http"

	"github.com/outlet-sh/outlet/internal/logic/admin/campaigns"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// Send a test of a campaign to one address, built as a real send is (design, merge tags, unsubscribe link)
func SendCampaignTestHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SendCampaignTestRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := campaigns.NewSendCampaignTestLogic(r.Context(), svcCtx)
		resp, err := l.SendCampaignTest(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
