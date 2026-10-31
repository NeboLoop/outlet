package designs

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateEmailDesignLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateEmailDesignLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateEmailDesignLogic {
	return &CreateEmailDesignLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateEmailDesignLogic) CreateEmailDesign(req *types.CreateEmailDesignRequest) (resp *types.EmailDesignInfo, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	category := req.Category
	if category == "" {
		category = "general"
	}

	var isActive sql.NullInt64
	if req.IsActive {
		isActive = sql.NullInt64{Int64: 1, Valid: true}
	} else {
		isActive = sql.NullInt64{Int64: 0, Valid: true}
	}

	design, err := l.svcCtx.DB.CreateEmailDesign(l.ctx, db.CreateEmailDesignParams{
		OrgID:       orgID,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
		Category:    sql.NullString{String: category, Valid: true},
		HtmlBody:    req.HtmlBody,
		PlainText:   sql.NullString{String: req.PlainText, Valid: req.PlainText != ""},
		IsActive:    isActive,
	})
	if err != nil {
		l.Errorf("Failed to create email design: %v", err)
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
