package users

import (
	"context"
	"database/sql"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"
)

type CreateUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateUserLogic {
	return &CreateUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateUserLogic) CreateUser(req *types.CreateUserRequest) (resp *types.CreateUserResponse, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		l.Errorf("Failed to hash password: %v", err)
		return nil, err
	}

	role := req.Role
	if role == "" {
		role = "agent"
	}

	user, err := l.svcCtx.DB.CreateUser(l.ctx, db.CreateUserParams{
		ID:            uuid.New().String(),
		Email:         req.Email,
		PasswordHash:  string(hashedPassword),
		Name:          req.Name,
		Role:          role,
		Status:        "active",
		EmailVerified: 0,
		Phone:         sql.NullString{Valid: false},
	})
	if err != nil {
		l.Errorf("Failed to create user: %v", err)
		return nil, err
	}

	return &types.CreateUserResponse{
		User: types.UserInfo{
			Id:        user.ID,
			Email:     user.Email,
			Name:      user.Name,
			Role:      user.Role,
			Active:    user.Status == "active",
			CreatedAt: utils.FormatNullString(user.CreatedAt),
		},
	}, nil
}
