package organizations

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"net/mail"
	"strings"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateInvitationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Invite a person to a brand. Outlet is invite-only: the invitation decides the brand and role.
func NewCreateInvitationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateInvitationLogic {
	return &CreateInvitationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateInvitationLogic) CreateInvitation(req *types.CreateInvitationRequest) (resp *types.CreateInvitationResponse, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, req.Id, access.Admin); err != nil {
		return nil, err
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return nil, errorx.NewBadRequestError("Enter a valid email address.")
	}
	role := req.Role
	if role == "" {
		role = string(access.Member)
	}
	if !access.BrandRole(role) {
		return nil, errorx.NewBadRequestError("A brand role is admin or member.")
	}

	brand, err := l.svcCtx.DB.GetOrganizationByID(l.ctx, req.Id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errorx.NewNotFoundError("Brand not found.")
	}
	if err != nil {
		return nil, err
	}

	// Someone who already has an account gets an invitation too, and
	// accepts it signed in: nobody joins a brand without reading the email.
	exists, err := l.svcCtx.DB.CheckEmailExists(l.ctx, email)
	if err != nil {
		return nil, err
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(tokenBytes)

	// A new invitation replaces an earlier one for the same person and brand.
	if err := l.svcCtx.DB.DeletePendingOrgInvitations(l.ctx, db.DeletePendingOrgInvitationsParams{OrgID: brand.ID, Email: email}); err != nil {
		return nil, err
	}
	invitation, err := l.svcCtx.DB.CreateOrgInvitation(l.ctx, db.CreateOrgInvitationParams{
		ID:    uuid.NewString(),
		OrgID: brand.ID,
		Email: email,
		Role:  sql.NullString{String: role, Valid: true},
		Token: token,
	})
	if err != nil {
		return nil, err
	}

	// The link goes out only by email, never back to the inviter: using it
	// proves the person reads that mailbox.
	inviteURL := fmt.Sprintf("%s/auth/register?token=%s", strings.TrimSuffix(l.svcCtx.Config.App.BaseURL, "/"), token)
	subject := fmt.Sprintf("You're invited to %s on Outlet", brand.Name)
	action := "Create your account"
	if exists != 0 {
		action = "Sign in and join"
	}
	body := fmt.Sprintf(`<html>
<body>
<p>You're invited to join %s on Outlet as %s.</p>
<p><a href="%s">%s</a></p>
<p>This link works once and expires in 7 days. If you weren't expecting it, you can ignore this email.</p>
</body>
</html>`, html.EscapeString(brand.Name), role, inviteURL, action)

	emailSent := true
	if err := l.svcCtx.EmailService.SendEmail(l.ctx, email, subject, body); err != nil {
		l.Errorf("Invitation email to %s failed: %v", email, err)
		emailSent = false
	}

	return &types.CreateInvitationResponse{
		Email:     email,
		Role:      role,
		ExpiresAt: invitation.ExpiresAt,
		EmailSent: emailSent,
	}, nil
}
