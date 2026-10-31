package transactional

import (
	"context"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListTransactionalEmailsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListTransactionalEmailsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListTransactionalEmailsLogic {
	return &ListTransactionalEmailsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListTransactionalEmailsLogic) ListTransactionalEmails(req *types.ListTransactionalEmailsRequest) (resp *types.ListTransactionalEmailsResponse, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	emails, err := l.svcCtx.DB.ListTransactionalEmails(l.ctx, orgID)
	if err != nil {
		l.Errorf("Failed to list transactional emails: %v", err)
		return nil, err
	}

	templates := make([]types.TransactionalEmailInfo, 0)
	for _, email := range emails {
		templates = append(templates, transactionalEmailToInfo(email))
	}

	return &types.ListTransactionalEmailsResponse{
		Templates: templates,
	}, nil
}

func transactionalEmailToInfo(email db.TransactionalEmail) types.TransactionalEmailInfo {
	info := types.TransactionalEmailInfo{
		Id:        email.ID,
		OrgId:     email.OrgID,
		Name:      email.Name,
		Slug:      email.Slug,
		Subject:   email.Subject,
		HtmlBody:  email.HtmlBody,
		IsActive:  email.IsActive.Int64 == 1,
		CreatedAt: utils.FormatNullString(email.CreatedAt),
		UpdatedAt: utils.FormatNullString(email.UpdatedAt),
	}

	if email.DesignID.Valid {
		designID := strconv.FormatInt(email.DesignID.Int64, 10)
		info.DesignId = &designID
	}
	if email.Description.Valid {
		info.Description = email.Description.String
	}
	if email.PlainText.Valid {
		info.PlainText = email.PlainText.String
	}
	if email.FromName.Valid {
		info.FromName = email.FromName.String
	}
	if email.FromEmail.Valid {
		info.FromEmail = email.FromEmail.String
	}
	if email.ReplyTo.Valid {
		info.ReplyTo = email.ReplyTo.String
	}

	return info
}
