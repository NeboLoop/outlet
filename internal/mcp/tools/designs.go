package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/mcp/mcpctx"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// designActions defines valid actions for designs.
var designActions = []string{"create", "list", "get", "update", "delete"}

// DesignInput defines input for the design tool.
type DesignInput struct {
	Action string `json:"action" jsonschema:"required,Action to perform: create, list, get, update, delete"`

	// Common
	ID string `json:"id,omitempty" jsonschema:"Design ID (for get, update, delete)"`

	// List filter
	Category string `json:"category,omitempty" jsonschema:"Filter by category (for list)"`

	// Create/Update fields
	Name        string `json:"name,omitempty" jsonschema:"Design name (required for create)"`
	Slug        string `json:"slug,omitempty" jsonschema:"URL-friendly slug"`
	Description string `json:"description,omitempty" jsonschema:"Design description"`
	HTMLBody    string `json:"html_body,omitempty" jsonschema:"HTML content of the design (required for create)"`
	PlainText   string `json:"plain_text,omitempty" jsonschema:"Plain text version"`
	Active      *bool  `json:"active,omitempty" jsonschema:"Whether the design is active (default: true)"`
}

// DesignItem represents a design in list output.
type DesignItem struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
	Active      bool   `json:"active"`
	CreatedAt   string `json:"created_at"`
}

// DesignListOutput defines output for design list.
type DesignListOutput struct {
	Designs []DesignItem `json:"designs"`
	Total   int          `json:"total"`
}

// DesignCreateOutput defines output for design create.
type DesignCreateOutput struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	Active  bool   `json:"active"`
	Created bool   `json:"created"`
}

// DesignGetOutput defines output for design get.
type DesignGetOutput struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Description  string `json:"description,omitempty"`
	Category     string `json:"category,omitempty"`
	HTMLBody     string `json:"html_body"`
	PlainText    string `json:"plain_text,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	Active       bool   `json:"active"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// DesignUpdateOutput defines output for design update.
type DesignUpdateOutput struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	Active  bool   `json:"active"`
	Updated bool   `json:"updated"`
}

// RegisterDesignTool registers the design tool.
func RegisterDesignTool(server *mcp.Server, toolCtx *mcpctx.ToolContext) {
	mcp.AddTool(server, &mcp.Tool{
		Name:  "design",
		Title: "Email Design Templates",
		Description: `Manage reusable email design templates.

PREREQUISITE: You must first select a brand using brand(resource: brand, action: select).

Actions and Required Fields:
- create: Create a new design template (requires: name, html_body)
- list: List all design templates (optional: category filter)
- get: Get design template with full HTML content (requires: id)
- update: Update a design template (requires: id)
- delete: Delete a design template (requires: id)

Categories:
Templates can be organized by category (e.g., 'newsletter', 'welcome', 'promotional', 'transactional').

Examples:
  design(action: create, name: "Welcome Email", slug: "welcome", html_body: "<h1>Welcome!</h1>", category: "welcome")
  design(action: list)
  design(action: list, category: "newsletter")
  design(action: get, id: "123")
  design(action: update, id: "123", name: "Updated Welcome")
  design(action: delete, id: "123")`,
	}, designHandler(toolCtx))
}

func designHandler(toolCtx *mcpctx.ToolContext) func(ctx context.Context, req *mcp.CallToolRequest, input DesignInput) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input DesignInput) (*mcp.CallToolResult, any, error) {
		// Validate action
		if !slices.Contains(designActions, input.Action) {
			return nil, nil, mcpctx.NewValidationError(
				fmt.Sprintf("invalid action '%s', must be: %s", input.Action, strings.Join(designActions, ", ")),
				"action")
		}

		switch input.Action {
		case "create":
			return handleDesignCreate(ctx, toolCtx, input)
		case "list":
			return handleDesignList(ctx, toolCtx, input)
		case "get":
			return handleDesignGet(ctx, toolCtx, input)
		case "update":
			return handleDesignUpdate(ctx, toolCtx, input)
		case "delete":
			return handleDesignDelete(ctx, toolCtx, input)
		}
		return nil, nil, nil
	}
}

func handleDesignCreate(ctx context.Context, toolCtx *mcpctx.ToolContext, input DesignInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.Name) == "" {
		return nil, nil, mcpctx.NewValidationError("name is required", "name")
	}
	if strings.TrimSpace(input.HTMLBody) == "" {
		return nil, nil, mcpctx.NewValidationError("html_body is required", "html_body")
	}

	// Generate slug if not provided
	slug := input.Slug
	if slug == "" {
		slug = strings.ToLower(strings.ReplaceAll(input.Name, " ", "-"))
	}

	active := true
	if input.Active != nil {
		active = *input.Active
	}

	design, err := toolCtx.DB().CreateEmailDesign(ctx, db.CreateEmailDesignParams{
		OrgID:       toolCtx.BrandID(ctx),
		Name:        input.Name,
		Slug:        slug,
		Description: sql.NullString{String: input.Description, Valid: input.Description != ""},
		Category:    sql.NullString{String: input.Category, Valid: input.Category != ""},
		HtmlBody:    input.HTMLBody,
		PlainText:   sql.NullString{String: input.PlainText, Valid: input.PlainText != ""},
		IsActive:    sql.NullInt64{Int64: boolToInt64(active), Valid: true},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create design: %w", err)
	}

	return nil, DesignCreateOutput{
		ID:      design.ID,
		Name:    design.Name,
		Slug:    design.Slug,
		Active:  int64ToBool(design.IsActive),
		Created: true,
	}, nil
}

func handleDesignList(ctx context.Context, toolCtx *mcpctx.ToolContext, input DesignInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	var designs []db.EmailDesign
	var err error

	if input.Category != "" {
		designs, err = toolCtx.DB().ListEmailDesignsByCategory(ctx, db.ListEmailDesignsByCategoryParams{
			OrgID:    toolCtx.BrandID(ctx),
			Category: sql.NullString{String: input.Category, Valid: true},
		})
	} else {
		designs, err = toolCtx.DB().ListEmailDesigns(ctx, toolCtx.BrandID(ctx))
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list designs: %w", err)
	}

	items := make([]DesignItem, 0, len(designs))
	for _, d := range designs {
		items = append(items, DesignItem{
			ID:          d.ID,
			Name:        d.Name,
			Slug:        d.Slug,
			Description: d.Description.String,
			Category:    d.Category.String,
			Active:      int64ToBool(d.IsActive),
			CreatedAt:   d.CreatedAt.String,
		})
	}

	return nil, DesignListOutput{
		Designs: items,
		Total:   len(items),
	}, nil
}

func handleDesignGet(ctx context.Context, toolCtx *mcpctx.ToolContext, input DesignInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	id, err := strconv.ParseInt(input.ID, 10, 64)
	if err != nil {
		return nil, nil, mcpctx.NewValidationError("id must be a valid number", "id")
	}

	design, err := toolCtx.DB().GetEmailDesign(ctx, db.GetEmailDesignParams{
		ID:    id,
		OrgID: toolCtx.BrandID(ctx),
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("design %s not found", input.ID))
	}

	return nil, DesignGetOutput{
		ID:           design.ID,
		Name:         design.Name,
		Slug:         design.Slug,
		Description:  design.Description.String,
		Category:     design.Category.String,
		HTMLBody:     design.HtmlBody,
		PlainText:    design.PlainText.String,
		ThumbnailURL: design.ThumbnailUrl.String,
		Active:       int64ToBool(design.IsActive),
		CreatedAt:    design.CreatedAt.String,
		UpdatedAt:    design.UpdatedAt.String,
	}, nil
}

func handleDesignUpdate(ctx context.Context, toolCtx *mcpctx.ToolContext, input DesignInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	id, err := strconv.ParseInt(input.ID, 10, 64)
	if err != nil {
		return nil, nil, mcpctx.NewValidationError("id must be a valid number", "id")
	}

	// Verify design exists
	existing, err := toolCtx.DB().GetEmailDesign(ctx, db.GetEmailDesignParams{
		ID:    id,
		OrgID: toolCtx.BrandID(ctx),
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("design %s not found", input.ID))
	}

	// Build update - use existing values as defaults
	active := int64ToBool(existing.IsActive)
	if input.Active != nil {
		active = *input.Active
	}

	design, err := toolCtx.DB().UpdateEmailDesign(ctx, db.UpdateEmailDesignParams{
		ID:          id,
		OrgID:       toolCtx.BrandID(ctx),
		Name:        input.Name,
		Slug:        input.Slug,
		Description: input.Description,
		Category:    input.Category,
		HtmlBody:    input.HTMLBody,
		PlainText:   input.PlainText,
		IsActive:    sql.NullInt64{Int64: boolToInt64(active), Valid: true},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update design: %w", err)
	}

	return nil, DesignUpdateOutput{
		ID:      design.ID,
		Name:    design.Name,
		Slug:    design.Slug,
		Active:  int64ToBool(design.IsActive),
		Updated: true,
	}, nil
}

func handleDesignDelete(ctx context.Context, toolCtx *mcpctx.ToolContext, input DesignInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	id, err := strconv.ParseInt(input.ID, 10, 64)
	if err != nil {
		return nil, nil, mcpctx.NewValidationError("id must be a valid number", "id")
	}

	// Verify design exists
	_, err = toolCtx.DB().GetEmailDesign(ctx, db.GetEmailDesignParams{
		ID:    id,
		OrgID: toolCtx.BrandID(ctx),
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("design %s not found", input.ID))
	}

	err = toolCtx.DB().DeleteEmailDesign(ctx, db.DeleteEmailDesignParams{
		ID:    id,
		OrgID: toolCtx.BrandID(ctx),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to delete design: %w", err)
	}

	return nil, DeleteOutput{
		Success: true,
		Message: fmt.Sprintf("Design %s deleted successfully", input.ID),
	}, nil
}

// registerDesignToolToRegistry registers design tool to the direct-call registry.
func registerDesignToolToRegistry(registry *ToolRegistry, toolCtx *mcpctx.ToolContext) {
	registry.Register("design", func(ctx context.Context, args json.RawMessage) (interface{}, error) {
		var input DesignInput
		if err := json.Unmarshal(args, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		handler := designHandler(toolCtx)
		_, output, err := handler(ctx, nil, input)
		return output, err
	})
}
