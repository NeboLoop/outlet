package access

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
)

// Record finds the brand a record belongs to from the record's ID. The admin
// REST API and the MCP tools use the same ones.
type Record func(ctx context.Context, q db.Querier, id string) (string, error)

// The records an action can name by ID.
var (
	List Record = func(ctx context.Context, q db.Querier, id string) (string, error) {
		n, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return "", sql.ErrNoRows
		}
		list, err := q.GetEmailList(ctx, n)
		return list.OrgID, err
	}
	Sequence Record = func(ctx context.Context, q db.Querier, id string) (string, error) {
		return q.GetSequenceOrgID(ctx, id)
	}
	Template Record = func(ctx context.Context, q db.Querier, id string) (string, error) {
		return q.GetTemplateOrgID(ctx, id)
	}
	EntryRule Record = func(ctx context.Context, q db.Querier, id string) (string, error) {
		return q.GetEntryRuleOrgID(ctx, id)
	}
	QueuedEmail Record = func(ctx context.Context, q db.Querier, id string) (string, error) {
		return q.GetQueuedEmailOrgID(ctx, id)
	}
	Contact Record = func(ctx context.Context, q db.Querier, id string) (string, error) {
		c, err := q.GetContactByID(ctx, id)
		return c.OrgID.String, err
	}
)

// ErrNotFound is the answer for a record that is missing or belongs to
// another brand: the two look the same to the caller.
var ErrNotFound = errorx.NewNotFoundError("Not found.")

// brandOf finds a record's brand. A record with no brand is not found.
func brandOf(ctx context.Context, q db.Querier, rec Record, id string) (string, error) {
	brandID, err := rec(ctx, q, id)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && brandID == "") {
		return "", ErrNotFound
	}
	return brandID, err
}

// RequireOwner runs Require against the brand that owns a record, and
// returns that brand's ID. The admin REST API uses it for actions that name
// a record and no brand.
func RequireOwner(ctx context.Context, q db.Querier, rec Record, id string, need Role) (string, error) {
	brandID, err := brandOf(ctx, q, rec, id)
	if err != nil {
		return "", err
	}
	if _, err := Require(ctx, q, brandID, need); err != nil {
		return "", err
	}
	return brandID, nil
}

// SameBrand refuses a reference to a record of another brand. An action
// that already passed Check for brandID uses it on every record ID the
// caller hands it.
func SameBrand(ctx context.Context, q db.Querier, rec Record, id, brandID string) error {
	owner, err := brandOf(ctx, q, rec, id)
	if err != nil {
		return err
	}
	if owner != brandID {
		return ErrNotFound
	}
	return nil
}

// BrandLists checks that every ID names a list of this brand and returns
// the IDs as numbers. Campaigns use it for their target and excluded lists.
func BrandLists(ctx context.Context, q db.Querier, brandID string, ids []string) ([]int64, error) {
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if err := SameBrand(ctx, q, List, id, brandID); err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, errorx.NewNotFoundError(fmt.Sprintf("List %s not found.", id))
			}
			return nil, err
		}
		n, _ := strconv.ParseInt(id, 10, 64)
		out = append(out, n)
	}
	return out, nil
}
