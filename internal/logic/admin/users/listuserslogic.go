package users

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListUsersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListUsersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListUsersLogic {
	return &ListUsersLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListUsersLogic) ListUsers() (resp *types.UserListResponse, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return nil, err
	}

	users, err := l.svcCtx.DB.ListUsers(l.ctx, db.ListUsersParams{
		FilterRole:   nil,
		FilterStatus: nil,
		PageSize:     100,
		PageOffset:   0,
	})
	if err != nil {
		l.Errorf("Failed to list users: %v", err)
		return nil, err
	}

	userList := make([]types.UserInfo, 0, len(users))
	for _, u := range users {
		userList = append(userList, types.UserInfo{
			Id:        u.ID,
			Email:     u.Email,
			Name:      u.Name,
			Role:      u.Role,
			Active:    u.Status == "active",
			CreatedAt: utils.FormatNullString(u.CreatedAt),
		})
	}

	return &types.UserListResponse{
		Users: userList,
		Total: len(userList),
	}, nil
}
