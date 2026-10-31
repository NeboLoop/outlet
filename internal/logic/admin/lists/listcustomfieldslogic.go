package lists

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListCustomFieldsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListCustomFieldsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListCustomFieldsLogic {
	return &ListCustomFieldsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListCustomFieldsLogic) ListCustomFields(req *types.ListCustomFieldsRequest) (resp *types.ListCustomFieldsResponse, err error) {
	listID, err := strconv.ParseInt(req.ListId, 10, 64)
	if err != nil {
		l.Errorf("Invalid list ID: %s", req.ListId)
		return nil, err
	}

	if _, err := requireList(l.ctx, l.svcCtx, listID, access.Member); err != nil {
		return nil, err
	}

	fields, err := l.svcCtx.DB.ListCustomFieldsByList(l.ctx, listID)
	if err != nil {
		l.Errorf("Failed to list custom fields for list %d: %v", listID, err)
		return nil, err
	}

	fieldInfos := make([]types.CustomFieldInfo, len(fields))
	for i, field := range fields {
		var options []string
		if field.Options.Valid && field.Options.String != "" {
			_ = json.Unmarshal([]byte(field.Options.String), &options)
		}

		fieldInfos[i] = types.CustomFieldInfo{
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
		}
	}

	return &types.ListCustomFieldsResponse{
		Fields: fieldInfos,
		Total:  len(fieldInfos),
	}, nil
}
