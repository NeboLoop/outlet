package auth

import (
	"fmt"
	"time"

	"github.com/outlet-sh/outlet/internal/config"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/utils"
)

// issueSession is the one way a signed-in person gets their tokens: an
// access token and a refresh token, both signed with the configured secrets.
// Password sign-in, emailed sign-in and token refresh all end here.
func issueSession(c config.Config, user db.User) (*types.LoginResponse, error) {
	accessExpire := time.Duration(c.Auth.AccessExpire) * time.Second
	accessToken, err := utils.GenerateToken(user.ID, user.Email, user.Role, c.Auth.AccessSecret, accessExpire)
	if err != nil {
		return nil, fmt.Errorf("failed to generate authentication token: %w", err)
	}
	refreshExpire := time.Duration(c.Auth.RefreshExpire) * time.Second
	refreshToken, err := utils.GenerateToken(user.ID, user.Email, user.Role, c.Auth.RefreshSecret, refreshExpire)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	return &types.LoginResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(accessExpire).Format(time.RFC3339),
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
