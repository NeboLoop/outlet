package subscribers

import (
	"context"
	"database/sql"
	"errors"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/svc"
)

// requireContact runs the access check against the brand that owns a contact.
func requireContact(ctx context.Context, svcCtx *svc.ServiceContext, contactID string) error {
	contact, err := svcCtx.DB.GetContactByID(ctx, contactID)
	if errors.Is(err, sql.ErrNoRows) {
		return errorx.NewNotFoundError("Subscriber not found.")
	}
	if err != nil {
		return err
	}
	_, err = access.Require(ctx, svcCtx.DB, contact.OrgID.String, access.Member)
	return err
}
