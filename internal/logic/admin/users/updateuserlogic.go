package users

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
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
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return nil, err
	}

	// Get current user to preserve existing values
	currentUser, err := l.svcCtx.DB.GetUserByID(l.ctx, req.Id)
	if err != nil {
		l.Errorf("Failed to get user: %v", err)
		return nil, err
	}

	// Use existing values if not provided in request
	name := req.Name
	if name == "" {
		name = currentUser.Name
	}

	// Update user in database
	err = l.svcCtx.DB.UpdateUser(l.ctx, db.UpdateUserParams{
		ID:     req.Id,
		Name:   name,
		Status: currentUser.Status, // Keep existing status
	})
	if err != nil {
		l.Errorf("Failed to update user: %v", err)
		return nil, err
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
