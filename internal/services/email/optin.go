package email

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
)

// An outreach list (email_lists.kind 'outreach') holds prospects: people
// found for outreach, who never signed up. Only outreach sequences, sent from
// Outbound through the brand's mailboxes, write to them. Opt-in mail (a
// newsletter campaign, any sequence other than an outreach sequence, a
// marketing email sent through the API) never goes to one.

// CheckOptInLists refuses an outreach list among the lists opt-in mail goes
// to. A list that no longer exists sends nothing and passes.
func CheckOptInLists(ctx context.Context, q db.Querier, listIDs []int64) error {
	for _, id := range listIDs {
		list, err := q.GetEmailList(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		if list.Kind == "outreach" {
			return errorx.NewBadRequestError(fmt.Sprintf("%q is an outreach list: its people are prospects, who never signed up. Newsletter campaigns and autoresponders go only to newsletter lists; prospects are emailed from Outbound.", list.Name))
		}
	}
	return nil
}
