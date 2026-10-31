package auth

import (
	"github.com/outlet-sh/outlet/internal/errorx"
)

var (
	errAccountExists   = errorx.NewBadRequestError("An account with this email already exists. Sign in, then open the invitation link again to accept it.")
	errPasswordTooWeak = errorx.NewBadRequestError("Use a password of at least 8 characters.")
)
