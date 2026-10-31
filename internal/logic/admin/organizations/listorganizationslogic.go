package organizations

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListOrganizationsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListOrganizationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListOrganizationsLogic {
	return &ListOrganizationsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListOrganizationsLogic) ListOrganizations() (resp *types.OrgListResponse, err error) {
	user, ok := access.User(l.ctx)
	if !ok {
		return nil, errorx.NewUnauthorizedError("Sign in first.")
	}
	orgs, err := access.Brands(l.ctx, l.svcCtx.DB, user.ID)
	if err != nil {
		return nil, err
	}

	orgInfos := make([]types.OrgInfo, len(orgs))
	for i, org := range orgs {
		orgInfos[i] = types.OrgInfo{
			Id:               org.ID,
			Name:             org.Name,
			Slug:             org.Slug,
			ApiKey:           brandAPIKey(org.Organization, org.Role),
			BillingStatus:    "trial",
			Plan:             "starter",
			MaxContacts:      int(org.MaxContacts.Int64),
			StripeConfigured: false,
			AppUrl:           org.AppUrl.String,
			FromName:         org.FromName.String,
			FromEmail:        org.FromEmail.String,
			ReplyTo:          org.ReplyTo.String,
			CreatedAt:        org.CreatedAt.String,
		}
	}

	return &types.OrgListResponse{
		Organizations: orgInfos,
		Total:         len(orgInfos),
	}, nil
}

// brandAPIKey shows the brand's API key to its admins only: it sends mail as
// the brand.
func brandAPIKey(org db.Organization, role access.Role) string {
	if role == access.Member {
		return ""
	}
	return org.ApiKey
}
