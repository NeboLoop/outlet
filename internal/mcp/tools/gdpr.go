package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/mcp/mcpctx"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/services/gdpr"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// gdprActions defines valid actions for the GDPR tool.
var gdprActions = []string{"lookup", "export", "delete", "get_consent", "update_consent"}

// GDPRInput defines input for the GDPR tool.
type GDPRInput struct {
	Action string `json:"action" jsonschema:"required,Action to perform: lookup, export, delete, get_consent, update_consent"`

	// Lookup by email
	Email string `json:"email,omitempty" jsonschema:"Email address to lookup (lookup)"`

	// By contact ID
	ContactID string `json:"contact_id,omitempty" jsonschema:"Contact ID (export, delete, get_consent, update_consent)"`

	// Consent update fields
	GDPRConsent *bool `json:"gdpr_consent,omitempty" jsonschema:"GDPR consent status (update_consent)"`

	// Confirmation
	Confirm      bool `json:"confirm,omitempty" jsonschema:"Confirmation for delete action (delete)"`
	ProviderLogs bool `json:"provider_logs,omitempty" jsonschema:"Also ask the brand's email providers to delete what they keep about the address (delete). Amazon SES: the address comes off the account's suppression list, the one per-recipient record its API can delete. SMTP relays and mailboxes have no API for it, which the result says"`
	ConfirmLive  bool `json:"confirm_live,omitempty" jsonschema:"Set to true to delete while the brand's API key is in live use (delete)"`
}

// RegisterGDPRTool registers the GDPR tool.
func RegisterGDPRTool(server *mcp.Server, toolCtx *mcpctx.ToolContext) {
	mcp.AddTool(server, &mcp.Tool{
		Name:  "gdpr",
		Title: "GDPR Compliance",
		Description: `GDPR compliance tools for managing contact data and consent.

PREREQUISITE: You must first select a brand using brand(resource: brand, action: select).

Actions and Required Fields:
- lookup: Look up a contact by email (requires: email)
- export: Export all data for a contact (requires: contact_id)
- delete: Delete all data for a contact (requires: contact_id, confirm=true; optional: provider_logs). Recorded in the GDPR audit log
- get_consent: Get consent status for a contact (requires: contact_id)
- update_consent: Update consent status (requires: contact_id, optional: gdpr_consent)

Examples:
  gdpr(action: lookup, email: "user@example.com")
  gdpr(action: export, contact_id: "uuid")
  gdpr(action: get_consent, contact_id: "uuid")
  gdpr(action: update_consent, contact_id: "uuid", gdpr_consent: true)
  gdpr(action: delete, contact_id: "uuid", confirm: true)`,
	}, gdprHandler(toolCtx))
}

func gdprHandler(toolCtx *mcpctx.ToolContext) func(ctx context.Context, req *mcp.CallToolRequest, input GDPRInput) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input GDPRInput) (*mcp.CallToolResult, any, error) {
		// Validate action
		if !slices.Contains(gdprActions, input.Action) {
			return nil, nil, mcpctx.NewValidationError(
				fmt.Sprintf("invalid action '%s', must be: %s", input.Action, strings.Join(gdprActions, ", ")),
				"action")
		}

		switch input.Action {
		case "lookup":
			return handleGDPRLookup(ctx, toolCtx, input)
		case "export":
			return handleGDPRExport(ctx, toolCtx, input)
		case "delete":
			return handleGDPRDelete(ctx, toolCtx, input)
		case "get_consent":
			return handleGDPRGetConsent(ctx, toolCtx, input)
		case "update_consent":
			return handleGDPRUpdateConsent(ctx, toolCtx, input)
		}
		return nil, nil, nil // unreachable
	}
}

// ============================================================================
// OUTPUT TYPES
// ============================================================================

// GDPRLookupOutput defines output for lookup.
type GDPRLookupOutput struct {
	ContactID string `json:"contact_id,omitempty"`
	Email     string `json:"email,omitempty"`
	Name      string `json:"name,omitempty"`
	Status    string `json:"status,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	Found     bool   `json:"found"`
}

// GDPRConsentOutput defines output for get_consent and update_consent.
type GDPRConsentOutput struct {
	ContactID        string `json:"contact_id"`
	Email            string `json:"email,omitempty"`
	GDPRConsent      bool   `json:"gdpr_consent"`
	ConsentTimestamp string `json:"consent_timestamp,omitempty"`
	Updated          bool   `json:"updated,omitempty"`
}

// ============================================================================
// HANDLERS
// ============================================================================

func handleGDPRLookup(ctx context.Context, toolCtx *mcpctx.ToolContext, input GDPRInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.Email) == "" {
		return nil, nil, mcpctx.NewValidationError("email is required", "email")
	}

	contact, err := toolCtx.DB().GetContactByOrgAndEmail(ctx, db.GetContactByOrgAndEmailParams{
		OrgID: sql.NullString{String: toolCtx.BrandID(ctx), Valid: true},
		Email: input.Email,
	})
	if err != nil {
		return nil, GDPRLookupOutput{
			Found: false,
		}, nil
	}

	return nil, GDPRLookupOutput{
		ContactID: contact.ID,
		Email:     contact.Email,
		Name:      contact.Name,
		Status:    contact.Status.String,
		CreatedAt: contact.CreatedAt.String,
		Found:     true,
	}, nil
}

func handleGDPRExport(ctx context.Context, toolCtx *mcpctx.ToolContext, input GDPRInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ContactID) == "" {
		return nil, nil, mcpctx.NewValidationError("contact_id is required", "contact_id")
	}

	export, err := gdpr.Export(ctx, toolCtx.DB(), toolCtx.BrandID(ctx), input.ContactID)
	if errors.Is(err, gdpr.ErrNotFound) {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("contact %s not found", input.ContactID))
	}
	if err != nil {
		return nil, nil, err
	}
	return nil, export, nil
}

func handleGDPRDelete(ctx context.Context, toolCtx *mcpctx.ToolContext, input GDPRInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ContactID) == "" {
		return nil, nil, mcpctx.NewValidationError("contact_id is required", "contact_id")
	}

	if !input.Confirm {
		return nil, nil, mcpctx.NewValidationError("confirm=true is required to delete contact data", "confirm")
	}

	requestedBy, requestedByEmail := "api-key", ""
	if u := toolCtx.User(); u != nil {
		requestedBy, requestedByEmail = u.ID, u.Email
	}
	deleted, err := gdpr.Delete(ctx, toolCtx.DB(), toolCtx.Svc().EmailService, gdpr.DeleteRequest{
		OrgID:            toolCtx.BrandID(ctx),
		ContactID:        input.ContactID,
		ProviderLogs:     input.ProviderLogs,
		RequestedBy:      requestedBy,
		RequestedByEmail: requestedByEmail,
		Via:              "mcp",
	})
	if errors.Is(err, gdpr.ErrNotFound) {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("contact %s not found", input.ContactID))
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to delete contact: %w", err)
	}

	return nil, GDPRDeleteOutput{
		Success:   true,
		Message:   fmt.Sprintf("Contact %s and all associated data deleted successfully", input.ContactID),
		AuditID:   deleted.AuditID,
		Providers: deleted.Providers,
	}, nil
}

// GDPRDeleteOutput defines output for gdpr delete.
type GDPRDeleteOutput struct {
	Success   bool                    `json:"success"`
	Message   string                  `json:"message"`
	AuditID   string                  `json:"audit_id"` // the deletion's row in the GDPR audit log
	Providers []email.ProviderErasure `json:"providers,omitempty"`
}

func handleGDPRGetConsent(ctx context.Context, toolCtx *mcpctx.ToolContext, input GDPRInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ContactID) == "" {
		return nil, nil, mcpctx.NewValidationError("contact_id is required", "contact_id")
	}

	// Get contact
	contact, err := toolCtx.DB().GetContactByOrgID(ctx, db.GetContactByOrgIDParams{
		ID:    input.ContactID,
		OrgID: sql.NullString{String: toolCtx.BrandID(ctx), Valid: true},
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("contact %s not found", input.ContactID))
	}

	return nil, GDPRConsentOutput{
		ContactID:        contact.ID,
		Email:            contact.Email,
		GDPRConsent:      int64ToBool(contact.GdprConsent),
		ConsentTimestamp: contact.GdprConsentAt.String,
	}, nil
}

func handleGDPRUpdateConsent(ctx context.Context, toolCtx *mcpctx.ToolContext, input GDPRInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ContactID) == "" {
		return nil, nil, mcpctx.NewValidationError("contact_id is required", "contact_id")
	}

	// Get current contact
	contact, err := toolCtx.DB().GetContactByOrgID(ctx, db.GetContactByOrgIDParams{
		ID:    input.ContactID,
		OrgID: sql.NullString{String: toolCtx.BrandID(ctx), Valid: true},
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("contact %s not found", input.ContactID))
	}

	// Determine value
	gdprConsent := int64ToBool(contact.GdprConsent)
	if input.GDPRConsent != nil {
		gdprConsent = *input.GDPRConsent
	}

	// Update consent
	err = toolCtx.DB().UpdateContactGDPR(ctx, db.UpdateContactGDPRParams{
		ID:      input.ContactID,
		Consent: sql.NullInt64{Int64: boolToInt64(gdprConsent), Valid: true},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update consent: %w", err)
	}

	return nil, GDPRConsentOutput{
		ContactID:   contact.ID,
		Email:       contact.Email,
		GDPRConsent: gdprConsent,
		Updated:     true,
	}, nil
}

// registerGDPRToolToRegistry registers GDPR tool to the direct-call registry.
func registerGDPRToolToRegistry(registry *ToolRegistry, toolCtx *mcpctx.ToolContext) {
	registry.Register("gdpr", func(ctx context.Context, args json.RawMessage) (interface{}, error) {
		var input GDPRInput
		if err := json.Unmarshal(args, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		handler := gdprHandler(toolCtx)
		_, output, err := handler(ctx, nil, input)
		return output, err
	})
}
