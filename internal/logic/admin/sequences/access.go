package sequences

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/svc"
)

var errNotFound = errorx.NewNotFoundError("Not found.")

// requireBrandOf runs the access check against the brand that owns a record
// and returns that brand's ID. lookup finds the record's brand.
func requireBrandOf(ctx context.Context, svcCtx *svc.ServiceContext, lookup func(context.Context, string) (string, error), id string, need access.Role) (string, error) {
	orgID, err := lookup(ctx, id)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && orgID == "") {
		return "", errNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err := access.Require(ctx, svcCtx.DB, orgID, need); err != nil {
		return "", err
	}
	return orgID, nil
}

// requireSameBrand refuses a reference to a record of another brand.
func requireSameBrand(ctx context.Context, lookup func(context.Context, string) (string, error), id, orgID string) error {
	other, err := lookup(ctx, id)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && other != orgID) {
		return errNotFound
	}
	return err
}

// listOrgID finds the brand of a list from its ID.
func listOrgID(svcCtx *svc.ServiceContext) func(context.Context, string) (string, error) {
	return func(ctx context.Context, id string) (string, error) {
		n, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return "", sql.ErrNoRows
		}
		list, err := svcCtx.DB.GetEmailList(ctx, n)
		return list.OrgID, err
	}
}
