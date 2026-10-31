package lists

import (
	"context"
	"encoding/json"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCustomFieldLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCustomFieldLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCustomFieldLogic {
	return &GetCustomFieldLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCustomFieldLogic) GetCustomField(req *types.GetCustomFieldRequest) (resp *types.CustomFieldInfo, err error) {
	field, err := requireCustomField(l.ctx, l.svcCtx, req.ListId, req.FieldId, access.Member)
	if err != nil {
		return nil, err
	}

	var options []string
	if field.Options.Valid && field.Options.String != "" {
		_ = json.Unmarshal([]byte(field.Options.String), &options)
	}

	return &types.CustomFieldInfo{
		Id:           field.ID,
		ListId:       req.ListId,
		Name:         field.Name,
		FieldKey:     field.FieldKey,
		FieldType:    field.FieldType,
		Options:      options,
		Required:     field.Required == 1,
		DefaultValue: field.DefaultValue.String,
		Placeholder:  field.Placeholder.String,
		SortOrder:    int(field.SortOrder),
		CreatedAt:    field.CreatedAt.String,
		UpdatedAt:    field.UpdatedAt.String,
	}, nil
}
