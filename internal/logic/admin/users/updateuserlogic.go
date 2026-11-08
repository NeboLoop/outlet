package users

import (
	"context"
	"database/sql"
	"errors"
	"net/mail"
	"strings"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"
)

type UpdateUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserLogic {
	return &UpdateUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateUserLogic) UpdateUser(req *types.UpdateUserRequest) (resp *types.UserInfo, err error) {
	// Only a super admin changes people, and nobody holds a role above it, so
	// nobody can raise their own role.
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return nil, err
	}

	// Get current user to preserve existing values
	currentUser, err := l.svcCtx.DB.GetUserByID(l.ctx, req.Id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errorx.NewNotFoundError("User not found.")
	}
	if err != nil {
		l.Errorf("Failed to get user: %v", err)
		return nil, err
	}

	// Use existing values if not provided in request
	name := req.Name
	if name == "" {
		name = currentUser.Name
	}
	role := currentUser.Role
	if req.Role != "" {
		if !access.PlatformRole(req.Role) {
			return nil, errorx.NewBadRequestError("A role is super_admin or " + access.PlatformUser + ".")
		}
		role = req.Role
	}
	email := currentUser.Email
	if e := strings.TrimSpace(req.Email); e != "" && !strings.EqualFold(e, currentUser.Email) {
		if addr, err := mail.ParseAddress(e); err != nil || addr.Address != e {
			return nil, errorx.NewBadRequestError("Enter a valid email address.")
		}
		if other, err := l.svcCtx.DB.GetUserByEmail(l.ctx, e); err == nil && other.ID != currentUser.ID {
			return nil, errorx.NewBadRequestError("Another user has this email address.")
		}
		email = e
	}

	// Update user in database
	n, err := l.svcCtx.DB.UpdateUser(l.ctx, db.UpdateUserParams{
		ID:     req.Id,
		Name:   name,
		Email:  email,
		Role:   role,
		Status: currentUser.Status, // Keep existing status
	})
	if err != nil {
		l.Errorf("Failed to update user: %v", err)
		return nil, err
	}
	if n == 0 {
		return nil, errorx.NewBadRequestError("This is the last super admin. Make someone else a super admin first.")
	}

	// Update password if provided
	if req.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			l.Errorf("Failed to hash password: %v", err)
			return nil, err
		}
		err = l.svcCtx.DB.UpdateUserPassword(l.ctx, db.UpdateUserPasswordParams{
			ID:           req.Id,
			PasswordHash: string(hashedPassword),
		})
		if err != nil {
			l.Errorf("Failed to update password: %v", err)
			return nil, err
		}
	}

	// Get updated user
	user, err := l.svcCtx.DB.GetUserByID(l.ctx, req.Id)
	if err != nil {
		l.Errorf("Failed to get updated user: %v", err)
		return nil, err
	}

	return &types.UserInfo{
		Id:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      user.Role,
		Active:    user.Status == "active",
		CreatedAt: utils.FormatNullString(user.CreatedAt),
	}, nil
}
