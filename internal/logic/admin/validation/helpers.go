package validation

import (
	"github.com/outlet-sh/outlet/internal/services/emailval"
)

// suppressibleStatuses are the result statuses a job can be applied to the suppression list with.
var suppressibleStatuses = []string{
	emailval.StatusValid, emailval.StatusInvalid, emailval.StatusCatchAll, emailval.StatusUnknown,
	emailval.StatusSpamtrap, emailval.StatusAbuse, emailval.StatusDoNotMail,
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
