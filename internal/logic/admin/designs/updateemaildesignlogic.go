package designs

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateEmailDesignLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateEmailDesignLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateEmailDesignLogic {
	return &UpdateEmailDesignLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateEmailDesignLogic) UpdateEmailDesign(req *types.UpdateEmailDesignRequest) (resp *types.EmailDesignInfo, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	designID, err := strconv.ParseInt(req.Id, 10, 64)
	if err != nil {
		return nil, errors.New("invalid design ID")
	}

	var isActive sql.NullInt64
	if req.IsActive {
		isActive = sql.NullInt64{Int64: 1, Valid: true}
	} else {
		isActive = sql.NullInt64{Int64: 0, Valid: true}
	}

	design, err := l.svcCtx.DB.UpdateEmailDesign(l.ctx, db.UpdateEmailDesignParams{
		ID:          designID,
		OrgID:       orgID,
		Name:        sql.NullString{String: req.Name, Valid: req.Name != ""},
		Slug:        sql.NullString{String: req.Slug, Valid: req.Slug != ""},
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
		Category:    sql.NullString{String: req.Category, Valid: req.Category != ""},
		HtmlBody:    sql.NullString{String: req.HtmlBody, Valid: req.HtmlBody != ""},
		PlainText:   sql.NullString{String: req.PlainText, Valid: req.PlainText != ""},
		IsActive:    isActive,
	})
	if err != nil {
		l.Errorf("Failed to update email design: %v", err)
		return nil, err
	}

	return &types.EmailDesignInfo{
		Id:          strconv.FormatInt(design.ID, 10),
		OrgId:       design.OrgID,
		Name:        design.Name,
		Slug:        design.Slug,
		Description: design.Description.String,
		Category:    design.Category.String,
		HtmlBody:    design.HtmlBody,
		PlainText:   design.PlainText.String,
		IsActive:    design.IsActive.Int64 == 1,
		CreatedAt:   utils.FormatNullString(design.CreatedAt),
		UpdatedAt:   utils.FormatNullString(design.UpdatedAt),
	}, nil
}
