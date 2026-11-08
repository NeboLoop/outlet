package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/middleware"
	"github.com/outlet-sh/outlet/internal/svc"
)

// Every /sdk/v1/outbound route runs the admin route's shared function of the
// same name, for the brand of the API key. The brand is read again by ID, so
// a setting changed since the key was cached counts.
func brand(ctx context.Context, svcCtx *svc.ServiceContext) (db.Organization, error) {
	org, ok := ctx.Value(middleware.OrgKey).(db.Organization)
	if !ok {
		return db.Organization{}, errorx.NewUnauthorizedError("API key required")
	}
	return svcCtx.DB.GetOrganizationByID(ctx, org.ID)
}
