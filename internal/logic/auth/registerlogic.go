package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"
)

type RegisterLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// invitedPlatformRole is the platform role (users.role) of a person who joins
// by invitation. Only super_admin means anything there; what a person may do
// in a brand is their role in org_users, which the invitation sets.
const invitedPlatformRole = "agent"

// Register creates an account from an invitation. Outlet is invite-only: the
// invitation decides the email, the brand and the role, and works once.
func (l *RegisterLogic) Register(req *types.RegisterRequest) (resp *types.RegisterResponse, err error) {
	inv, err := usableInvitation(l.ctx, l.svcCtx.DB, req.Token)
	if err != nil {
		return nil, err
	}
	if len(req.Password) < 8 {
		return nil, errPasswordTooWeak
	}

	email := strings.ToLower(strings.TrimSpace(inv.Email))
	exists, err := l.svcCtx.DB.CheckEmailExists(l.ctx, email)
	if err != nil {
		l.Errorf("Error checking email existence: %v", err)
		return nil, fmt.Errorf("failed to check email availability")
	}
	if exists != 0 {
		return nil, errAccountExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		l.Errorf("Error hashing password: %v", err)
		return nil, fmt.Errorf("failed to process password")
	}

	name := strings.TrimSpace(req.FirstName + " " + req.LastName)
	if name == "" {
		// Use part before @ as default name
		if idx := strings.Index(email, "@"); idx > 0 {
			name = email[:idx]
		}
	}

	// The invitation was emailed to this address, so using it proves the
	// address: the account is active and verified at once.
	var user db.CreateUserRow
	err = l.svcCtx.DB.ExecTx(l.ctx, func(q *db.Queries) error {
		accepted, err := q.AcceptOrgInvitation(l.ctx, inv.ID)
		if err != nil {
			return err
		}
		if accepted == 0 {
			return errInviteUsed
		}
		user, err = q.CreateUser(l.ctx, db.CreateUserParams{
			ID:            uuid.New().String(),
			Email:         email,
			PasswordHash:  string(hashedPassword),
			Name:          name,
			Role:          invitedPlatformRole,
			Status:        "active",
			EmailVerified: int64(1),
			Phone:         sql.NullString{String: req.Phone, Valid: req.Phone != ""},
		})
		if err != nil {
			return err
		}
		return q.AddUserToOrganization(l.ctx, db.AddUserToOrganizationParams{
			OrgID:  inv.OrgID,
			UserID: user.ID,
			Role:   sql.NullString{String: inv.Role, Valid: true},
		})
	})
	if errors.Is(err, errInviteUsed) {
		return nil, err
	}
	if err != nil {
		l.Errorf("Error creating invited account: %v", err)
		return nil, fmt.Errorf("failed to create account")
	}

	accessExpire := time.Duration(l.svcCtx.Config.Auth.AccessExpire) * time.Second
	accessToken, err := utils.GenerateToken(
		user.ID,
		user.Email,
		user.Role,
		l.svcCtx.Config.Auth.AccessSecret,
		accessExpire,
	)
	if err != nil {
		l.Errorf("Error generating access token: %v", err)
		return nil, fmt.Errorf("account created but failed to generate token - please login")
	}

	l.Infof("Invited user registered: %s (brand %s, role %s)", email, inv.OrgID, inv.Role)

	return &types.RegisterResponse{
		Token: accessToken,
		User: types.UserInfo{
			Id:        user.ID,
			Email:     user.Email,
			Name:      user.Name,
			Role:      user.Role,
			Active:    true,
			CreatedAt: utils.FormatNullString(user.CreatedAt),
		},
	}, nil
}
