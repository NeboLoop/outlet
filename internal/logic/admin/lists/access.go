package lists

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/svc"
)

// requireCustomField runs the access check against the brand of the list in
// the path and makes sure the field belongs to that list.
func requireCustomField(ctx context.Context, svcCtx *svc.ServiceContext, listIDParam, fieldID string, need access.Role) (db.CustomField, error) {
	listID, err := strconv.ParseInt(listIDParam, 10, 64)
	if err != nil {
		return db.CustomField{}, errorx.NewBadRequestError("Invalid list ID.")
	}
	if _, err := access.RequireOwner(ctx, svcCtx.DB, access.List, listIDParam, need); err != nil {
		return db.CustomField{}, err
	}
	field, err := svcCtx.DB.GetCustomField(ctx, fieldID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && field.ListID != listID) {
		return db.CustomField{}, errorx.NewNotFoundError("Custom field not found.")
	}
	return field, err
}
