package transactional

import (
	"context"
	"database/sql"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateTransactionalEmailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateTransactionalEmailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateTransactionalEmailLogic {
	return &UpdateTransactionalEmailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateTransactionalEmailLogic) UpdateTransactionalEmail(req *types.UpdateTransactionalEmailRequest) (resp *types.TransactionalEmailInfo, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	var isActive sql.NullInt64
	if req.IsActive {
		isActive = sql.NullInt64{Int64: 1, Valid: true}
	} else {
		isActive = sql.NullInt64{Int64: 0, Valid: true}
	}

	email, err := l.svcCtx.DB.UpdateTransactionalEmail(l.ctx, db.UpdateTransactionalEmailParams{
		ID:          req.Id,
		OrgID:       orgID,
		Name:        sql.NullString{String: req.Name, Valid: req.Name != ""},
		Slug:        sql.NullString{String: req.Slug, Valid: req.Slug != ""},
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
		Subject:     sql.NullString{String: req.Subject, Valid: req.Subject != ""},
		HtmlBody:    sql.NullString{String: req.HtmlBody, Valid: req.HtmlBody != ""},
		PlainText:   sql.NullString{String: req.PlainText, Valid: req.PlainText != ""},
		FromName:    sql.NullString{String: req.FromName, Valid: req.FromName != ""},
		FromEmail:   sql.NullString{String: req.FromEmail, Valid: req.FromEmail != ""},
		ReplyTo:     sql.NullString{String: req.ReplyTo, Valid: req.ReplyTo != ""},
		IsActive:    isActive,
	})
	if err != nil {
		l.Errorf("Failed to update transactional email: %v", err)
		return nil, err
	}

	info := transactionalEmailToInfo(email)
	return &info, nil
}
