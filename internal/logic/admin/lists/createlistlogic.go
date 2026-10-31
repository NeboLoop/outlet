package lists

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

type CreateListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateListLogic {
	return &CreateListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateListLogic) CreateList(req *types.CreateListRequest) (resp *types.ListInfo, err error) {
	// The request names its brand; it never falls back to some other brand.
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	list, err := l.svcCtx.DB.CreateEmailList(l.ctx, db.CreateEmailListParams{
		PublicID: utils.GeneratePublicID(),
		OrgID:    orgID,
		Name:     req.Name,
		Slug:     req.Slug,
		Description: sql.NullString{
			String: req.Description,
			Valid:  req.Description != "",
		},
		DoubleOptin: sql.NullInt64{
			Int64: boolToInt64(req.DoubleOptin),
			Valid: true,
		},
	})
	if err != nil {
		l.Errorf("Failed to create list: %v", err)
		return nil, err
	}

	return &types.ListInfo{
		Id:              strconv.FormatInt(list.ID, 10),
		PublicId:        list.PublicID,
		OrgId:           list.OrgID,
		Name:            list.Name,
		Slug:            list.Slug,
		Description:     list.Description.String,
		DoubleOptin:     list.DoubleOptin.Int64 == 1,
		SubscriberCount: 0,
		CreatedAt:       utils.FormatNullString(list.CreatedAt),
		UpdatedAt:       utils.FormatNullString(list.UpdatedAt),
	}, nil
}

func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
