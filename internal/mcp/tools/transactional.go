package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/mcp/mcpctx"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// transactionalActions defines valid actions for the transactional tool.
var transactionalActions = []string{"create", "list", "get", "update", "delete", "stats"}

// TransactionalInput defines input for the transactional tool.
type TransactionalInput struct {
	Action string `json:"action" jsonschema:"required,Action to perform: create, list, get, update, delete, stats"`

	// Common
	ID string `json:"id,omitempty" jsonschema:"Template ID (for get, update, delete, stats)"`

	// Create/Update fields
	Name        string `json:"name,omitempty" jsonschema:"Template name (create, update)"`
	Slug        string `json:"slug,omitempty" jsonschema:"URL-friendly slug (create, update)"`
	Description string `json:"description,omitempty" jsonschema:"Template description (create, update)"`
	Subject     string `json:"subject,omitempty" jsonschema:"Email subject line (create, update)"`
	HTMLBody    string `json:"html_body,omitempty" jsonschema:"HTML content of the email (create, update)"`
	PlainText   string `json:"plain_text,omitempty" jsonschema:"Plain text version (create, update)"`
	FromName    string `json:"from_name,omitempty" jsonschema:"Sender name override (create, update)"`
	FromEmail   string `json:"from_email,omitempty" jsonschema:"Sender email override (create, update)"`
	ReplyTo     string `json:"reply_to,omitempty" jsonschema:"Reply-to address (create, update)"`
	Active      *bool  `json:"active,omitempty" jsonschema:"Whether template is active (create, update)"`

	// Pagination
	Page     int `json:"page,omitempty" jsonschema:"Page number (default: 1)"`
	PageSize int `json:"page_size,omitempty" jsonschema:"Items per page (default: 20, max: 100)"`
}

// RegisterTransactionalTool registers the transactional email tool.
func RegisterTransactionalTool(server *mcp.Server, toolCtx *mcpctx.ToolContext) {
	mcp.AddTool(server, &mcp.Tool{
		Name:  "transactional",
		Title: "Transactional Email Management",
		Description: `Manage transactional email templates.

PREREQUISITE: You must first select a brand using brand(resource: brand, action: select).

Actions and Required Fields:
- create: Create a transactional email template (requires: name, subject, html_body)
- list: List all transactional email templates (optional: page, page_size)
- get: Get a transactional email template with full content (requires: id)
- update: Update a transactional email template (requires: id)
- delete: Delete a transactional email template (requires: id)
- stats: Get send statistics for a template (requires: id)

Examples:
  transactional(action: list)
  transactional(action: create, name: "Welcome Email", subject: "Welcome!", html_body: "<h1>Hello</h1>")
  transactional(action: get, id: "uuid")
  transactional(action: stats, id: "uuid")
  transactional(action: update, id: "uuid", subject: "New Subject")
  transactional(action: delete, id: "uuid")`,
	}, transactionalHandler(toolCtx))
}

func transactionalHandler(toolCtx *mcpctx.ToolContext) func(ctx context.Context, req *mcp.CallToolRequest, input TransactionalInput) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input TransactionalInput) (*mcp.CallToolResult, any, error) {
		// Validate action
		if !slices.Contains(transactionalActions, input.Action) {
			return nil, nil, mcpctx.NewValidationError(
				fmt.Sprintf("invalid action '%s', must be: %s", input.Action, strings.Join(transactionalActions, ", ")),
				"action")
		}

		switch input.Action {
		case "create":
			return handleTransactionalCreate(ctx, toolCtx, input)
		case "list":
			return handleTransactionalList(ctx, toolCtx, input)
		case "get":
			return handleTransactionalGet(ctx, toolCtx, input)
		case "update":
			return handleTransactionalUpdate(ctx, toolCtx, input)
		case "delete":
			return handleTransactionalDelete(ctx, toolCtx, input)
		case "stats":
			return handleTransactionalStats(ctx, toolCtx, input)
		}
		return nil, nil, nil // unreachable
	}
}

// ============================================================================
// OUTPUT TYPES
// ============================================================================

// TransactionalTemplateItem represents a template in the list.
type TransactionalTemplateItem struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Subject   string `json:"subject"`
	Active    bool   `json:"active"`
	CreatedAt string `json:"created_at,omitempty"`
}

// TransactionalListOutput defines output for list.
type TransactionalListOutput struct {
	Templates []TransactionalTemplateItem `json:"templates"`
	Total     int                         `json:"total"`
}

// TransactionalCreateOutput defines output for create.
type TransactionalCreateOutput struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	Subject string `json:"subject"`
	Active  bool   `json:"active"`
	Created bool   `json:"created"`
}

// TransactionalGetOutput defines output for get.
type TransactionalGetOutput struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
	Subject     string `json:"subject"`
	HTMLBody    string `json:"html_body"`
	PlainText   string `json:"plain_text,omitempty"`
	FromName    string `json:"from_name,omitempty"`
	FromEmail   string `json:"from_email,omitempty"`
	ReplyTo     string `json:"reply_to,omitempty"`
	Active      bool   `json:"active"`
}

// TransactionalUpdateOutput defines output for update.
type TransactionalUpdateOutput struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	Subject string `json:"subject"`
	Active  bool   `json:"active"`
	Updated bool   `json:"updated"`
}

// TransactionalStatsOutput defines output for stats.
type TransactionalStatsOutput struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	TotalSent int64   `json:"total_sent"`
	Delivered int64   `json:"delivered"`
	Opened    int64   `json:"opened"`
	Clicked   int64   `json:"clicked"`
	Bounced   int64   `json:"bounced"`
	Failed    int64   `json:"failed"`
	OpenRate  float64 `json:"open_rate"`
	ClickRate float64 `json:"click_rate"`
}

// ============================================================================
// HANDLERS
// ============================================================================

func handleTransactionalCreate(ctx context.Context, toolCtx *mcpctx.ToolContext, input TransactionalInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.Name) == "" {
		return nil, nil, mcpctx.NewValidationError("name is required", "name")
	}
	if strings.TrimSpace(input.Subject) == "" {
		return nil, nil, mcpctx.NewValidationError("subject is required", "subject")
	}
	if strings.TrimSpace(input.HTMLBody) == "" {
		return nil, nil, mcpctx.NewValidationError("html_body is required", "html_body")
	}

	slug := input.Slug
	if slug == "" {
		slug = generateSlug(input.Name)
	}

	active := true
	if input.Active != nil {
		active = *input.Active
	}

	templateID := uuid.New().String()
	template, err := toolCtx.DB().CreateTransactionalEmail(ctx, db.CreateTransactionalEmailParams{
		ID:          templateID,
		OrgID:       toolCtx.BrandID(ctx),
		Name:        input.Name,
		Slug:        slug,
		Description: sql.NullString{String: input.Description, Valid: input.Description != ""},
		Subject:     input.Subject,
		HtmlBody:    input.HTMLBody,
		PlainText:   sql.NullString{String: input.PlainText, Valid: input.PlainText != ""},
		FromName:    sql.NullString{String: input.FromName, Valid: input.FromName != ""},
		FromEmail:   sql.NullString{String: input.FromEmail, Valid: input.FromEmail != ""},
		ReplyTo:     sql.NullString{String: input.ReplyTo, Valid: input.ReplyTo != ""},
		IsActive:    sql.NullInt64{Int64: boolToInt64(active), Valid: true},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create transactional email: %w", err)
	}

	return nil, TransactionalCreateOutput{
		ID:      template.ID,
		Name:    template.Name,
		Slug:    template.Slug,
		Subject: template.Subject,
		Active:  int64ToBool(template.IsActive),
		Created: true,
	}, nil
}

func handleTransactionalList(ctx context.Context, toolCtx *mcpctx.ToolContext, input TransactionalInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	templates, err := toolCtx.DB().ListTransactionalEmails(ctx, toolCtx.BrandID(ctx))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list transactional emails: %w", err)
	}

	items := make([]TransactionalTemplateItem, 0, len(templates))
	for _, t := range templates {
		items = append(items, TransactionalTemplateItem{
			ID:        t.ID,
			Name:      t.Name,
			Slug:      t.Slug,
			Subject:   t.Subject,
			Active:    int64ToBool(t.IsActive),
			CreatedAt: t.CreatedAt.String,
		})
	}

	return nil, TransactionalListOutput{
		Templates: items,
		Total:     len(items),
	}, nil
}

func handleTransactionalGet(ctx context.Context, toolCtx *mcpctx.ToolContext, input TransactionalInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	template, err := toolCtx.DB().GetTransactionalEmail(ctx, db.GetTransactionalEmailParams{
		ID:    input.ID,
		OrgID: toolCtx.BrandID(ctx),
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("transactional email %s not found", input.ID))
	}

	return nil, TransactionalGetOutput{
		ID:          template.ID,
		Name:        template.Name,
		Slug:        template.Slug,
		Description: template.Description.String,
		Subject:     template.Subject,
		HTMLBody:    template.HtmlBody,
		PlainText:   template.PlainText.String,
		FromName:    template.FromName.String,
		FromEmail:   template.FromEmail.String,
		ReplyTo:     template.ReplyTo.String,
		Active:      int64ToBool(template.IsActive),
	}, nil
}

func handleTransactionalUpdate(ctx context.Context, toolCtx *mcpctx.ToolContext, input TransactionalInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	// Verify template exists
	existing, err := toolCtx.DB().GetTransactionalEmail(ctx, db.GetTransactionalEmailParams{
		ID:    input.ID,
		OrgID: toolCtx.BrandID(ctx),
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("transactional email %s not found", input.ID))
	}

	// Determine active value
	var isActive sql.NullInt64
	if input.Active != nil {
		isActive = sql.NullInt64{Int64: boolToInt64(*input.Active), Valid: true}
	}

	template, err := toolCtx.DB().UpdateTransactionalEmail(ctx, db.UpdateTransactionalEmailParams{
		ID:          input.ID,
		OrgID:       toolCtx.BrandID(ctx),
		Name:        input.Name,
		Slug:        input.Slug,
		Description: input.Description,
		Subject:     input.Subject,
		HtmlBody:    input.HTMLBody,
		PlainText:   input.PlainText,
		FromName:    input.FromName,
		FromEmail:   input.FromEmail,
		ReplyTo:     input.ReplyTo,
		IsActive:    isActive,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update transactional email: %w", err)
	}

	// Get final active state
	finalActive := int64ToBool(template.IsActive)
	if input.Active != nil {
		finalActive = *input.Active
	} else {
		finalActive = int64ToBool(existing.IsActive)
	}

	return nil, TransactionalUpdateOutput{
		ID:      template.ID,
		Name:    template.Name,
		Slug:    template.Slug,
		Subject: template.Subject,
		Active:  finalActive,
		Updated: true,
	}, nil
}

func handleTransactionalDelete(ctx context.Context, toolCtx *mcpctx.ToolContext, input TransactionalInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	// Verify template exists
	_, err := toolCtx.DB().GetTransactionalEmail(ctx, db.GetTransactionalEmailParams{
		ID:    input.ID,
		OrgID: toolCtx.BrandID(ctx),
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("transactional email %s not found", input.ID))
	}

	err = toolCtx.DB().DeleteTransactionalEmail(ctx, db.DeleteTransactionalEmailParams{
		ID:    input.ID,
		OrgID: toolCtx.BrandID(ctx),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to delete transactional email: %w", err)
	}

	return nil, DeleteOutput{
		Success: true,
		Message: fmt.Sprintf("Transactional email %s deleted successfully", input.ID),
	}, nil
}

func handleTransactionalStats(ctx context.Context, toolCtx *mcpctx.ToolContext, input TransactionalInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	// Verify template exists and belongs to brand
	template, err := toolCtx.DB().GetTransactionalEmail(ctx, db.GetTransactionalEmailParams{
		ID:    input.ID,
		OrgID: toolCtx.BrandID(ctx),
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("transactional email %s not found", input.ID))
	}

	// Get stats
	stats, err := toolCtx.DB().GetTransactionalStats(ctx, input.ID)
	if err != nil {
		// If no stats, return zeros
		stats = db.GetTransactionalStatsRow{}
	}

	// Calculate rates
	var openRate, clickRate float64
	sent := int64(0)
	if stats.Sent.Valid {
		sent = int64(stats.Sent.Float64)
	}
	opened := int64(0)
	if stats.Opened.Valid {
		opened = int64(stats.Opened.Float64)
	}
	clicked := int64(0)
	if stats.Clicked.Valid {
		clicked = int64(stats.Clicked.Float64)
	}
	delivered := int64(0)
	if stats.Delivered.Valid {
		delivered = int64(stats.Delivered.Float64)
	}
	bounced := int64(0)
	if stats.Bounced.Valid {
		bounced = int64(stats.Bounced.Float64)
	}
	failed := int64(0)
	if stats.Failed.Valid {
		failed = int64(stats.Failed.Float64)
	}

	if sent > 0 {
		openRate = float64(opened) / float64(sent) * 100
		clickRate = float64(clicked) / float64(sent) * 100
	}

	return nil, TransactionalStatsOutput{
		ID:        input.ID,
		Name:      template.Name,
		TotalSent: sent,
		Delivered: delivered,
		Opened:    opened,
		Clicked:   clicked,
		Bounced:   bounced,
		Failed:    failed,
		OpenRate:  openRate,
		ClickRate: clickRate,
	}, nil
}

// registerTransactionalToolToRegistry registers transactional tool to the direct-call registry.
func registerTransactionalToolToRegistry(registry *ToolRegistry, toolCtx *mcpctx.ToolContext) {
	registry.Register("transactional", func(ctx context.Context, args json.RawMessage) (interface{}, error) {
		var input TransactionalInput
		if err := json.Unmarshal(args, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		handler := transactionalHandler(toolCtx)
		_, output, err := handler(ctx, nil, input)
		return output, err
	})
}
