package outbound

import (
	"context"
	"encoding/csv"
	"io"
	"net/http"
	"strings"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/listexport"
)

// The leads export: one row per person per outreach campaign, with their
// segment, where their sequence stands and their last reply. Read and
// written a page at a time, so a large export never sits in memory.

// LeadQuery picks the leads to export: one campaign (empty: all), one reply
// classification (empty: any; interested is the hot leads), at most Limit
// rows (0: all), continuing from a Write's Next.
type LeadQuery struct {
	CampaignID     string
	Classification string
	Limit          int
	After          string
}

// LeadColumns are the export's columns.
var LeadColumns = []string{"email", "name", "company", "title", "segment", "campaign", "status", "stopped_reason", "reply_classification", "replied_at", "started_at"}

const leadPage = 500

// WriteLeads writes the header and the leads as CSV. next continues the
// export when Limit stopped it; empty when it is done.
func (s *Service) WriteLeads(ctx context.Context, org db.Organization, w io.Writer, q LeadQuery) (rows int, next string, err error) {
	cw := csv.NewWriter(w)
	if err := cw.Write(LeadColumns); err != nil {
		return 0, "", err
	}
	after := q.After
	for {
		if err := ctx.Err(); err != nil {
			return rows, "", err
		}
		take := leadPage
		if q.Limit > 0 && q.Limit-rows < take {
			take = q.Limit - rows
		}
		page, err := s.db.ListOutreachLeads(ctx, db.ListOutreachLeadsParams{
			OrgID: nullStr(org.ID), CampaignID: q.CampaignID, Classification: strings.TrimSpace(q.Classification), After: after, LimitCount: int64(take),
		})
		if err != nil {
			return rows, "", err
		}
		for _, r := range page {
			fields := s.seq.GetCustomFieldsForContact(ctx, r.ContactID, r.ListID)
			if err := cw.Write(listexport.SafeRow([]string{
				r.Email, r.Name, fields["company"], fields["title"], r.SegmentName, r.CampaignName,
				leadStatus(r), r.StoppedReason, r.ReplyClassification, r.RepliedAt, r.StartedAt.String,
			})); err != nil {
				return rows, "", err
			}
		}
		rows += len(page)
		cw.Flush()
		if err := cw.Error(); err != nil {
			return rows, "", err
		}
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		if len(page) < take {
			return rows, "", nil
		}
		after = page[len(page)-1].CursorKey
		if q.Limit > 0 && rows >= q.Limit {
			return rows, after, nil
		}
	}
}

// leadStatus is where a person's sequence stands, in a word.
func leadStatus(r db.ListOutreachLeadsRow) string {
	switch {
	case r.ReplyClassification != "":
		return "replied"
	case r.StoppedReason != "":
		return "stopped"
	case r.IsActive.Int64 == 1 && !r.CompletedAt.Valid:
		return "in_progress"
	}
	return "finished_no_reply"
}
