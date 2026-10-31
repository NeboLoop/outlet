package lists

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateCustomFieldLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateCustomFieldLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateCustomFieldLogic {
	return &CreateCustomFieldLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateCustomFieldLogic) CreateCustomField(req *types.CreateCustomFieldRequest) (resp *types.CustomFieldInfo, err error) {
	listID, err := strconv.ParseInt(req.ListId, 10, 64)
	if err != nil {
		l.Errorf("Invalid list ID: %s", req.ListId)
		return nil, err
	}

	if _, err := requireList(l.ctx, l.svcCtx, listID, access.Member); err != nil {
		return nil, err
	}

	// Normalize field key to lowercase with underscores
	fieldKey := strings.ToLower(strings.ReplaceAll(req.FieldKey, " ", "_"))
	fieldKey = strings.ReplaceAll(fieldKey, "-", "_")

	// Serialize options to JSON if present
	var optionsJSON sql.NullString
	if len(req.Options) > 0 {
		optionsData, err := json.Marshal(req.Options)
		if err != nil {
			l.Errorf("Failed to marshal options: %v", err)
			return nil, err
		}
		optionsJSON = sql.NullString{String: string(optionsData), Valid: true}
	}

	var required int64 = 0
	if req.Required {
		required = 1
	}

	field, err := l.svcCtx.DB.CreateCustomField(l.ctx, db.CreateCustomFieldParams{
		ID:           uuid.New().String(),
		ListID:       listID,
		Name:         req.Name,
		FieldKey:     fieldKey,
		FieldType:    req.FieldType,
		Options:      optionsJSON,
		Required:     required,
		DefaultValue: sql.NullString{String: req.DefaultValue, Valid: req.DefaultValue != ""},
		Placeholder:  sql.NullString{String: req.Placeholder, Valid: req.Placeholder != ""},
		SortOrder:    int64(req.SortOrder),
	})
	if err != nil {
		l.Errorf("Failed to create custom field: %v", err)
		return nil, err
	}

	return &types.CustomFieldInfo{
		Id:           field.ID,
		ListId:       req.ListId,
		Name:         field.Name,
		FieldKey:     field.FieldKey,
		FieldType:    field.FieldType,
		Options:      req.Options,
		Required:     field.Required == 1,
		DefaultValue: field.DefaultValue.String,
		Placeholder:  field.Placeholder.String,
		SortOrder:    int(field.SortOrder),
		CreatedAt:    field.CreatedAt.String,
		UpdatedAt:    field.UpdatedAt.String,
	}, nil
}
