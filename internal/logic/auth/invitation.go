package auth

import (
	"context"
	"database/sql"
	"errors"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
)

var (
	errInviteOnly      = errorx.NewBadRequestError("Outlet is invite-only. Ask a brand admin for an invitation.")
	errInviteInvalid   = errorx.NewBadRequestError("This invitation isn't valid.")
	errInviteUsed      = errorx.NewBadRequestError("This invitation was already used.")
	errInviteExpired   = errorx.NewBadRequestError("This invitation has expired. Ask for a new one.")
	errAccountExists   = errorx.NewBadRequestError("An account with this email already exists. Sign in instead.")
	errPasswordTooWeak = errorx.NewBadRequestError("Use a password of at least 8 characters.")
)

// usableInvitation returns the invitation a token names, if it can still be
// used: it exists, it hasn't been used, and it hasn't expired.
func usableInvitation(ctx context.Context, q db.Querier, token string) (db.GetOrgInvitationByTokenRow, error) {
	if token == "" {
		return db.GetOrgInvitationByTokenRow{}, errInviteOnly
	}
	inv, err := q.GetOrgInvitationByToken(ctx, token)
	if errors.Is(err, sql.ErrNoRows) {
		return inv, errInviteInvalid
	}
	if err != nil {
		return inv, err
	}
	if inv.AcceptedAt.Valid {
		return inv, errInviteUsed
	}
	if inv.Expired != 0 {
		return inv, errInviteExpired
	}
	return inv, nil
}
