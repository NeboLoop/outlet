// Package access decides who may do what in Outlet. Every brand action, from
// the admin REST API and from MCP alike, goes through Check.
package access

import (
	"context"
	"database/sql"
	"errors"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
)

// Role is what a person may do. A higher role includes everything below it.
type Role string

const (
	// Member reads a brand and runs its everyday mail: contacts, lists,
	// sequences, templates, campaigns and sends.
	Member Role = "member"
	// Admin also runs the brand itself: its settings, sending setup, keys,
	// webhooks, people, and the actions that delete or clear its data.
	Admin Role = "admin"
	// SuperAdmin is the platform role (users.role). It reaches every brand,
	// and only it manages users, platform settings, backups and new brands.
	SuperAdmin Role = "super_admin"
)

// BrandRole reports whether s names a role a brand can grant.
func BrandRole(s string) bool {
	return s == string(Member) || s == string(Admin)
}

func rank(r Role) int {
	switch r {
	case Member:
		return 1
	case Admin:
		return 2
	case SuperAdmin:
		return 3
	}
	return 0
}

// Caller is a person who passed Check.
type Caller struct {
	User db.User
	// Role is what the person holds where they asked: their brand role, or
	// SuperAdmin for a super admin.
	Role Role
}

var (
	errSignIn      = errorx.NewUnauthorizedError("Sign in first.")
	errNoAccount   = errorx.NewUnauthorizedError("This account no longer exists.")
	errInactive    = errorx.NewUnauthorizedError("This account is not active.")
	errNoBrand     = errorx.NewBadRequestError("Choose a brand first.")
	errNotInBrand  = errorx.NewForbiddenError("You don't have access to this brand.")
	errNeedsAdmin  = errorx.NewForbiddenError("This needs the brand's admin role.")
	errSuperAdmin  = errorx.NewForbiddenError("Only a super admin can do this.")
	errUnknownRole = errorx.NewBadRequestError("A brand role is admin or member.")
)

// Active loads a person and refuses one who is gone or not active. Sign-in,
// token refresh and every call use it.
func Active(ctx context.Context, q db.Querier, userID string) (db.User, error) {
	if userID == "" {
		return db.User{}, errSignIn
	}
	u, err := q.GetUserByID(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return db.User{}, errNoAccount
	}
	if err != nil {
		return db.User{}, err
	}
	if u.Status != "active" {
		return db.User{}, errInactive
	}
	return db.User{
		ID: u.ID, Email: u.Email, PasswordHash: u.PasswordHash, Name: u.Name, Role: u.Role,
		Status: u.Status, EmailVerified: u.EmailVerified, Phone: u.Phone, AvatarUrl: u.AvatarUrl,
		LastLoginAt: u.LastLoginAt, PasswordChangedAt: u.PasswordChangedAt,
		FailedLoginAttempts: u.FailedLoginAttempts, LockedUntil: u.LockedUntil,
		CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
	}, nil
}

// Check is the one access check: is this person active, and do they hold
// the needed role in this brand? A super admin reaches every brand. An empty
// brandID, or need SuperAdmin, asks for the platform role.
func Check(ctx context.Context, q db.Querier, userID, brandID string, need Role) (Caller, error) {
	if rank(need) == 0 {
		return Caller{}, errUnknownRole
	}
	user, err := Active(ctx, q, userID)
	if err != nil {
		return Caller{}, err
	}
	if Role(user.Role) == SuperAdmin {
		return Caller{User: user, Role: SuperAdmin}, nil
	}
	if need == SuperAdmin || brandID == "" {
		return Caller{}, errSuperAdmin
	}
	role, err := q.GetOrgUserRole(ctx, db.GetOrgUserRoleParams{OrgID: brandID, UserID: user.ID})
	if errors.Is(err, sql.ErrNoRows) {
		return Caller{}, errNotInBrand
	}
	if err != nil {
		return Caller{}, err
	}
	if rank(Role(role)) < rank(need) {
		if need == Admin {
			return Caller{}, errNeedsAdmin
		}
		return Caller{}, errNotInBrand
	}
	return Caller{User: user, Role: Role(role)}, nil
}

// Brand is a brand a person can reach, with their role in it.
type Brand struct {
	db.Organization
	Role Role
}

// Brands lists the brands a person can reach: every brand for a super admin,
// otherwise the brands they belong to.
func Brands(ctx context.Context, q db.Querier, userID string) ([]Brand, error) {
	user, err := Active(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	if Role(user.Role) == SuperAdmin {
		orgs, err := q.ListOrganizations(ctx)
		if err != nil {
			return nil, err
		}
		brands := make([]Brand, 0, len(orgs))
		for _, o := range orgs {
			brands = append(brands, Brand{Organization: o, Role: SuperAdmin})
		}
		return brands, nil
	}
	rows, err := q.GetUserOrganizations(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	brands := make([]Brand, 0, len(rows))
	for _, r := range rows {
		role := Member
		if r.UserRole.Valid && r.UserRole.String != "" {
			role = Role(r.UserRole.String)
		}
		brands = append(brands, Brand{Organization: db.Organization{
			ID: r.ID, Name: r.Name, Slug: r.Slug, ApiKey: r.ApiKey, ApiKeyCreatedAt: r.ApiKeyCreatedAt,
			FromName: r.FromName, FromEmail: r.FromEmail, ReplyTo: r.ReplyTo, MaxContacts: r.MaxContacts,
			Settings: r.Settings, AppUrl: r.AppUrl, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		}, Role: role})
	}
	return brands, nil
}

type userKey struct{}
type brandKey struct{}

// WithUser records the signed-in person on a request. The admin REST auth
// middleware calls it after the token and the account check out.
func WithUser(ctx context.Context, user db.User) context.Context {
	return context.WithValue(ctx, userKey{}, user)
}

// User returns the signed-in person, if any.
func User(ctx context.Context) (db.User, bool) {
	u, ok := ctx.Value(userKey{}).(db.User)
	return u, ok
}

// WithBrand records the brand a request names (the X-Org-Id header).
func WithBrand(ctx context.Context, brandID string) context.Context {
	return context.WithValue(ctx, brandKey{}, brandID)
}

// Require runs Check for the person signed in on this request.
func Require(ctx context.Context, q db.Querier, brandID string, need Role) (Caller, error) {
	u, ok := User(ctx)
	if !ok {
		return Caller{}, errSignIn
	}
	return Check(ctx, q, u.ID, brandID, need)
}

// RequireBrand runs Check against the brand the request names and returns
// that brand's ID.
func RequireBrand(ctx context.Context, q db.Querier, need Role) (string, error) {
	if _, ok := User(ctx); !ok {
		return "", errSignIn
	}
	brandID, _ := ctx.Value(brandKey{}).(string)
	if brandID == "" {
		return "", errNoBrand
	}
	if _, err := Require(ctx, q, brandID, need); err != nil {
		return "", err
	}
	return brandID, nil
}
