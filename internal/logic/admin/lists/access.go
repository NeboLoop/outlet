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

// requireList runs the access check against the brand that owns a list.
func requireList(ctx context.Context, svcCtx *svc.ServiceContext, listID int64, need access.Role) (db.EmailList, error) {
	list, err := svcCtx.DB.GetEmailList(ctx, listID)
	if errors.Is(err, sql.ErrNoRows) {
		return db.EmailList{}, errorx.NewNotFoundError("List not found.")
	}
	if err != nil {
		return db.EmailList{}, err
	}
	if _, err := access.Require(ctx, svcCtx.DB, list.OrgID, need); err != nil {
		return db.EmailList{}, err
	}
	return list, nil
}

// requireCustomField runs the access check for a custom field and makes sure
// the field belongs to the list in the path.
func requireCustomField(ctx context.Context, svcCtx *svc.ServiceContext, listIDParam, fieldID string, need access.Role) (db.CustomField, error) {
	listID, err := strconv.ParseInt(listIDParam, 10, 64)
	if err != nil {
		return db.CustomField{}, errorx.NewBadRequestError("Invalid list ID.")
	}
	if _, err := requireList(ctx, svcCtx, listID, need); err != nil {
		return db.CustomField{}, err
	}
	field, err := svcCtx.DB.GetCustomField(ctx, fieldID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && field.ListID != listID) {
		return db.CustomField{}, errorx.NewNotFoundError("Custom field not found.")
	}
	return field, err
}
