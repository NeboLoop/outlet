package transactional

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateTransactionalEmailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateTransactionalEmailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateTransactionalEmailLogic {
	return &CreateTransactionalEmailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateTransactionalEmailLogic) CreateTransactionalEmail(req *types.CreateTransactionalEmailRequest) (resp *types.TransactionalEmailInfo, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	var designID sql.NullInt64
	if req.DesignId != nil {
		parsedDesignID, err := strconv.ParseInt(*req.DesignId, 10, 64)
		if err != nil {
			return nil, errors.New("invalid design_id format")
		}
		designID = sql.NullInt64{Int64: parsedDesignID, Valid: true}
	}

	var isActive sql.NullInt64
	if req.IsActive {
		isActive = sql.NullInt64{Int64: 1, Valid: true}
	} else {
		isActive = sql.NullInt64{Int64: 0, Valid: true}
	}

	email, err := l.svcCtx.DB.CreateTransactionalEmail(l.ctx, db.CreateTransactionalEmailParams{
		ID:          uuid.New().String(),
		OrgID:       orgID,
		DesignID:    designID,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
		Subject:     req.Subject,
		HtmlBody:    req.HtmlBody,
		PlainText:   sql.NullString{String: req.PlainText, Valid: req.PlainText != ""},
		FromName:    sql.NullString{String: req.FromName, Valid: req.FromName != ""},
		FromEmail:   sql.NullString{String: req.FromEmail, Valid: req.FromEmail != ""},
		ReplyTo:     sql.NullString{String: req.ReplyTo, Valid: req.ReplyTo != ""},
		IsActive:    isActive,
	})
	if err != nil {
		l.Errorf("Failed to create transactional email: %v", err)
		return nil, err
	}

	info := transactionalEmailToInfo(email)
	return &info, nil
}
