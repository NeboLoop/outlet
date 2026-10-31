package access

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"
)

var (
	errBadSignIn = errorx.NewUnauthorizedError("Wrong email or password.")
	errLocked    = errorx.NewUnauthorizedError("This account is locked after too many wrong passwords. Try again in 30 minutes.")
)

// SignIn checks an email and password. It is the one sign-in check: the
// admin sign-in and the MCP OAuth sign-in form both use it. Five wrong
// passwords in a row lock the account for 30 minutes, and a locked account
// is refused before its password is checked, so guessing while it is locked
// learns nothing. A person who is not active is refused.
func SignIn(ctx context.Context, q db.Querier, email, password string) (db.User, error) {
	if email == "" || password == "" {
		return db.User{}, errBadSignIn
	}
	u, err := q.GetUserByEmail(ctx, email)
	if errors.Is(err, sql.ErrNoRows) {
		return db.User{}, errBadSignIn
	}
	if err != nil {
		return db.User{}, err
	}
	if lockedUntil, err := time.Parse(time.DateTime, u.LockedUntil.String); err == nil && time.Now().UTC().Before(lockedUntil) {
		return db.User{}, errLocked
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		if err := q.IncrementFailedLogins(ctx, u.ID); err != nil {
			logx.WithContext(ctx).Errorf("Sign-in: could not count a failed attempt for %s: %v", u.ID, err)
		}
		return db.User{}, errBadSignIn
	}
	user, err := Active(ctx, q, u.ID)
	if err != nil {
		return db.User{}, err
	}
	// Records the sign-in and clears the failed attempts.
	if err := q.UpdateLastLogin(ctx, u.ID); err != nil {
		logx.WithContext(ctx).Errorf("Sign-in: could not record the sign-in for %s: %v", u.ID, err)
	}
	return user, nil
}
