package organizations

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOrganizationBySlugLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetOrganizationBySlugLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOrganizationBySlugLogic {
	return &GetOrganizationBySlugLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetOrganizationBySlugLogic) GetOrganizationBySlug(req *types.GetOrgBySlugRequest) (resp *types.OrgInfo, err error) {
	org, err := l.svcCtx.DB.GetOrganizationBySlug(l.ctx, req.Slug)
	if err != nil {
		return nil, err
	}

	caller, err := access.Require(l.ctx, l.svcCtx.DB, org.ID, access.Member)
	if err != nil {
		return nil, err
	}

	return &types.OrgInfo{
		Id:               org.ID,
		Name:             org.Name,
		Slug:             org.Slug,
		ApiKey:           brandAPIKey(org, caller.Role),
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
