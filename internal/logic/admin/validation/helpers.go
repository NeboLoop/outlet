package validation

import (
	"context"

	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/middleware"
	"github.com/outlet-sh/outlet/internal/services/emailval"
)

// suppressibleStatuses are the result statuses a job can be applied to the suppression list with.
var suppressibleStatuses = []string{
	emailval.StatusValid, emailval.StatusInvalid, emailval.StatusCatchAll, emailval.StatusUnknown,
	emailval.StatusSpamtrap, emailval.StatusAbuse, emailval.StatusDoNotMail,
}

// orgIDFrom returns the brand selected for this admin request.
func orgIDFrom(ctx context.Context) (string, error) {
	orgID, ok := ctx.Value(middleware.OrgIDKey).(string)
	if !ok || orgID == "" {
		return "", errorx.NewBadRequestError("X-Org-Id header is required")
	}
	return orgID, nil
}

// paging normalizes page/limit query values; limit is capped at 200.
func paging(page, limit, defaultLimit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = defaultLimit
	}
	if limit > 200 {
		limit = 200
	}
	return page, limit
}
