package organizations

import (
	"context"
	"database/sql"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateOrganizationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateOrganizationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateOrganizationLogic {
	return &UpdateOrganizationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateOrganizationLogic) UpdateOrganization(req *types.UpdateOrgRequest) (resp *types.OrgInfo, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, req.Id, access.Admin); err != nil {
		return nil, err
	}

	var org db.Organization

	// Update max_contacts separately since it needs to allow NULL (unlimited)
	// max_contacts: 0 = unlimited, >0 = limit
	if req.MaxContacts >= 0 {
		maxContacts := sql.NullInt64{Valid: false} // NULL = unlimited
		if req.MaxContacts > 0 {
			maxContacts = sql.NullInt64{Int64: int64(req.MaxContacts), Valid: true}
		}
		org, err = l.svcCtx.DB.UpdateOrgMaxContacts(l.ctx, db.UpdateOrgMaxContactsParams{
			ID:          req.Id,
			MaxContacts: maxContacts,
		})
		if err != nil {
			return nil, err
		}
	}

	// Update other fields if provided
	if req.Name != "" || req.AppUrl != "" {
		params := db.UpdateOrganizationParams{
			ID:     req.Id,
			Name:   req.Name,
			AppUrl: req.AppUrl,
		}
		org, err = l.svcCtx.DB.UpdateOrganization(l.ctx, params)
		if err != nil {
			return nil, err
		}
	}

	// If we didn't update anything, fetch the current org
	if org.ID == "" {
		org, err = l.svcCtx.DB.GetOrganizationByID(l.ctx, req.Id)
		if err != nil {
			return nil, err
		}
	}

	return &types.OrgInfo{
		Id:               org.ID,
		Name:             org.Name,
		Slug:             org.Slug,
		ApiKey:           org.ApiKey,
		BillingStatus:    "trial",
		Plan:             "starter",
		MaxContacts:      int(org.MaxContacts.Int64),
		StripeConfigured: false,
		FromName:         org.FromName.String,
		FromEmail:        org.FromEmail.String,
		ReplyTo:          org.ReplyTo.String,
		AppUrl:           org.AppUrl.String,
		CreatedAt:        org.CreatedAt.String,
	}, nil
}
