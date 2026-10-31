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

type ListEmailDesignsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListEmailDesignsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListEmailDesignsLogic {
	return &ListEmailDesignsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListEmailDesignsLogic) ListEmailDesigns(req *types.ListEmailDesignsRequest) (resp *types.ListEmailDesignsResponse, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	designs := make([]types.EmailDesignInfo, 0)

	if req.Category != "" {
		dbDesigns, err := l.svcCtx.DB.ListEmailDesignsByCategory(l.ctx, db.ListEmailDesignsByCategoryParams{
			OrgID:    orgID,
			Category: sql.NullString{String: req.Category, Valid: true},
		})
		if err != nil {
			l.Errorf("Failed to list designs by category: %v", err)
			return nil, err
		}
		for _, d := range dbDesigns {
			designs = append(designs, mapDesignToInfo(d))
		}
	} else {
		dbDesigns, err := l.svcCtx.DB.ListEmailDesigns(l.ctx, orgID)
		if err != nil {
			l.Errorf("Failed to list designs: %v", err)
			return nil, err
		}
		for _, d := range dbDesigns {
			designs = append(designs, mapDesignToInfo(d))
		}
	}

	return &types.ListEmailDesignsResponse{
		Designs: designs,
		Total:   len(designs),
	}, nil
}

func mapDesignToInfo(d db.EmailDesign) types.EmailDesignInfo {
	return types.EmailDesignInfo{
		Id:          strconv.FormatInt(d.ID, 10),
		OrgId:       d.OrgID,
		Name:        d.Name,
		Slug:        d.Slug,
		Description: d.Description.String,
		Category:    d.Category.String,
		HtmlBody:    d.HtmlBody,
		PlainText:   d.PlainText.String,
		IsActive:    d.IsActive.Int64 == 1,
		CreatedAt:   utils.FormatNullString(d.CreatedAt),
		UpdatedAt:   utils.FormatNullString(d.UpdatedAt),
	}
}
