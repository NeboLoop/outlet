package lists

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetListEmbedCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetListEmbedCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetListEmbedCodeLogic {
	return &GetListEmbedCodeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetListEmbedCodeLogic) GetListEmbedCode(req *types.GetEmbedCodeRequest) (resp *types.EmbedCodeResponse, err error) {
	listID, err := strconv.ParseInt(req.Id, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid list ID: %w", err)
	}

	if _, err := access.RequireOwner(l.ctx, l.svcCtx.DB, access.List, req.Id, access.Member); err != nil {
		return nil, err
	}
	list, err := l.svcCtx.DB.GetEmailList(l.ctx, listID)
	if err != nil {
		return nil, err
	}

	baseURL := l.svcCtx.Config.App.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:9888"
	}

	formURL := fmt.Sprintf("%s/s/%s", baseURL, list.PublicID)

	// Get custom fields for this list
	customFields, _ := l.svcCtx.DB.ListCustomFieldsByList(l.ctx, listID)

	// Build custom field inputs
	var customFieldsHTML strings.Builder
	for _, field := range customFields {
		requiredAttr := ""
		requiredLabel := ""
		if field.Required == 1 {
			requiredAttr = " required"
			requiredLabel = " *"
		}

		placeholder := field.Name
		if field.Placeholder.Valid && field.Placeholder.String != "" {
			placeholder = field.Placeholder.String
		}

		defaultValue := ""
		if field.DefaultValue.Valid {
			defaultValue = field.DefaultValue.String
		}

		switch field.FieldType {
		case "dropdown":
			var options []string
			if field.Options.Valid && field.Options.String != "" {
				_ = json.Unmarshal([]byte(field.Options.String), &options)
			}
			customFieldsHTML.WriteString(fmt.Sprintf(`  <div style="margin-bottom: 12px;">
    <select name="custom_fields[%s]"%s
      style="width: 100%%; padding: 10px; border: 1px solid #ccc; border-radius: 4px; font-size: 14px;">
      <option value="">%s%s</option>
`, field.FieldKey, requiredAttr, placeholder, requiredLabel))
			for _, opt := range options {
				selected := ""
				if opt == defaultValue {
					selected = " selected"
				}
				customFieldsHTML.WriteString(fmt.Sprintf(`      <option value="%s"%s>%s</option>
`, opt, selected, opt))
			}
			customFieldsHTML.WriteString(`    </select>
  </div>
`)
		case "number":
			customFieldsHTML.WriteString(fmt.Sprintf(`  <div style="margin-bottom: 12px;">
    <input type="number" name="custom_fields[%s]" placeholder="%s%s" value="%s"%s
      style="width: 100%%; padding: 10px; border: 1px solid #ccc; border-radius: 4px; font-size: 14px;">
  </div>
`, field.FieldKey, placeholder, requiredLabel, defaultValue, requiredAttr))
		case "date":
			customFieldsHTML.WriteString(fmt.Sprintf(`  <div style="margin-bottom: 12px;">
    <input type="date" name="custom_fields[%s]" placeholder="%s%s" value="%s"%s
      style="width: 100%%; padding: 10px; border: 1px solid #ccc; border-radius: 4px; font-size: 14px;">
  </div>
`, field.FieldKey, placeholder, requiredLabel, defaultValue, requiredAttr))
		default: // text
			customFieldsHTML.WriteString(fmt.Sprintf(`  <div style="margin-bottom: 12px;">
    <input type="text" name="custom_fields[%s]" placeholder="%s%s" value="%s"%s
      style="width: 100%%; padding: 10px; border: 1px solid #ccc; border-radius: 4px; font-size: 14px;">
  </div>
`, field.FieldKey, placeholder, requiredLabel, defaultValue, requiredAttr))
		}
	}

	html := fmt.Sprintf(`<!-- Outlet.sh Subscribe Form for "%s" -->
<form action="%s" method="POST" style="max-width: 400px; font-family: system-ui, sans-serif;">
  <div style="margin-bottom: 12px;">
    <input type="email" name="email" placeholder="Email address" required
      style="width: 100%%; padding: 10px; border: 1px solid #ccc; border-radius: 4px; font-size: 14px;">
  </div>
  <div style="margin-bottom: 12px;">
    <input type="text" name="name" placeholder="Your name (optional)"
      style="width: 100%%; padding: 10px; border: 1px solid #ccc; border-radius: 4px; font-size: 14px;">
  </div>
%s  <button type="submit"
    style="width: 100%%; padding: 12px; background: #0066ff; color: white; border: none; border-radius: 4px; font-size: 14px; cursor: pointer;">
    Subscribe
  </button>
</form>`, list.Name, formURL, customFieldsHTML.String())

	return &types.EmbedCodeResponse{
		Html:     html,
		ListId:   strconv.FormatInt(list.ID, 10),
		PublicId: list.PublicID,
		Slug:     list.Slug,
		BaseUrl:  baseURL,
	}, nil
}
