package organizations

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListOrgPeopleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// A brand's people: its members with their roles, and invitations not yet accepted with when they expire (brand admins)
func NewListOrgPeopleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListOrgPeopleLogic {
	return &ListOrgPeopleLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ListOrgPeople lists a brand's members and pending invitations. People are
// the brand admin's to manage, as inviting is.
func (l *ListOrgPeopleLogic) ListOrgPeople(req *types.ListOrgPeopleRequest) (resp *types.ListOrgPeopleResponse, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, req.Id, access.Admin); err != nil {
		return nil, err
	}
	users, err := l.svcCtx.DB.GetOrganizationUsers(l.ctx, req.Id)
	if err != nil {
		return nil, err
	}
	invitations, err := l.svcCtx.DB.ListPendingOrgInvitations(l.ctx, req.Id)
	if err != nil {
		return nil, err
	}

	resp = &types.ListOrgPeopleResponse{
		Members:     make([]types.OrgMember, 0, len(users)),
		Invitations: make([]types.OrgPendingInvitation, 0, len(invitations)),
	}
	for _, u := range users {
		role := u.OrgRole.String
		if role == "" {
			role = string(access.Member)
		}
		resp.Members = append(resp.Members, types.OrgMember{
			UserId: u.ID, Name: u.Name, Email: u.Email, Role: role, JoinedAt: u.JoinedAt.String,
		})
	}
	for _, i := range invitations {
		resp.Invitations = append(resp.Invitations, types.OrgPendingInvitation{
			Email: i.Email, Role: i.Role, ExpiresAt: i.ExpiresAt, Expired: i.Expired == 1, CreatedAt: i.CreatedAt,
		})
	}
	return resp, nil
}
