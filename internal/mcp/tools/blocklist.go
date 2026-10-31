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

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// blocklistActions defines valid actions for each blocklist resource.
var blocklistActions = map[string][]string{
	"suppression": {"list", "add", "delete", "bulk_add", "clear"},
	"domain":      {"list", "add", "delete", "bulk_add"},
}

// BlocklistInput defines input for the blocklist tool.
type BlocklistInput struct {
	Resource string `json:"resource" jsonschema:"required,Resource type: suppression or domain"`
	Action   string `json:"action" jsonschema:"required,Action to perform: list, add, delete, bulk_add, clear"`

	// Suppression fields
	Email  string `json:"email,omitempty" jsonschema:"Email address (suppression.add, suppression.delete)"`
	Emails string `json:"emails,omitempty" jsonschema:"Comma-separated emails (suppression.bulk_add)"`
	Reason string `json:"reason,omitempty" jsonschema:"Reason for suppression (suppression.add, domain.add)"`
	Source string `json:"source,omitempty" jsonschema:"Source of suppression: manual, bounce, complaint (suppression.add)"`

	// Domain fields
	Domain  string `json:"domain,omitempty" jsonschema:"Domain name (domain.add, domain.delete)"`
	Domains string `json:"domains,omitempty" jsonschema:"Comma-separated domains (domain.bulk_add)"`

	// Common
	ID int64 `json:"id,omitempty" jsonschema:"Item ID (for delete by ID)"`

	// Pagination
	Page     int `json:"page,omitempty" jsonschema:"Page number (default: 1)"`
	PageSize int `json:"page_size,omitempty" jsonschema:"Items per page (default: 20, max: 100)"`

	// Confirmation
	Confirm     bool `json:"confirm,omitempty" jsonschema:"Confirmation for destructive actions (suppression.clear)"`
	ConfirmLive bool `json:"confirm_live,omitempty" jsonschema:"Set to true to clear the list while the brand's API key is in live use (suppression.clear)"`
}

// RegisterBlocklistTool registers the blocklist tool.
func RegisterBlocklistTool(server *mcp.Server, toolCtx *mcpctx.ToolContext) {
	mcp.AddTool(server, &mcp.Tool{
		Name:  "blocklist",
		Title: "Email Blocklist Management",
		Description: `Manage email suppression list and blocked domains.

PREREQUISITE: You must first select a brand using brand(resource: brand, action: select).

Resources:
- suppression: Per-brand email suppression list
- domain: Per-brand blocked domain list

Actions and Required Fields:
- suppression.list: List suppressed emails (optional: page, page_size)
- suppression.add: Add email to suppression list (requires: email, optional: reason, source)
- suppression.delete: Remove email from suppression list (requires: email or id)
- suppression.bulk_add: Bulk add emails (requires: emails - comma-separated)
- suppression.clear: Clear all suppressions (requires: confirm=true)
- domain.list: List blocked domains (optional: page, page_size)
- domain.add: Block a domain (requires: domain, optional: reason)
- domain.delete: Unblock a domain (requires: domain or id)
- domain.bulk_add: Bulk block domains (requires: domains - comma-separated)

Examples:
  blocklist(resource: suppression, action: list)
  blocklist(resource: suppression, action: add, email: "spam@example.com", reason: "bounced")
  blocklist(resource: suppression, action: delete, email: "user@example.com")
  blocklist(resource: suppression, action: bulk_add, emails: "a@x.com,b@x.com,c@x.com")
  blocklist(resource: suppression, action: clear, confirm: true)
  blocklist(resource: domain, action: list)
  blocklist(resource: domain, action: add, domain: "spam.com", reason: "known spam domain")
  blocklist(resource: domain, action: bulk_add, domains: "spam1.com,spam2.com")`,
	}, blocklistHandler(toolCtx))
}

func blocklistHandler(toolCtx *mcpctx.ToolContext) func(ctx context.Context, req *mcp.CallToolRequest, input BlocklistInput) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input BlocklistInput) (*mcp.CallToolResult, any, error) {
		// Validate resource
		validActions, ok := blocklistActions[input.Resource]
		if !ok {
			return nil, nil, mcpctx.NewValidationError(
				fmt.Sprintf("invalid resource '%s', must be: suppression or domain", input.Resource),
				"resource")
		}

		// Validate action
		if !slices.Contains(validActions, input.Action) {
			return nil, nil, mcpctx.NewValidationError(
				fmt.Sprintf("invalid action '%s' for resource '%s', must be: %s",
					input.Action, input.Resource, strings.Join(validActions, ", ")),
				"action")
		}

		switch input.Resource {
		case "suppression":
			return handleSuppression(ctx, toolCtx, input)
		case "domain":
			return handleBlockedDomain(ctx, toolCtx, input)
		}
		return nil, nil, nil // unreachable
	}
}

// ============================================================================
// OUTPUT TYPES
// ============================================================================

// SuppressionItem represents an item in the suppression list.
type SuppressionItem struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	Reason    string `json:"reason,omitempty"`
	Source    string `json:"source,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

// SuppressionListOutput defines output for suppression.list.
type SuppressionListOutput struct {
	Emails   []SuppressionItem `json:"emails"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

// SuppressionAddOutput defines output for suppression.add.
type SuppressionAddOutput struct {
	ID     int64  `json:"id"`
	Email  string `json:"email"`
	Reason string `json:"reason,omitempty"`
	Source string `json:"source,omitempty"`
	Added  bool   `json:"added"`
}

// BulkAddOutput defines output for bulk_add operations.
type BulkAddOutput struct {
	AddedCount   int  `json:"added_count"`
	SkippedCount int  `json:"skipped_count"`
	Success      bool `json:"success"`
}

// ClearOutput defines output for suppression.clear.
type ClearOutput struct {
	ClearedCount int64 `json:"cleared_count"`
	Success      bool  `json:"success"`
}

// BlockedDomainItem represents a blocked domain.
type BlockedDomainItem struct {
	ID        int64  `json:"id"`
	Domain    string `json:"domain"`
	Reason    string `json:"reason,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

// DomainListOutput defines output for domain.list.
type DomainListOutput struct {
	Domains  []BlockedDomainItem `json:"domains"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}

// DomainAddOutput defines output for domain.add.
type DomainAddOutput struct {
	ID     int64  `json:"id"`
	Domain string `json:"domain"`
	Reason string `json:"reason,omitempty"`
	Added  bool   `json:"added"`
}

// ============================================================================
// SUPPRESSION HANDLERS
// ============================================================================

func handleSuppression(ctx context.Context, toolCtx *mcpctx.ToolContext, input BlocklistInput) (*mcp.CallToolResult, any, error) {
	switch input.Action {
	case "list":
		return handleSuppressionList(ctx, toolCtx, input)
	case "add":
		return handleSuppressionAdd(ctx, toolCtx, input)
	case "delete":
		return handleSuppressionDelete(ctx, toolCtx, input)
	case "bulk_add":
		return handleSuppressionBulkAdd(ctx, toolCtx, input)
	case "clear":
		return handleSuppressionClear(ctx, toolCtx, input)
	}
	return nil, nil, nil
}

func handleSuppressionList(ctx context.Context, toolCtx *mcpctx.ToolContext, input BlocklistInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	// Pagination defaults
	page := input.Page
	if page <= 0 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	// Get total count
	total, err := toolCtx.DB().CountSuppressedEmails(ctx, toolCtx.BrandID(ctx))
	if err != nil {
		total = 0
	}

	// Get list
	suppressions, err := toolCtx.DB().ListSuppressedEmails(ctx, db.ListSuppressedEmailsParams{
		OrgID:      toolCtx.BrandID(ctx),
		PageOffset: int64(offset),
		PageSize:   int64(pageSize),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list suppressed emails: %w", err)
	}

	items := make([]SuppressionItem, 0, len(suppressions))
	for _, s := range suppressions {
		items = append(items, SuppressionItem{
			ID:        s.ID,
			Email:     s.Email,
			Reason:    s.Reason.String,
			Source:    s.Source.String,
			CreatedAt: s.CreatedAt.String,
		})
	}

	return nil, SuppressionListOutput{
		Emails:   items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func handleSuppressionAdd(ctx context.Context, toolCtx *mcpctx.ToolContext, input BlocklistInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.Email) == "" {
		return nil, nil, mcpctx.NewValidationError("email is required", "email")
	}

	source := input.Source
	if source == "" {
		source = "manual"
	}

	suppression, err := toolCtx.DB().AddToSuppressionList(ctx, db.AddToSuppressionListParams{
		OrgID:  toolCtx.BrandID(ctx),
		Email:  input.Email,
		Reason: sql.NullString{String: input.Reason, Valid: input.Reason != ""},
		Source: sql.NullString{String: source, Valid: true},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to add to suppression list: %w", err)
	}

	return nil, SuppressionAddOutput{
		ID:     suppression.ID,
		Email:  suppression.Email,
		Reason: suppression.Reason.String,
		Source: suppression.Source.String,
		Added:  true,
	}, nil
}

func handleSuppressionDelete(ctx context.Context, toolCtx *mcpctx.ToolContext, input BlocklistInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if input.ID > 0 {
		// Delete by ID
		err := toolCtx.DB().DeleteSuppressionByID(ctx, db.DeleteSuppressionByIDParams{
			ID:    input.ID,
			OrgID: toolCtx.BrandID(ctx),
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to delete suppression: %w", err)
		}
		return nil, DeleteOutput{
			Success: true,
			Message: fmt.Sprintf("Suppression %d deleted successfully", input.ID),
		}, nil
	}

	if strings.TrimSpace(input.Email) == "" {
		return nil, nil, mcpctx.NewValidationError("email or id is required", "email")
	}

	// Delete by email
	err := toolCtx.DB().DeleteFromSuppressionList(ctx, db.DeleteFromSuppressionListParams{
		OrgID: toolCtx.BrandID(ctx),
		Email: input.Email,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to delete from suppression list: %w", err)
	}

	return nil, DeleteOutput{
		Success: true,
		Message: fmt.Sprintf("Email %s removed from suppression list", input.Email),
	}, nil
}

func handleSuppressionBulkAdd(ctx context.Context, toolCtx *mcpctx.ToolContext, input BlocklistInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.Emails) == "" {
		return nil, nil, mcpctx.NewValidationError("emails is required (comma-separated)", "emails")
	}

	emails := strings.Split(input.Emails, ",")
	addedCount := 0
	skippedCount := 0

	source := input.Source
	if source == "" {
		source = "bulk_import"
	}

	for _, email := range emails {
		email = strings.TrimSpace(email)
		if email == "" {
			continue
		}

		// Check if already exists
		existing, err := toolCtx.DB().GetSuppressedEmail(ctx, db.GetSuppressedEmailParams{
			OrgID: toolCtx.BrandID(ctx),
			Email: email,
		})
		if err == nil && existing.ID > 0 {
			skippedCount++
			continue
		}

		_, err = toolCtx.DB().AddToSuppressionList(ctx, db.AddToSuppressionListParams{
			OrgID:  toolCtx.BrandID(ctx),
			Email:  email,
			Reason: sql.NullString{String: input.Reason, Valid: input.Reason != ""},
			Source: sql.NullString{String: source, Valid: true},
		})
		if err != nil {
			skippedCount++
		} else {
			addedCount++
		}
	}

	return nil, BulkAddOutput{
		AddedCount:   addedCount,
		SkippedCount: skippedCount,
		Success:      true,
	}, nil
}

func handleSuppressionClear(ctx context.Context, toolCtx *mcpctx.ToolContext, input BlocklistInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if !input.Confirm {
		return nil, nil, mcpctx.NewValidationError("confirm=true is required to clear suppression list", "confirm")
	}

	// Get count before clearing
	count, _ := toolCtx.DB().CountSuppressedEmails(ctx, toolCtx.BrandID(ctx))

	err := toolCtx.DB().ClearSuppressionList(ctx, toolCtx.BrandID(ctx))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to clear suppression list: %w", err)
	}

	return nil, ClearOutput{
		ClearedCount: count,
		Success:      true,
	}, nil
}

// ============================================================================
// DOMAIN HANDLERS
// ============================================================================

func handleBlockedDomain(ctx context.Context, toolCtx *mcpctx.ToolContext, input BlocklistInput) (*mcp.CallToolResult, any, error) {
	switch input.Action {
	case "list":
		return handleDomainList(ctx, toolCtx, input)
	case "add":
		return handleDomainAdd(ctx, toolCtx, input)
	case "delete":
		return handleDomainDelete(ctx, toolCtx, input)
	case "bulk_add":
		return handleDomainBulkAdd(ctx, toolCtx, input)
	}
	return nil, nil, nil
}

func handleDomainList(ctx context.Context, toolCtx *mcpctx.ToolContext, input BlocklistInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	// Pagination defaults
	page := input.Page
	if page <= 0 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	// Get total count
	total, err := toolCtx.DB().CountBlockedDomains(ctx, toolCtx.BrandID(ctx))
	if err != nil {
		total = 0
	}

	// Get list
	domains, err := toolCtx.DB().ListBlockedDomains(ctx, db.ListBlockedDomainsParams{
		OrgID:      toolCtx.BrandID(ctx),
		PageOffset: int64(offset),
		PageSize:   int64(pageSize),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list blocked domains: %w", err)
	}

	items := make([]BlockedDomainItem, 0, len(domains))
	for _, d := range domains {
		items = append(items, BlockedDomainItem{
			ID:        d.ID,
			Domain:    d.Domain,
			Reason:    d.Reason.String,
			CreatedAt: d.CreatedAt.String,
		})
	}

	return nil, DomainListOutput{
		Domains:  items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func handleDomainAdd(ctx context.Context, toolCtx *mcpctx.ToolContext, input BlocklistInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.Domain) == "" {
		return nil, nil, mcpctx.NewValidationError("domain is required", "domain")
	}

	domain, err := toolCtx.DB().CreateBlockedDomain(ctx, db.CreateBlockedDomainParams{
		OrgID:  toolCtx.BrandID(ctx),
		Domain: input.Domain,
		Reason: sql.NullString{String: input.Reason, Valid: input.Reason != ""},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to block domain: %w", err)
	}

	return nil, DomainAddOutput{
		ID:     domain.ID,
		Domain: domain.Domain,
		Reason: domain.Reason.String,
		Added:  true,
	}, nil
}

func handleDomainDelete(ctx context.Context, toolCtx *mcpctx.ToolContext, input BlocklistInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if input.ID > 0 {
		// Delete by ID
		err := toolCtx.DB().DeleteBlockedDomainByID(ctx, db.DeleteBlockedDomainByIDParams{
			ID:    input.ID,
			OrgID: toolCtx.BrandID(ctx),
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to delete blocked domain: %w", err)
		}
		return nil, DeleteOutput{
			Success: true,
			Message: fmt.Sprintf("Blocked domain %d deleted successfully", input.ID),
		}, nil
	}

	if strings.TrimSpace(input.Domain) == "" {
		return nil, nil, mcpctx.NewValidationError("domain or id is required", "domain")
	}

	// Delete by domain name
	err := toolCtx.DB().DeleteBlockedDomain(ctx, db.DeleteBlockedDomainParams{
		OrgID:  toolCtx.BrandID(ctx),
		Domain: input.Domain,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unblock domain: %w", err)
	}

	return nil, DeleteOutput{
		Success: true,
		Message: fmt.Sprintf("Domain %s unblocked successfully", input.Domain),
	}, nil
}

func handleDomainBulkAdd(ctx context.Context, toolCtx *mcpctx.ToolContext, input BlocklistInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.Domains) == "" {
		return nil, nil, mcpctx.NewValidationError("domains is required (comma-separated)", "domains")
	}

	domains := strings.Split(input.Domains, ",")
	addedCount := 0
	skippedCount := 0

	for _, domain := range domains {
		domain = strings.TrimSpace(domain)
		if domain == "" {
			continue
		}

		err := toolCtx.DB().BulkInsertBlockedDomains(ctx, db.BulkInsertBlockedDomainsParams{
			OrgID:  toolCtx.BrandID(ctx),
			Domain: domain,
			Reason: sql.NullString{String: input.Reason, Valid: input.Reason != ""},
		})
		if err != nil {
			skippedCount++
		} else {
			addedCount++
		}
	}

	return nil, BulkAddOutput{
		AddedCount:   addedCount,
		SkippedCount: skippedCount,
		Success:      true,
	}, nil
}

// registerBlocklistToolToRegistry registers blocklist tool to the direct-call registry.
func registerBlocklistToolToRegistry(registry *ToolRegistry, toolCtx *mcpctx.ToolContext) {
	registry.Register("blocklist", func(ctx context.Context, args json.RawMessage) (interface{}, error) {
		var input BlocklistInput
		if err := json.Unmarshal(args, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		handler := blocklistHandler(toolCtx)
		_, output, err := handler(ctx, nil, input)
		return output, err
	})
}
