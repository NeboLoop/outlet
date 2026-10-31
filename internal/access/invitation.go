package access

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
)

// Outlet is invite-only. An invitation names an email, a brand and a role,
// works once, and reaches the person only by email: whoever uses it proves
// they read that mailbox.

var (
	errInviteOnly    = errorx.NewBadRequestError("Outlet is invite-only. Ask a brand admin for an invitation.")
	errInviteInvalid = errorx.NewBadRequestError("This invitation isn't valid.")
	errInviteExpired = errorx.NewBadRequestError("This invitation has expired. Ask for a new one.")
	errInviteNotYou  = errorx.NewForbiddenError("This invitation is for another email address. Sign in as that address to accept it.")
	// ErrInvitationUsed is the answer for an invitation that was already used.
	ErrInvitationUsed = errorx.NewBadRequestError("This invitation was already used.")
)

// Invitation returns the invitation a token names, if it can still be used:
// it exists, it hasn't been used, and it hasn't expired.
func Invitation(ctx context.Context, q db.Querier, token string) (db.GetOrgInvitationByTokenRow, error) {
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
		return inv, ErrInvitationUsed
	}
	if inv.Expired != 0 {
		return inv, errInviteExpired
	}
	return inv, nil
}

// Join uses an invitation: it marks it used, once, and adds the person to
// the invitation's brand with its role. The person must have the
// invitation's email address. Run it inside the transaction that needs it.
func Join(ctx context.Context, q db.Querier, inv db.GetOrgInvitationByTokenRow, user db.User) error {
	if !strings.EqualFold(strings.TrimSpace(user.Email), strings.TrimSpace(inv.Email)) {
		return errInviteNotYou
	}
	accepted, err := q.AcceptOrgInvitation(ctx, inv.ID)
	if err != nil {
		return err
	}
	if accepted == 0 {
		return ErrInvitationUsed
	}
	return q.AddUserToOrganization(ctx, db.AddUserToOrganizationParams{
		OrgID:  inv.OrgID,
		UserID: user.ID,
		Role:   sql.NullString{String: inv.Role, Valid: true},
	})
}
