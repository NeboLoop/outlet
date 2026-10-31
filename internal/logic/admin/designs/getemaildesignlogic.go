package designs

import (
	"context"
	"errors"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetEmailDesignLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetEmailDesignLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetEmailDesignLogic {
	return &GetEmailDesignLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetEmailDesignLogic) GetEmailDesign(req *types.GetEmailDesignRequest) (resp *types.EmailDesignInfo, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	designID, err := strconv.ParseInt(req.Id, 10, 64)
	if err != nil {
		return nil, errors.New("invalid design ID")
	}

	design, err := l.svcCtx.DB.GetEmailDesign(l.ctx, db.GetEmailDesignParams{
		ID:    designID,
		OrgID: orgID,
	})
	if err != nil {
		l.Errorf("Failed to get email design: %v", err)
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
