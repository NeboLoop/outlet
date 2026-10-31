package lists

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateCustomFieldLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateCustomFieldLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCustomFieldLogic {
	return &UpdateCustomFieldLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateCustomFieldLogic) UpdateCustomField(req *types.UpdateCustomFieldRequest) (resp *types.CustomFieldInfo, err error) {
	// Get existing field to merge with updates
	existing, err := requireCustomField(l.ctx, l.svcCtx, req.ListId, req.FieldId, access.Member)
	if err != nil {
		return nil, err
	}

	// Use existing values as defaults
	name := existing.Name
	if req.Name != "" {
		name = req.Name
	}

	fieldKey := existing.FieldKey
	if req.FieldKey != "" {
		fieldKey = strings.ToLower(strings.ReplaceAll(req.FieldKey, " ", "_"))
		fieldKey = strings.ReplaceAll(fieldKey, "-", "_")
	}

	fieldType := existing.FieldType
	if req.FieldType != "" {
		fieldType = req.FieldType
	}

	options := existing.Options
	if req.Options != nil {
		optionsData, err := json.Marshal(req.Options)
		if err != nil {
			l.Errorf("Failed to marshal options: %v", err)
			return nil, err
		}
		options = sql.NullString{String: string(optionsData), Valid: true}
	}

	required := existing.Required
	if req.Required != nil {
		if *req.Required {
			required = 1
		} else {
			required = 0
		}
	}

	defaultValue := existing.DefaultValue
	if req.DefaultValue != "" {
		defaultValue = sql.NullString{String: req.DefaultValue, Valid: true}
	}

	placeholder := existing.Placeholder
	if req.Placeholder != "" {
		placeholder = sql.NullString{String: req.Placeholder, Valid: true}
	}

	sortOrder := existing.SortOrder
	if req.SortOrder != nil {
		sortOrder = int64(*req.SortOrder)
	}

	field, err := l.svcCtx.DB.UpdateCustomField(l.ctx, db.UpdateCustomFieldParams{
		ID:           req.FieldId,
		Name:         name,
		FieldKey:     fieldKey,
		FieldType:    fieldType,
		Options:      options,
		Required:     required,
		DefaultValue: defaultValue,
		Placeholder:  placeholder,
		SortOrder:    sortOrder,
	})
	if err != nil {
		l.Errorf("Failed to update custom field %s: %v", req.FieldId, err)
		return nil, err
	}

	var respOptions []string
	if field.Options.Valid && field.Options.String != "" {
		_ = json.Unmarshal([]byte(field.Options.String), &respOptions)
	}

	return &types.CustomFieldInfo{
		Id:           field.ID,
		ListId:       req.ListId,
		Name:         field.Name,
		FieldKey:     field.FieldKey,
		FieldType:    field.FieldType,
		Options:      respOptions,
		Required:     field.Required == 1,
		DefaultValue: field.DefaultValue.String,
		Placeholder:  field.Placeholder.String,
		SortOrder:    int(field.SortOrder),
		CreatedAt:    field.CreatedAt.String,
		UpdatedAt:    field.UpdatedAt.String,
	}, nil
}
