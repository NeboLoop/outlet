package designs

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
)

// BrandDesign reads the design a campaign or transactional email names,
// which must be the brand's own. An empty id means no design.
func BrandDesign(ctx context.Context, store *db.Store, orgID, id string) (*db.EmailDesign, error) {
	if id == "" {
		return nil, nil
	}
	designID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, errorx.NewBadRequestError("Choose a design from this brand's templates.")
	}
	design, err := store.GetEmailDesign(ctx, db.GetEmailDesignParams{ID: designID, OrgID: orgID})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errorx.NewBadRequestError("Choose a design from this brand's templates.")
	}
	if err != nil {
		return nil, err
	}
	return &design, nil
}

// DesignColumn is the design_id value stored for a design.
func DesignColumn(d *db.EmailDesign) sql.NullInt64 {
	if d == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: d.ID, Valid: true}
}
