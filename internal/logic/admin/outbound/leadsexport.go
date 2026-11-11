package outbound

import (
	"fmt"
	"net/http"
	"time"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/middleware"
	"github.com/outlet-sh/outlet/internal/services/outbound"
	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// leadsRequest is what the leads download reads from its request.
type leadsRequest struct {
	CampaignID     string `form:"campaign_id,optional"`
	Classification string `form:"classification,optional"` // interested: the hot leads
}

// LeadsExportHandler serves GET /api/admin/outbound/leads/export behind the
// admin auth middleware: the brand's outreach leads as CSV, one row per
// person per campaign, streamed as it is read.
func LeadsExportHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req leadsRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, errorx.NewBadRequestError(err.Error()))
			return
		}
		org, err := brand(r.Context(), svcCtx, access.Member)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-leads-%s.csv"`, org.Slug, time.Now().UTC().Format("2006-01-02")))
		w.Header().Set("Cache-Control", "no-store")
		if err := middleware.Unbounded(w); err != nil {
			fmt.Printf("[Leads Export] %s: %v\n", org.ID, err)
		}
		// Once a row has gone the status can't change: a failure cuts the file off.
		if _, _, err := svcCtx.Outbound.WriteLeads(r.Context(), org, w, outbound.LeadQuery{CampaignID: req.CampaignID, Classification: req.Classification}); err != nil && r.Context().Err() == nil {
			fmt.Printf("[Leads Export] %s: %v\n", org.ID, err)
		}
	}
}
