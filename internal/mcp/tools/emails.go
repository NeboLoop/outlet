package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/mcp/mcpctx"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func int64ToBool(i sql.NullInt64) bool {
	return i.Valid && i.Int64 == 1
}

func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	slug = result.String()
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")
	return slug
}

// sameBrand refuses a record of another brand as not found. The gate
// checked the caller's access to this call's brand; every record ID the
// caller hands a tool must belong to that brand too.
func sameBrand(ctx context.Context, toolCtx *mcpctx.ToolContext, rec access.Record, id, what string) error {
	err := access.SameBrand(ctx, toolCtx.DB(), rec, id, toolCtx.BrandID(ctx))
	if errors.Is(err, access.ErrNotFound) {
		return mcpctx.NewNotFoundError(fmt.Sprintf("%s %s not found", what, id))
	}
	return err
}

// emailActions defines valid actions for each email resource.
var emailActions = map[string][]string{
	"list":       {"create", "list", "get", "update", "delete", "stats", "subscribers", "subscribe", "unsubscribe"},
	"sequence":   {"create", "list", "get", "update", "delete", "stats"},
	"template":   {"create", "list", "get", "update", "delete"},
	"enrollment": {"enroll", "unenroll", "pause", "resume", "list"},
	"entry_rule": {"create", "list", "update", "delete"},
	"queue":      {"list", "cancel"},
}

// EmailInput defines input for the unified email tool.
type EmailInput struct {
	Resource string `json:"resource" jsonschema:"required,Resource type: list, sequence, template, enrollment, entry_rule, or queue"`
	Action   string `json:"action" jsonschema:"required,Action to perform"`

	// Common
	ID string `json:"id,omitempty" jsonschema:"Resource ID (for get, update, delete)"`

	// List fields
	Name                string `json:"name,omitempty" jsonschema:"Name for list or sequence (create/update)"`
	Description         string `json:"description,omitempty" jsonschema:"List description"`
	DoubleOptin         *bool  `json:"double_optin,omitempty" jsonschema:"Require email confirmation (list)"`
	ConfirmationSubject string `json:"confirmation_subject,omitempty" jsonschema:"Custom subject for double opt-in confirmation email (list.update)"`
	ConfirmationBody    string `json:"confirmation_body,omitempty" jsonschema:"Custom HTML body for double opt-in confirmation email. Use {{confirm_url}} for the confirmation link (list.update)"`

	// Sequence fields
	ListID       string `json:"list_id,omitempty" jsonschema:"List ID to attach sequence to (sequence.create) or filter by (sequence.list)"`
	TriggerEvent string `json:"trigger_event,omitempty" jsonschema:"Event that starts sequence: on_subscribe, on_purchase, manual"`
	SequenceType string `json:"sequence_type,omitempty" jsonschema:"Type: lifecycle, promotional, transactional"`
	Active       *bool  `json:"active,omitempty" jsonschema:"Whether resource is active"`

	// Template fields
	SequenceID string `json:"sequence_id,omitempty" jsonschema:"Sequence ID (template.create, template.list, enrollment, entry_rule)"`
	Subject    string `json:"subject,omitempty" jsonschema:"Email subject line (template)"`
	HTMLBody   string `json:"html_body,omitempty" jsonschema:"HTML content of the email (template)"`
	PlainText  string `json:"plain_text,omitempty" jsonschema:"Plain text version (template)"`
	Position   int    `json:"position,omitempty" jsonschema:"Position in sequence (1-based)"`
	DelayHours int    `json:"delay_hours,omitempty" jsonschema:"Hours after previous email (or trigger for position 1)"`

	// Pagination fields for list.subscribers
	Page     int    `json:"page,omitempty" jsonschema:"Page number (default: 1)"`
	PageSize int    `json:"page_size,omitempty" jsonschema:"Items per page (default: 20, max: 100)"`
	Status   string `json:"status,omitempty" jsonschema:"Filter by status: active, pending, unsubscribed (list.subscribers, queue.list)"`

	// Contact/Subscribe fields
	Email     string `json:"email,omitempty" jsonschema:"Email address (list.subscribe, list.unsubscribe)"`
	ContactID string `json:"contact_id,omitempty" jsonschema:"Contact ID (enrollment operations, queue.list filter)"`

	// Entry rule fields
	TriggerType string `json:"trigger_type,omitempty" jsonschema:"Trigger type: list_subscribe, sequence_complete, tag_added (entry_rule.create)"`
	SourceID    string `json:"source_id,omitempty" jsonschema:"Source ID (list ID or sequence ID) for the trigger (entry_rule.create)"`
	Priority    int    `json:"priority,omitempty" jsonschema:"Rule priority (higher runs first, default: 10)"`
}

// RegisterEmailTool registers the unified email tool.
func RegisterEmailTool(server *mcp.Server, toolCtx *mcpctx.ToolContext) {
	mcp.AddTool(server, &mcp.Tool{
		Name:  "email",
		Title: "Email Management",
		Description: `Manage email lists, sequences, templates, enrollments, entry rules, and email queue.

PREREQUISITE: You must first select a brand using brand(resource: brand, action: select).

Resources:
- list: Email lists for grouping subscribers
- sequence: Automated drip campaigns attached to lists
- template: Individual emails within a sequence
- enrollment: Manage contact sequence enrollments
- entry_rule: Automatic enrollment rules for sequences
- queue: Pending/scheduled emails

Actions and Required Fields:

LIST RESOURCE:
- list.create: Create an email list (requires: name)
- list.list: List all email lists
- list.get: Get an email list (requires: id)
- list.update: Update an email list (requires: id, optional: name, description, double_optin, confirmation_subject, confirmation_body)
- list.delete: Delete an email list (requires: id)
- list.stats: Get list statistics (requires: id)
- list.subscribers: List subscribers (requires: id, optional: page, page_size, status)
- list.subscribe: Subscribe contact to list (requires: id, email)
- list.unsubscribe: Unsubscribe contact from list (requires: id, email)

SEQUENCE RESOURCE:
- sequence.create: Create a sequence (requires: name, list_id)
- sequence.list: List sequences (optional: list_id filter)
- sequence.get: Get a sequence with its emails (requires: id)
- sequence.update: Update a sequence (requires: id)
- sequence.delete: Delete a sequence (requires: id)
- sequence.stats: Get sequence statistics (requires: id)

TEMPLATE RESOURCE:
- template.create: Add email to sequence (requires: sequence_id, subject, html_body)
- template.list: List emails in a sequence (requires: sequence_id)
- template.get: Get an email with full html_body (requires: id)
- template.update: Update an email (requires: id)
- template.delete: Delete an email (requires: id)

ENROLLMENT RESOURCE:
- enrollment.enroll: Enroll contact in sequence (requires: sequence_id, contact_id)
- enrollment.unenroll: Remove contact from sequence (requires: sequence_id, contact_id)
- enrollment.pause: Pause contact's sequence (requires: sequence_id, contact_id)
- enrollment.resume: Resume contact's sequence (requires: sequence_id, contact_id)
- enrollment.list: List contact's enrollments (requires: contact_id)

ENTRY RULE RESOURCE:
- entry_rule.create: Create auto-enrollment rule (requires: sequence_id, trigger_type, source_id)
- entry_rule.list: List rules for sequence (requires: sequence_id)
- entry_rule.update: Update rule (requires: id)
- entry_rule.delete: Delete rule (requires: id)

QUEUE RESOURCE:
- queue.list: List pending emails (optional: status, contact_id)
- queue.cancel: Cancel pending email (requires: id)

Examples:
  email(resource: list, action: create, name: "Newsletter")
  email(resource: list, action: subscribe, id: "1", email: "user@example.com")
  email(resource: sequence, action: create, name: "Welcome Series", list_id: "1")
  email(resource: enrollment, action: enroll, sequence_id: "uuid", contact_id: "uuid")
  email(resource: enrollment, action: list, contact_id: "uuid")
  email(resource: entry_rule, action: create, sequence_id: "uuid", trigger_type: "list_subscribe", source_id: "1")
  email(resource: queue, action: list, status: "pending")`,
	}, emailHandler(toolCtx))
}

func emailHandler(toolCtx *mcpctx.ToolContext) func(ctx context.Context, req *mcp.CallToolRequest, input EmailInput) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input EmailInput) (*mcp.CallToolResult, any, error) {
		// Validate resource
		validActions, ok := emailActions[input.Resource]
		if !ok {
			return nil, nil, mcpctx.NewValidationError(
				fmt.Sprintf("invalid resource '%s', must be: list, sequence, template, enrollment, entry_rule, or queue", input.Resource),
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
		case "list":
			return handleList(ctx, toolCtx, input)
		case "sequence":
			return handleSequence(ctx, toolCtx, input)
		case "template":
			return handleTemplate(ctx, toolCtx, input)
		case "enrollment":
			return handleEnrollment(ctx, toolCtx, input)
		case "entry_rule":
			return handleEntryRule(ctx, toolCtx, input)
		case "queue":
			return handleQueue(ctx, toolCtx, input)
		}
		return nil, nil, nil // unreachable
	}
}

// ============================================================================
// LIST HANDLERS
// ============================================================================

// ListItem represents a single list in the list.
type ListItem struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Slug            string `json:"slug"`
	Description     string `json:"description,omitempty"`
	DoubleOptin     bool   `json:"double_optin"`
	SubscriberCount int64  `json:"subscriber_count"`
}

// ListListOutput defines output for list.list.
type ListListOutput struct {
	Lists []ListItem `json:"lists"`
	Total int        `json:"total"`
}

// ListCreateOutput defines output for list.create.
type ListCreateOutput struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
	DoubleOptin bool   `json:"double_optin"`
	Created     bool   `json:"created"`
}

// ListGetOutput defines output for list.get.
type ListGetOutput struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Slug            string `json:"slug"`
	Description     string `json:"description,omitempty"`
	DoubleOptin     bool   `json:"double_optin"`
	SubscriberCount int64  `json:"subscriber_count"`
}

// ListUpdateOutput defines output for list.update.
type ListUpdateOutput struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Slug                string `json:"slug"`
	Description         string `json:"description,omitempty"`
	DoubleOptin         bool   `json:"double_optin"`
	ConfirmationSubject string `json:"confirmation_subject,omitempty"`
	ConfirmationBody    string `json:"confirmation_body,omitempty"`
	Updated             bool   `json:"updated"`
}

// DeleteOutput defines output for delete actions.
type DeleteOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ListStatsOutput defines output for list.stats.
type ListStatsOutput struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	TotalSubscribers int64  `json:"total_subscribers"`
	ActiveCount      int64  `json:"active_count"`
	PendingCount     int64  `json:"pending_count"`
	UnsubscribedCount int64 `json:"unsubscribed_count"`
}

// ListSubscriberItem represents a subscriber in the list.
type ListSubscriberItem struct {
	ID           string `json:"id"`
	ContactID    string `json:"contact_id"`
	Email        string `json:"email"`
	Name         string `json:"name,omitempty"`
	Status       string `json:"status"`
	SubscribedAt string `json:"subscribed_at,omitempty"`
	VerifiedAt   string `json:"verified_at,omitempty"`
}

// ListSubscribersOutput defines output for list.subscribers.
type ListSubscribersOutput struct {
	ListID      string               `json:"list_id"`
	ListName    string               `json:"list_name"`
	Subscribers []ListSubscriberItem `json:"subscribers"`
	Total       int64                `json:"total"`
	Page        int                  `json:"page"`
	PageSize    int                  `json:"page_size"`
}

func handleList(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	switch input.Action {
	case "create":
		return handleListCreate(ctx, toolCtx, input)
	case "list":
		return handleListList(ctx, toolCtx, input)
	case "get":
		return handleListGet(ctx, toolCtx, input)
	case "update":
		return handleListUpdate(ctx, toolCtx, input)
	case "delete":
		return handleListDelete(ctx, toolCtx, input)
	case "stats":
		return handleListStats(ctx, toolCtx, input)
	case "subscribers":
		return handleListSubscribers(ctx, toolCtx, input)
	case "subscribe":
		return handleListSubscribe(ctx, toolCtx, input)
	case "unsubscribe":
		return handleListUnsubscribe(ctx, toolCtx, input)
	}
	return nil, nil, nil
}

func handleListCreate(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.Name) == "" {
		return nil, nil, mcpctx.NewValidationError("name is required", "name")
	}

	slug := generateSlug(input.Name)
	doubleOptin := false
	if input.DoubleOptin != nil {
		doubleOptin = *input.DoubleOptin
	}

	list, err := toolCtx.DB().CreateEmailList(ctx, db.CreateEmailListParams{
		PublicID:    utils.GeneratePublicID(),
		OrgID:       toolCtx.BrandID(ctx),
		Name:        input.Name,
		Slug:        slug,
		Description: sql.NullString{String: input.Description, Valid: input.Description != ""},
		DoubleOptin: sql.NullInt64{Int64: boolToInt64(doubleOptin), Valid: true},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create list: %w", err)
	}

	return nil, ListCreateOutput{
		ID:          fmt.Sprintf("%d", list.ID),
		Name:        list.Name,
		Slug:        list.Slug,
		Description: list.Description.String,
		DoubleOptin: int64ToBool(list.DoubleOptin),
		Created:     true,
	}, nil
}

func handleListList(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	lists, err := toolCtx.DB().ListEmailLists(ctx, toolCtx.BrandID(ctx))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list lists: %w", err)
	}

	items := make([]ListItem, 0, len(lists))
	for _, list := range lists {
		count, err := toolCtx.DB().CountListSubscribers(ctx, db.CountListSubscribersParams{
			ListID:       list.ID,
			FilterStatus: "active",
		})
		if err != nil {
			count = 0
		}

		items = append(items, ListItem{
			ID:              fmt.Sprintf("%d", list.ID),
			Name:            list.Name,
			Slug:            list.Slug,
			Description:     list.Description.String,
			DoubleOptin:     int64ToBool(list.DoubleOptin),
			SubscriberCount: count,
		})
	}

	return nil, ListListOutput{
		Lists: items,
		Total: len(items),
	}, nil
}

func handleListGet(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	listID, err := strconv.ParseInt(input.ID, 10, 64)
	if err != nil {
		return nil, nil, mcpctx.NewValidationError("id must be a valid ID", "id")
	}

	list, err := toolCtx.DB().GetEmailList(ctx, listID)
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("list %s not found", input.ID))
	}

	if list.OrgID != toolCtx.BrandID(ctx) {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("list %s not found", input.ID))
	}

	count, err := toolCtx.DB().CountListSubscribers(ctx, db.CountListSubscribersParams{
		ListID:       list.ID,
		FilterStatus: "active",
	})
	if err != nil {
		count = 0
	}

	return nil, ListGetOutput{
		ID:              fmt.Sprintf("%d", list.ID),
		Name:            list.Name,
		Slug:            list.Slug,
		Description:     list.Description.String,
		DoubleOptin:     int64ToBool(list.DoubleOptin),
		SubscriberCount: count,
	}, nil
}

func handleListUpdate(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	listID, err := strconv.ParseInt(input.ID, 10, 64)
	if err != nil {
		return nil, nil, mcpctx.NewValidationError("id must be a valid ID", "id")
	}

	list, err := toolCtx.DB().GetEmailList(ctx, listID)
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("list %s not found", input.ID))
	}

	if list.OrgID != toolCtx.BrandID(ctx) {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("list %s not found", input.ID))
	}

	// Apply updates
	name := list.Name
	if input.Name != "" {
		name = input.Name
	}
	description := list.Description.String
	if input.Description != "" {
		description = input.Description
	}
	doubleOptin := int64ToBool(list.DoubleOptin)
	if input.DoubleOptin != nil {
		doubleOptin = *input.DoubleOptin
	}

	confirmSubject := list.ConfirmationEmailSubject
	if input.ConfirmationSubject != "" {
		confirmSubject = sql.NullString{String: input.ConfirmationSubject, Valid: true}
	}
	confirmBody := list.ConfirmationEmailBody
	if input.ConfirmationBody != "" {
		confirmBody = sql.NullString{String: input.ConfirmationBody, Valid: true}
	}

	updatedList, err := toolCtx.DB().UpdateEmailList(ctx, db.UpdateEmailListParams{
		ID:                  listID,
		Name:                name,
		Description:         sql.NullString{String: description, Valid: description != ""},
		DoubleOptin:         sql.NullInt64{Int64: boolToInt64(doubleOptin), Valid: true},
		ConfirmationSubject: confirmSubject,
		ConfirmationBody:    confirmBody,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update list: %w", err)
	}

	return nil, ListUpdateOutput{
		ID:                  input.ID,
		Name:                updatedList.Name,
		Slug:                updatedList.Slug,
		Description:         description,
		DoubleOptin:         doubleOptin,
		ConfirmationSubject: updatedList.ConfirmationEmailSubject.String,
		ConfirmationBody:    updatedList.ConfirmationEmailBody.String,
		Updated:             true,
	}, nil
}

func handleListDelete(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	listID, err := strconv.ParseInt(input.ID, 10, 64)
	if err != nil {
		return nil, nil, mcpctx.NewValidationError("id must be a valid ID", "id")
	}

	list, err := toolCtx.DB().GetEmailList(ctx, listID)
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("list %s not found", input.ID))
	}

	if list.OrgID != toolCtx.BrandID(ctx) {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("list %s not found", input.ID))
	}

	err = toolCtx.DB().DeleteEmailList(ctx, listID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to delete list: %w", err)
	}

	return nil, DeleteOutput{
		Success: true,
		Message: fmt.Sprintf("List %s deleted successfully", input.ID),
	}, nil
}

func handleListStats(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	listID, err := strconv.ParseInt(input.ID, 10, 64)
	if err != nil {
		return nil, nil, mcpctx.NewValidationError("id must be a valid ID", "id")
	}

	list, err := toolCtx.DB().GetEmailList(ctx, listID)
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("list %s not found", input.ID))
	}

	if list.OrgID != toolCtx.BrandID(ctx) {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("list %s not found", input.ID))
	}

	// Get counts by status
	totalCount, _ := toolCtx.DB().CountListSubscribers(ctx, db.CountListSubscribersParams{
		ListID:       listID,
		FilterStatus: nil,
	})
	activeCount, _ := toolCtx.DB().CountListSubscribers(ctx, db.CountListSubscribersParams{
		ListID:       listID,
		FilterStatus: "active",
	})
	pendingCount, _ := toolCtx.DB().CountListSubscribers(ctx, db.CountListSubscribersParams{
		ListID:       listID,
		FilterStatus: "pending",
	})
	unsubscribedCount, _ := toolCtx.DB().CountListSubscribers(ctx, db.CountListSubscribersParams{
		ListID:       listID,
		FilterStatus: "unsubscribed",
	})

	return nil, ListStatsOutput{
		ID:                input.ID,
		Name:              list.Name,
		TotalSubscribers:  totalCount,
		ActiveCount:       activeCount,
		PendingCount:      pendingCount,
		UnsubscribedCount: unsubscribedCount,
	}, nil
}

func handleListSubscribers(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	listID, err := strconv.ParseInt(input.ID, 10, 64)
	if err != nil {
		return nil, nil, mcpctx.NewValidationError("id must be a valid ID", "id")
	}

	list, err := toolCtx.DB().GetEmailList(ctx, listID)
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("list %s not found", input.ID))
	}

	if list.OrgID != toolCtx.BrandID(ctx) {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("list %s not found", input.ID))
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

	// Get total count for pagination
	var filterStatus interface{}
	if input.Status != "" {
		filterStatus = input.Status
	}
	totalCount, _ := toolCtx.DB().CountListSubscribers(ctx, db.CountListSubscribersParams{
		ListID:       listID,
		FilterStatus: filterStatus,
	})

	// Get subscribers
	subscribers, err := toolCtx.DB().ListListSubscribers(ctx, db.ListListSubscribersParams{
		ListID:       listID,
		FilterStatus: filterStatus,
		PageOffset:   int64(offset),
		PageSize:     int64(pageSize),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list subscribers: %w", err)
	}

	items := make([]ListSubscriberItem, 0, len(subscribers))
	for _, s := range subscribers {
		items = append(items, ListSubscriberItem{
			ID:           s.ID,
			ContactID:    s.ContactID,
			Email:        s.Email,
			Name:         s.Name,
			Status:       s.Status.String,
			SubscribedAt: s.SubscribedAt.String,
			VerifiedAt:   s.VerifiedAt.String,
		})
	}

	return nil, ListSubscribersOutput{
		ListID:      input.ID,
		ListName:    list.Name,
		Subscribers: items,
		Total:       totalCount,
		Page:        page,
		PageSize:    pageSize,
	}, nil
}

// ============================================================================
// SEQUENCE HANDLERS
// ============================================================================

// SequenceItem represents a single sequence in the list.
type SequenceItem struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	ListID       string `json:"list_id"`
	ListName     string `json:"list_name,omitempty"`
	TriggerEvent string `json:"trigger_event"`
	SequenceType string `json:"sequence_type"`
	Active       bool   `json:"active"`
	EmailCount   int    `json:"email_count"`
}

// SequenceListOutput defines output for sequence.list.
type SequenceListOutput struct {
	Sequences []SequenceItem `json:"sequences"`
	Total     int            `json:"total"`
}

// SequenceCreateOutput defines output for sequence.create.
type SequenceCreateOutput struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	ListID       string `json:"list_id"`
	TriggerEvent string `json:"trigger_event"`
	SequenceType string `json:"sequence_type"`
	Active       bool   `json:"active"`
	Created      bool   `json:"created"`
}

// SequenceEmail represents an email in a sequence.
type SequenceEmail struct {
	ID         string `json:"id"`
	Position   int    `json:"position"`
	Subject    string `json:"subject"`
	DelayHours int    `json:"delay_hours"`
	Active     bool   `json:"active"`
}

// SequenceGetOutput defines output for sequence.get.
type SequenceGetOutput struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Slug         string          `json:"slug"`
	ListID       string          `json:"list_id"`
	TriggerEvent string          `json:"trigger_event"`
	SequenceType string          `json:"sequence_type"`
	Active       bool            `json:"active"`
	Emails       []SequenceEmail `json:"emails"`
}

// SequenceUpdateOutput defines output for sequence.update.
type SequenceUpdateOutput struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	TriggerEvent string `json:"trigger_event"`
	SequenceType string `json:"sequence_type"`
	Active       bool   `json:"active"`
	Updated      bool   `json:"updated"`
}

// SequenceStatsOutput defines output for sequence.stats.
type SequenceStatsOutput struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	TotalSubscribers int64   `json:"total_subscribers"`
	CompletedCount   int64   `json:"completed_count"`
	UnsubscribedCount int64  `json:"unsubscribed_count"`
	EmailsSent       int64   `json:"emails_sent"`
	EmailsPending    int64   `json:"emails_pending"`
	OpenedCount      int64   `json:"opened_count"`
	ClickedCount     int64   `json:"clicked_count"`
	OpenRate         float64 `json:"open_rate"`
	ClickRate        float64 `json:"click_rate"`
}

func handleSequence(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	switch input.Action {
	case "create":
		return handleSequenceCreate(ctx, toolCtx, input)
	case "list":
		return handleSequenceList(ctx, toolCtx, input)
	case "get":
		return handleSequenceGet(ctx, toolCtx, input)
	case "update":
		return handleSequenceUpdate(ctx, toolCtx, input)
	case "delete":
		return handleSequenceDelete(ctx, toolCtx, input)
	case "stats":
		return handleSequenceStats(ctx, toolCtx, input)
	}
	return nil, nil, nil
}

func handleSequenceCreate(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.Name) == "" {
		return nil, nil, mcpctx.NewValidationError("name is required", "name")
	}

	if strings.TrimSpace(input.ListID) == "" {
		return nil, nil, mcpctx.NewValidationError("list_id is required", "list_id")
	}
	listID, err := strconv.ParseInt(input.ListID, 10, 64)
	if err != nil {
		return nil, nil, mcpctx.NewValidationError("list_id must be a valid ID", "list_id")
	}
	if err := sameBrand(ctx, toolCtx, access.List, input.ListID, "list"); err != nil {
		return nil, nil, err
	}

	slug := generateSlug(input.Name)
	triggerEvent := input.TriggerEvent
	if triggerEvent == "" {
		triggerEvent = "on_subscribe"
	}
	sequenceType := input.SequenceType
	if sequenceType == "" {
		sequenceType = "lifecycle"
	}
	active := true
	if input.Active != nil {
		active = *input.Active
	}

	brandID := toolCtx.BrandID(ctx)
	sequenceID := uuid.New().String()
	sequence, err := toolCtx.DB().CreateSequence(ctx, db.CreateSequenceParams{
		ID:           sequenceID,
		OrgID:        sql.NullString{String: brandID, Valid: true},
		ListID:       sql.NullInt64{Int64: listID, Valid: true},
		Slug:         slug,
		Name:         input.Name,
		TriggerEvent: triggerEvent,
		IsActive:     sql.NullInt64{Int64: boolToInt64(active), Valid: true},
		SequenceType: sql.NullString{String: sequenceType, Valid: true},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create sequence: %w", err)
	}

	return nil, SequenceCreateOutput{
		ID:           sequence.ID,
		Name:         sequence.Name,
		Slug:         sequence.Slug,
		ListID:       fmt.Sprintf("%d", sequence.ListID.Int64),
		TriggerEvent: sequence.TriggerEvent,
		SequenceType: sequence.SequenceType.String,
		Active:       int64ToBool(sequence.IsActive),
		Created:      true,
	}, nil
}

func handleSequenceList(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	var filterListID int64
	if strings.TrimSpace(input.ListID) != "" {
		var err error
		filterListID, err = strconv.ParseInt(input.ListID, 10, 64)
		if err != nil {
			return nil, nil, mcpctx.NewValidationError("list_id must be a valid ID", "list_id")
		}
	}

	items := make([]SequenceItem, 0)

	brandID := toolCtx.BrandID(ctx)
	sequences, err := toolCtx.DB().ListSequencesByOrg(ctx, sql.NullString{String: brandID, Valid: true})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list sequences: %w", err)
	}

	for _, seq := range sequences {
		if filterListID > 0 && seq.ListID.Int64 != filterListID {
			continue
		}

		templates, err := toolCtx.DB().ListTemplatesBySequence(ctx, sql.NullString{String: seq.ID, Valid: true})
		emailCount := 0
		if err == nil {
			emailCount = len(templates)
		}

		items = append(items, SequenceItem{
			ID:           seq.ID,
			Name:         seq.Name,
			Slug:         seq.Slug,
			ListID:       fmt.Sprintf("%d", seq.ListID.Int64),
			ListName:     seq.ListName.String,
			TriggerEvent: seq.TriggerEvent,
			SequenceType: seq.SequenceType.String,
			Active:       int64ToBool(seq.IsActive),
			EmailCount:   emailCount,
		})
	}

	return nil, SequenceListOutput{
		Sequences: items,
		Total:     len(items),
	}, nil
}

func handleSequenceGet(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	seq, err := toolCtx.DB().GetSequenceByID(ctx, input.ID)
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("sequence %s not found", input.ID))
	}

	if seq.OrgID.String != toolCtx.BrandID(ctx) {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("sequence %s not found", input.ID))
	}

	templates, err := toolCtx.DB().ListTemplatesBySequence(ctx, sql.NullString{String: input.ID, Valid: true})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get sequence emails: %w", err)
	}

	emails := make([]SequenceEmail, 0, len(templates))
	for _, t := range templates {
		emails = append(emails, SequenceEmail{
			ID:         t.ID,
			Position:   int(t.Position),
			Subject:    t.Subject,
			DelayHours: int(t.DelayHours),
			Active:     int64ToBool(t.IsActive),
		})
	}

	return nil, SequenceGetOutput{
		ID:           seq.ID,
		Name:         seq.Name,
		Slug:         seq.Slug,
		ListID:       fmt.Sprintf("%d", seq.ListID.Int64),
		TriggerEvent: seq.TriggerEvent,
		SequenceType: seq.SequenceType.String,
		Active:       int64ToBool(seq.IsActive),
		Emails:       emails,
	}, nil
}

func handleSequenceUpdate(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	seq, err := toolCtx.DB().GetSequenceByID(ctx, input.ID)
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("sequence %s not found", input.ID))
	}

	if seq.OrgID.String != toolCtx.BrandID(ctx) {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("sequence %s not found", input.ID))
	}

	// Apply updates
	name := seq.Name
	if input.Name != "" {
		name = input.Name
	}
	triggerEvent := seq.TriggerEvent
	if input.TriggerEvent != "" {
		triggerEvent = input.TriggerEvent
	}
	sequenceType := seq.SequenceType.String
	if input.SequenceType != "" {
		sequenceType = input.SequenceType
	}
	active := int64ToBool(seq.IsActive)
	if input.Active != nil {
		active = *input.Active
	}

	err = toolCtx.DB().UpdateSequence(ctx, db.UpdateSequenceParams{
		ID:           input.ID,
		Name:         name,
		TriggerEvent: triggerEvent,
		SequenceType: sql.NullString{String: sequenceType, Valid: true},
		IsActive:     sql.NullInt64{Int64: boolToInt64(active), Valid: true},
		SendHour:     seq.SendHour,
		SendTimezone: seq.SendTimezone,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update sequence: %w", err)
	}

	return nil, SequenceUpdateOutput{
		ID:           input.ID,
		Name:         name,
		Slug:         seq.Slug,
		TriggerEvent: triggerEvent,
		SequenceType: sequenceType,
		Active:       active,
		Updated:      true,
	}, nil
}

func handleSequenceDelete(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	seq, err := toolCtx.DB().GetSequenceByID(ctx, input.ID)
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("sequence %s not found", input.ID))
	}

	if seq.OrgID.String != toolCtx.BrandID(ctx) {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("sequence %s not found", input.ID))
	}

	// Delete templates first
	templates, err := toolCtx.DB().ListTemplatesBySequence(ctx, sql.NullString{String: input.ID, Valid: true})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list sequence emails: %w", err)
	}
	for _, t := range templates {
		if err := toolCtx.DB().DeleteTemplate(ctx, t.ID); err != nil {
			return nil, nil, fmt.Errorf("failed to delete sequence email %s: %w", t.ID, err)
		}
	}

	err = toolCtx.DB().DeleteSequence(ctx, input.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to delete sequence: %w", err)
	}

	return nil, DeleteOutput{
		Success: true,
		Message: fmt.Sprintf("Sequence %s deleted successfully", input.ID),
	}, nil
}

func handleSequenceStats(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	seq, err := toolCtx.DB().GetSequenceByID(ctx, input.ID)
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("sequence %s not found", input.ID))
	}

	if seq.OrgID.String != toolCtx.BrandID(ctx) {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("sequence %s not found", input.ID))
	}

	// Get sequence subscriber stats
	seqStats, err := toolCtx.DB().GetSequenceStats(ctx, sql.NullString{String: input.ID, Valid: true})
	if err != nil {
		// Stats query may fail if no data, use zero values
		seqStats = db.GetSequenceStatsRow{}
	}

	// Get email engagement stats
	emailStats, err := toolCtx.DB().GetEmailStatsForSequence(ctx, sql.NullString{String: input.ID, Valid: true})
	if err != nil {
		// Stats query may fail if no data, use zero values
		emailStats = db.GetEmailStatsForSequenceRow{}
	}

	// Calculate rates
	sentCount := int64(0)
	openedCount := int64(0)
	clickedCount := int64(0)
	if emailStats.SentCount.Valid {
		sentCount = int64(emailStats.SentCount.Float64)
	}
	if emailStats.OpenedCount.Valid {
		openedCount = int64(emailStats.OpenedCount.Float64)
	}
	if emailStats.ClickedCount.Valid {
		clickedCount = int64(emailStats.ClickedCount.Float64)
	}

	var openRate, clickRate float64
	if sentCount > 0 {
		openRate = float64(openedCount) / float64(sentCount) * 100
		clickRate = float64(clickedCount) / float64(sentCount) * 100
	}

	return nil, SequenceStatsOutput{
		ID:                input.ID,
		Name:              seq.Name,
		TotalSubscribers:  seqStats.TotalSubscribers,
		CompletedCount:    seqStats.Completed,
		UnsubscribedCount: seqStats.Unsubscribed,
		EmailsSent:        seqStats.EmailsSent,
		EmailsPending:     seqStats.EmailsPending,
		OpenedCount:       openedCount,
		ClickedCount:      clickedCount,
		OpenRate:          openRate,
		ClickRate:         clickRate,
	}, nil
}

// ============================================================================
// TEMPLATE HANDLERS
// ============================================================================

// TemplateItem represents a template/email in the list.
type TemplateItem struct {
	ID         string `json:"id"`
	Position   int    `json:"position"`
	Subject    string `json:"subject"`
	DelayHours int    `json:"delay_hours"`
	Active     bool   `json:"active"`
}

// TemplateListOutput defines output for template.list.
type TemplateListOutput struct {
	SequenceID string         `json:"sequence_id"`
	Emails     []TemplateItem `json:"emails"`
	Total      int            `json:"total"`
}

// TemplateCreateOutput defines output for template.create.
type TemplateCreateOutput struct {
	ID         string `json:"id"`
	SequenceID string `json:"sequence_id"`
	Subject    string `json:"subject"`
	Position   int    `json:"position"`
	DelayHours int    `json:"delay_hours"`
	Active     bool   `json:"active"`
	Created    bool   `json:"created"`
}

// TemplateGetOutput defines output for template.get.
type TemplateGetOutput struct {
	ID         string `json:"id"`
	SequenceID string `json:"sequence_id"`
	Position   int    `json:"position"`
	Subject    string `json:"subject"`
	HTMLBody   string `json:"html_body"`
	PlainText  string `json:"plain_text,omitempty"`
	DelayHours int    `json:"delay_hours"`
	Active     bool   `json:"active"`
}

// TemplateUpdateOutput defines output for template.update.
type TemplateUpdateOutput struct {
	ID         string `json:"id"`
	SequenceID string `json:"sequence_id"`
	Subject    string `json:"subject"`
	Position   int    `json:"position"`
	DelayHours int    `json:"delay_hours"`
	Active     bool   `json:"active"`
	Updated    bool   `json:"updated"`
}

func handleTemplate(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	switch input.Action {
	case "create":
		return handleTemplateCreate(ctx, toolCtx, input)
	case "list":
		return handleTemplateList(ctx, toolCtx, input)
	case "get":
		return handleTemplateGet(ctx, toolCtx, input)
	case "update":
		return handleTemplateUpdate(ctx, toolCtx, input)
	case "delete":
		return handleTemplateDelete(ctx, toolCtx, input)
	}
	return nil, nil, nil
}

func handleTemplateCreate(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.SequenceID) == "" {
		return nil, nil, mcpctx.NewValidationError("sequence_id is required", "sequence_id")
	}
	if err := sameBrand(ctx, toolCtx, access.Sequence, input.SequenceID, "sequence"); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.Subject) == "" {
		return nil, nil, mcpctx.NewValidationError("subject is required", "subject")
	}

	if strings.TrimSpace(input.HTMLBody) == "" {
		return nil, nil, mcpctx.NewValidationError("html_body is required", "html_body")
	}

	position := input.Position
	if position <= 0 {
		templates, err := toolCtx.DB().ListTemplatesBySequence(ctx, sql.NullString{String: input.SequenceID, Valid: true})
		if err == nil {
			position = len(templates) + 1
		} else {
			position = 1
		}
	}

	active := true
	if input.Active != nil {
		active = *input.Active
	}

	templateID := uuid.New().String()
	template, err := toolCtx.DB().CreateTemplate(ctx, db.CreateTemplateParams{
		ID:           templateID,
		SequenceID:   sql.NullString{String: input.SequenceID, Valid: true},
		Position:     int64(position),
		DelayHours:   int64(input.DelayHours),
		Subject:      input.Subject,
		HtmlBody:     input.HTMLBody,
		PlainText:    sql.NullString{String: input.PlainText, Valid: input.PlainText != ""},
		TemplateType: sql.NullString{String: "sequence", Valid: true},
		IsActive:     sql.NullInt64{Int64: boolToInt64(active), Valid: true},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to add email to sequence: %w", err)
	}

	return nil, TemplateCreateOutput{
		ID:         template.ID,
		SequenceID: template.SequenceID.String,
		Subject:    template.Subject,
		Position:   int(template.Position),
		DelayHours: int(template.DelayHours),
		Active:     int64ToBool(template.IsActive),
		Created:    true,
	}, nil
}

func handleTemplateList(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.SequenceID) == "" {
		return nil, nil, mcpctx.NewValidationError("sequence_id is required", "sequence_id")
	}

	seq, err := toolCtx.DB().GetSequenceByID(ctx, input.SequenceID)
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("sequence %s not found", input.SequenceID))
	}

	if seq.OrgID.String != toolCtx.BrandID(ctx) {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("sequence %s not found", input.SequenceID))
	}

	templates, err := toolCtx.DB().ListTemplatesBySequence(ctx, sql.NullString{String: input.SequenceID, Valid: true})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list sequence emails: %w", err)
	}

	emails := make([]TemplateItem, 0, len(templates))
	for _, t := range templates {
		emails = append(emails, TemplateItem{
			ID:         t.ID,
			Position:   int(t.Position),
			Subject:    t.Subject,
			DelayHours: int(t.DelayHours),
			Active:     int64ToBool(t.IsActive),
		})
	}

	return nil, TemplateListOutput{
		SequenceID: input.SequenceID,
		Emails:     emails,
		Total:      len(emails),
	}, nil
}

func handleTemplateGet(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	template, err := toolCtx.DB().GetTemplateByID(ctx, input.ID)
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("template %s not found", input.ID))
	}

	if !template.SequenceID.Valid {
		return nil, nil, mcpctx.NewValidationError("template is not part of a sequence", "id")
	}

	seq, err := toolCtx.DB().GetSequenceByID(ctx, template.SequenceID.String)
	if err != nil || seq.OrgID.String != toolCtx.BrandID(ctx) {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("template %s not found", input.ID))
	}

	return nil, TemplateGetOutput{
		ID:         template.ID,
		SequenceID: template.SequenceID.String,
		Position:   int(template.Position),
		Subject:    template.Subject,
		HTMLBody:   template.HtmlBody,
		PlainText:  template.PlainText.String,
		DelayHours: int(template.DelayHours),
		Active:     int64ToBool(template.IsActive),
	}, nil
}

func handleTemplateUpdate(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	template, err := toolCtx.DB().GetTemplateByID(ctx, input.ID)
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("template %s not found", input.ID))
	}

	if !template.SequenceID.Valid {
		return nil, nil, mcpctx.NewValidationError("template is not part of a sequence", "id")
	}

	seq, err := toolCtx.DB().GetSequenceByID(ctx, template.SequenceID.String)
	if err != nil || seq.OrgID.String != toolCtx.BrandID(ctx) {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("template %s not found", input.ID))
	}

	// Apply updates
	subject := template.Subject
	if input.Subject != "" {
		subject = input.Subject
	}
	htmlBody := template.HtmlBody
	if input.HTMLBody != "" {
		htmlBody = input.HTMLBody
	}
	plainText := template.PlainText.String
	if input.PlainText != "" {
		plainText = input.PlainText
	}
	delayHours := int(template.DelayHours)
	if input.DelayHours > 0 {
		delayHours = input.DelayHours
	}
	position := int(template.Position)
	if input.Position > 0 {
		position = input.Position
	}
	active := int64ToBool(template.IsActive)
	if input.Active != nil {
		active = *input.Active
	}

	err = toolCtx.DB().UpdateTemplate(ctx, db.UpdateTemplateParams{
		ID:           input.ID,
		Position:     int64(position),
		DelayHours:   int64(delayHours),
		Subject:      subject,
		HtmlBody:     htmlBody,
		PlainText:    sql.NullString{String: plainText, Valid: plainText != ""},
		TemplateType: template.TemplateType,
		IsActive:     sql.NullInt64{Int64: boolToInt64(active), Valid: true},
		DesignID:     template.DesignID,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update template: %w", err)
	}

	return nil, TemplateUpdateOutput{
		ID:         input.ID,
		SequenceID: template.SequenceID.String,
		Subject:    subject,
		Position:   position,
		DelayHours: delayHours,
		Active:     active,
		Updated:    true,
	}, nil
}

func handleTemplateDelete(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	template, err := toolCtx.DB().GetTemplateByID(ctx, input.ID)
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("template %s not found", input.ID))
	}

	if !template.SequenceID.Valid {
		return nil, nil, mcpctx.NewValidationError("template is not part of a sequence", "id")
	}

	seq, err := toolCtx.DB().GetSequenceByID(ctx, template.SequenceID.String)
	if err != nil || seq.OrgID.String != toolCtx.BrandID(ctx) {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("template %s not found", input.ID))
	}

	err = toolCtx.DB().DeleteTemplate(ctx, input.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to delete template: %w", err)
	}

	return nil, DeleteOutput{
		Success: true,
		Message: fmt.Sprintf("Template %s deleted successfully", input.ID),
	}, nil
}

// ============================================================================
// LIST SUBSCRIBE/UNSUBSCRIBE HANDLERS
// ============================================================================

// ListSubscribeOutput defines output for list.subscribe.
type ListSubscribeOutput struct {
	ListID    string `json:"list_id"`
	ContactID string `json:"contact_id"`
	Email     string `json:"email"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

// ListUnsubscribeOutput defines output for list.unsubscribe.
type ListUnsubscribeOutput struct {
	ListID  string `json:"list_id"`
	Email   string `json:"email"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func handleListSubscribe(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id (list ID) is required", "id")
	}

	if strings.TrimSpace(input.Email) == "" {
		return nil, nil, mcpctx.NewValidationError("email is required", "email")
	}

	listID, err := strconv.ParseInt(input.ID, 10, 64)
	if err != nil {
		return nil, nil, mcpctx.NewValidationError("id must be a valid ID", "id")
	}

	// Verify list exists and belongs to brand
	list, err := toolCtx.DB().GetEmailList(ctx, listID)
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("list %s not found", input.ID))
	}
	if list.OrgID != toolCtx.BrandID(ctx) {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("list %s not found", input.ID))
	}

	// Find or create contact
	brandID := toolCtx.BrandID(ctx)
	contact, err := toolCtx.DB().GetContactByOrgAndEmail(ctx, db.GetContactByOrgAndEmailParams{
		OrgID: sql.NullString{String: brandID, Valid: true},
		Email: input.Email,
	})
	if err != nil {
		// Create new contact
		contactID := uuid.New().String()
		contact, err = toolCtx.DB().CreateContact(ctx, db.CreateContactParams{
			ID:     contactID,
			OrgID:  sql.NullString{String: brandID, Valid: true},
			Name:   "",
			Email:  input.Email,
			Source: sql.NullString{String: "mcp", Valid: true},
			Status: "active",
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create contact: %w", err)
		}
	}

	// Subscribe to list
	subID := uuid.New().String()
	_, err = toolCtx.DB().SubscribeToList(ctx, db.SubscribeToListParams{
		ID:        subID,
		ListID:    listID,
		ContactID: contact.ID,
	})
	if err != nil {
		// May already be subscribed
		return nil, ListSubscribeOutput{
			ListID:    input.ID,
			ContactID: contact.ID,
			Email:     input.Email,
			Status:    "already_subscribed",
			Message:   "Contact is already subscribed to this list",
		}, nil
	}

	return nil, ListSubscribeOutput{
		ListID:    input.ID,
		ContactID: contact.ID,
		Email:     input.Email,
		Status:    "subscribed",
		Message:   fmt.Sprintf("Successfully subscribed %s to list", input.Email),
	}, nil
}

func handleListUnsubscribe(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id (list ID) is required", "id")
	}

	if strings.TrimSpace(input.Email) == "" {
		return nil, nil, mcpctx.NewValidationError("email is required", "email")
	}

	listID, err := strconv.ParseInt(input.ID, 10, 64)
	if err != nil {
		return nil, nil, mcpctx.NewValidationError("id must be a valid ID", "id")
	}

	// Verify list exists and belongs to brand
	list, err := toolCtx.DB().GetEmailList(ctx, listID)
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("list %s not found", input.ID))
	}
	if list.OrgID != toolCtx.BrandID(ctx) {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("list %s not found", input.ID))
	}

	// Find contact
	brandID := toolCtx.BrandID(ctx)
	contact, err := toolCtx.DB().GetContactByOrgAndEmail(ctx, db.GetContactByOrgAndEmailParams{
		OrgID: sql.NullString{String: brandID, Valid: true},
		Email: input.Email,
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("contact with email %s not found", input.Email))
	}

	// Unsubscribe from list
	err = toolCtx.DB().UnsubscribeFromList(ctx, db.UnsubscribeFromListParams{
		ListID:    listID,
		ContactID: contact.ID,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unsubscribe: %w", err)
	}

	return nil, ListUnsubscribeOutput{
		ListID:  input.ID,
		Email:   input.Email,
		Success: true,
		Message: fmt.Sprintf("Successfully unsubscribed %s from list", input.Email),
	}, nil
}

// ============================================================================
// ENROLLMENT HANDLERS
// ============================================================================

// EnrollmentItem represents an enrollment in the list.
type EnrollmentItem struct {
	ID              string `json:"id"`
	SequenceID      string `json:"sequence_id"`
	SequenceName    string `json:"sequence_name"`
	CurrentPosition int    `json:"current_position"`
	TotalSteps      int    `json:"total_steps"`
	Status          string `json:"status"`
	StartedAt       string `json:"started_at,omitempty"`
	CompletedAt     string `json:"completed_at,omitempty"`
	PausedAt        string `json:"paused_at,omitempty"`
}

// EnrollmentListOutput defines output for enrollment.list.
type EnrollmentListOutput struct {
	ContactID   string           `json:"contact_id"`
	Enrollments []EnrollmentItem `json:"enrollments"`
	Total       int              `json:"total"`
}

// EnrollmentActionOutput defines output for enrollment actions.
type EnrollmentActionOutput struct {
	SequenceID string `json:"sequence_id"`
	ContactID  string `json:"contact_id"`
	Action     string `json:"action"`
	Success    bool   `json:"success"`
	Message    string `json:"message"`
}

func handleEnrollment(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	switch input.Action {
	case "enroll":
		return handleEnrollmentEnroll(ctx, toolCtx, input)
	case "unenroll":
		return handleEnrollmentUnenroll(ctx, toolCtx, input)
	case "pause":
		return handleEnrollmentPause(ctx, toolCtx, input)
	case "resume":
		return handleEnrollmentResume(ctx, toolCtx, input)
	case "list":
		return handleEnrollmentList(ctx, toolCtx, input)
	}
	return nil, nil, nil
}

func handleEnrollmentEnroll(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.SequenceID) == "" {
		return nil, nil, mcpctx.NewValidationError("sequence_id is required", "sequence_id")
	}

	if strings.TrimSpace(input.ContactID) == "" {
		return nil, nil, mcpctx.NewValidationError("contact_id is required", "contact_id")
	}

	// Verify sequence exists and belongs to brand
	seq, err := toolCtx.DB().GetSequenceByID(ctx, input.SequenceID)
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("sequence %s not found", input.SequenceID))
	}
	if seq.OrgID.String != toolCtx.BrandID(ctx) {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("sequence %s not found", input.SequenceID))
	}

	// Verify contact exists and belongs to brand
	contact, err := toolCtx.DB().GetContactByOrgID(ctx, db.GetContactByOrgIDParams{
		ID:    input.ContactID,
		OrgID: sql.NullString{String: toolCtx.BrandID(ctx), Valid: true},
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("contact %s not found", input.ContactID))
	}

	// Create enrollment
	stateID := uuid.New().String()
	_, err = toolCtx.DB().CreateContactSequenceState(ctx, db.CreateContactSequenceStateParams{
		ID:              stateID,
		ContactID:       sql.NullString{String: contact.ID, Valid: true},
		SequenceID:      sql.NullString{String: input.SequenceID, Valid: true},
		CurrentPosition: sql.NullInt64{Int64: 0, Valid: true},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to enroll contact: %w", err)
	}

	return nil, EnrollmentActionOutput{
		SequenceID: input.SequenceID,
		ContactID:  input.ContactID,
		Action:     "enroll",
		Success:    true,
		Message:    fmt.Sprintf("Contact enrolled in sequence %s", seq.Name),
	}, nil
}

func handleEnrollmentUnenroll(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.SequenceID) == "" {
		return nil, nil, mcpctx.NewValidationError("sequence_id is required", "sequence_id")
	}

	if strings.TrimSpace(input.ContactID) == "" {
		return nil, nil, mcpctx.NewValidationError("contact_id is required", "contact_id")
	}

	// Verify sequence belongs to brand
	seq, err := toolCtx.DB().GetSequenceByID(ctx, input.SequenceID)
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("sequence %s not found", input.SequenceID))
	}
	if seq.OrgID.String != toolCtx.BrandID(ctx) {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("sequence %s not found", input.SequenceID))
	}

	// Cancel the sequence enrollment
	err = toolCtx.DB().CancelContactSequence(ctx, db.CancelContactSequenceParams{
		ContactID:  sql.NullString{String: input.ContactID, Valid: true},
		SequenceID: sql.NullString{String: input.SequenceID, Valid: true},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unenroll contact: %w", err)
	}

	// Also cancel any pending emails
	_ = toolCtx.DB().CancelPendingEmailsForContactSequence(ctx, db.CancelPendingEmailsForContactSequenceParams{
		ContactID:  sql.NullString{String: input.ContactID, Valid: true},
		SequenceID: sql.NullString{String: input.SequenceID, Valid: true},
	})

	return nil, EnrollmentActionOutput{
		SequenceID: input.SequenceID,
		ContactID:  input.ContactID,
		Action:     "unenroll",
		Success:    true,
		Message:    fmt.Sprintf("Contact unenrolled from sequence %s", seq.Name),
	}, nil
}

func handleEnrollmentPause(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.SequenceID) == "" {
		return nil, nil, mcpctx.NewValidationError("sequence_id is required", "sequence_id")
	}

	if strings.TrimSpace(input.ContactID) == "" {
		return nil, nil, mcpctx.NewValidationError("contact_id is required", "contact_id")
	}

	// Verify sequence belongs to brand
	seq, err := toolCtx.DB().GetSequenceByID(ctx, input.SequenceID)
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("sequence %s not found", input.SequenceID))
	}
	if seq.OrgID.String != toolCtx.BrandID(ctx) {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("sequence %s not found", input.SequenceID))
	}

	err = toolCtx.DB().PauseContactSequence(ctx, db.PauseContactSequenceParams{
		ContactID:  sql.NullString{String: input.ContactID, Valid: true},
		SequenceID: sql.NullString{String: input.SequenceID, Valid: true},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to pause enrollment: %w", err)
	}

	return nil, EnrollmentActionOutput{
		SequenceID: input.SequenceID,
		ContactID:  input.ContactID,
		Action:     "pause",
		Success:    true,
		Message:    fmt.Sprintf("Enrollment paused for sequence %s", seq.Name),
	}, nil
}

func handleEnrollmentResume(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.SequenceID) == "" {
		return nil, nil, mcpctx.NewValidationError("sequence_id is required", "sequence_id")
	}

	if strings.TrimSpace(input.ContactID) == "" {
		return nil, nil, mcpctx.NewValidationError("contact_id is required", "contact_id")
	}

	// Verify sequence belongs to brand
	seq, err := toolCtx.DB().GetSequenceByID(ctx, input.SequenceID)
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("sequence %s not found", input.SequenceID))
	}
	if seq.OrgID.String != toolCtx.BrandID(ctx) {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("sequence %s not found", input.SequenceID))
	}

	err = toolCtx.DB().ResumeContactSequence(ctx, db.ResumeContactSequenceParams{
		ContactID:  sql.NullString{String: input.ContactID, Valid: true},
		SequenceID: sql.NullString{String: input.SequenceID, Valid: true},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resume enrollment: %w", err)
	}

	return nil, EnrollmentActionOutput{
		SequenceID: input.SequenceID,
		ContactID:  input.ContactID,
		Action:     "resume",
		Success:    true,
		Message:    fmt.Sprintf("Enrollment resumed for sequence %s", seq.Name),
	}, nil
}

func handleEnrollmentList(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ContactID) == "" {
		return nil, nil, mcpctx.NewValidationError("contact_id is required", "contact_id")
	}

	// Verify contact exists and belongs to brand
	_, err := toolCtx.DB().GetContactByOrgID(ctx, db.GetContactByOrgIDParams{
		ID:    input.ContactID,
		OrgID: sql.NullString{String: toolCtx.BrandID(ctx), Valid: true},
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("contact %s not found", input.ContactID))
	}

	states, err := toolCtx.DB().ListContactSequenceStatesWithDetails(ctx, db.ListContactSequenceStatesWithDetailsParams{
		ContactID: sql.NullString{String: input.ContactID, Valid: true},
		OrgID:     sql.NullString{String: toolCtx.BrandID(ctx), Valid: true},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list enrollments: %w", err)
	}

	items := make([]EnrollmentItem, 0, len(states))
	for _, s := range states {
		status := "active"
		if s.CompletedAt.Valid && s.CompletedAt.String != "" {
			status = "completed"
		} else if s.UnsubscribedAt.Valid && s.UnsubscribedAt.String != "" {
			status = "unsubscribed"
		} else if s.PausedAt.Valid && s.PausedAt.String != "" {
			status = "paused"
		} else if !int64ToBool(s.IsActive) {
			status = "inactive"
		}

		items = append(items, EnrollmentItem{
			ID:              s.ID,
			SequenceID:      s.SequenceID.String,
			SequenceName:    s.SequenceName,
			CurrentPosition: int(s.CurrentPosition.Int64),
			TotalSteps:      int(s.TotalSteps),
			Status:          status,
			StartedAt:       s.StartedAt.String,
			CompletedAt:     s.CompletedAt.String,
			PausedAt:        s.PausedAt.String,
		})
	}

	return nil, EnrollmentListOutput{
		ContactID:   input.ContactID,
		Enrollments: items,
		Total:       len(items),
	}, nil
}

// ============================================================================
// ENTRY RULE HANDLERS
// ============================================================================

// EntryRuleItem represents an entry rule in the list.
type EntryRuleItem struct {
	ID          string `json:"id"`
	SequenceID  string `json:"sequence_id"`
	TriggerType string `json:"trigger_type"`
	SourceID    string `json:"source_id"`
	SourceName  string `json:"source_name,omitempty"`
	Priority    int    `json:"priority"`
	Active      bool   `json:"active"`
	CreatedAt   string `json:"created_at,omitempty"`
}

// EntryRuleListOutput defines output for entry_rule.list.
type EntryRuleListOutput struct {
	SequenceID string          `json:"sequence_id"`
	Rules      []EntryRuleItem `json:"rules"`
	Total      int             `json:"total"`
}

// EntryRuleCreateOutput defines output for entry_rule.create.
type EntryRuleCreateOutput struct {
	ID          string `json:"id"`
	SequenceID  string `json:"sequence_id"`
	TriggerType string `json:"trigger_type"`
	SourceID    string `json:"source_id"`
	Priority    int    `json:"priority"`
	Active      bool   `json:"active"`
	Created     bool   `json:"created"`
}

// EntryRuleUpdateOutput defines output for entry_rule.update.
type EntryRuleUpdateOutput struct {
	ID       string `json:"id"`
	Priority int    `json:"priority"`
	Active   bool   `json:"active"`
	Updated  bool   `json:"updated"`
}

func handleEntryRule(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	switch input.Action {
	case "create":
		return handleEntryRuleCreate(ctx, toolCtx, input)
	case "list":
		return handleEntryRuleList(ctx, toolCtx, input)
	case "update":
		return handleEntryRuleUpdate(ctx, toolCtx, input)
	case "delete":
		return handleEntryRuleDelete(ctx, toolCtx, input)
	}
	return nil, nil, nil
}

func handleEntryRuleCreate(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.SequenceID) == "" {
		return nil, nil, mcpctx.NewValidationError("sequence_id is required", "sequence_id")
	}

	if strings.TrimSpace(input.TriggerType) == "" {
		return nil, nil, mcpctx.NewValidationError("trigger_type is required (list_subscribe, sequence_complete, tag_added)", "trigger_type")
	}

	if strings.TrimSpace(input.SourceID) == "" {
		return nil, nil, mcpctx.NewValidationError("source_id is required", "source_id")
	}

	// Verify sequence belongs to brand
	seq, err := toolCtx.DB().GetSequenceByID(ctx, input.SequenceID)
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("sequence %s not found", input.SequenceID))
	}
	if seq.OrgID.String != toolCtx.BrandID(ctx) {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("sequence %s not found", input.SequenceID))
	}

	priority := input.Priority
	if priority <= 0 {
		priority = 10
	}

	active := true
	if input.Active != nil {
		active = *input.Active
	}

	ruleID := uuid.New().String()
	rule, err := toolCtx.DB().CreateEntryRule(ctx, db.CreateEntryRuleParams{
		ID:          ruleID,
		SequenceID:  input.SequenceID,
		TriggerType: input.TriggerType,
		SourceID:    input.SourceID,
		Priority:    sql.NullInt64{Int64: int64(priority), Valid: true},
		IsActive:    sql.NullInt64{Int64: boolToInt64(active), Valid: true},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create entry rule: %w", err)
	}

	return nil, EntryRuleCreateOutput{
		ID:          rule.ID,
		SequenceID:  rule.SequenceID,
		TriggerType: rule.TriggerType,
		SourceID:    rule.SourceID,
		Priority:    int(rule.Priority.Int64),
		Active:      int64ToBool(rule.IsActive),
		Created:     true,
	}, nil
}

func handleEntryRuleList(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.SequenceID) == "" {
		return nil, nil, mcpctx.NewValidationError("sequence_id is required", "sequence_id")
	}

	// Verify sequence belongs to brand
	seq, err := toolCtx.DB().GetSequenceByID(ctx, input.SequenceID)
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("sequence %s not found", input.SequenceID))
	}
	if seq.OrgID.String != toolCtx.BrandID(ctx) {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("sequence %s not found", input.SequenceID))
	}

	rules, err := toolCtx.DB().ListEntryRulesBySequence(ctx, input.SequenceID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list entry rules: %w", err)
	}

	items := make([]EntryRuleItem, 0, len(rules))
	for _, r := range rules {
		sourceName := ""
		if r.ListName.Valid {
			sourceName = r.ListName.String
		} else if r.SourceSequenceName.Valid {
			sourceName = r.SourceSequenceName.String
		}

		items = append(items, EntryRuleItem{
			ID:          r.ID,
			SequenceID:  r.SequenceID,
			TriggerType: r.TriggerType,
			SourceID:    r.SourceID,
			SourceName:  sourceName,
			Priority:    int(r.Priority.Int64),
			Active:      int64ToBool(r.IsActive),
			CreatedAt:   r.CreatedAt.String,
		})
	}

	return nil, EntryRuleListOutput{
		SequenceID: input.SequenceID,
		Rules:      items,
		Total:      len(items),
	}, nil
}

func handleEntryRuleUpdate(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	// Get existing rule
	rule, err := toolCtx.DB().GetEntryRule(ctx, input.ID)
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("entry rule %s not found", input.ID))
	}

	// Verify sequence belongs to brand
	seq, err := toolCtx.DB().GetSequenceByID(ctx, rule.SequenceID)
	if err != nil || seq.OrgID.String != toolCtx.BrandID(ctx) {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("entry rule %s not found", input.ID))
	}

	priority := int(rule.Priority.Int64)
	if input.Priority > 0 {
		priority = input.Priority
	}

	active := int64ToBool(rule.IsActive)
	if input.Active != nil {
		active = *input.Active
	}

	err = toolCtx.DB().UpdateEntryRule(ctx, db.UpdateEntryRuleParams{
		ID:       input.ID,
		Priority: sql.NullInt64{Int64: int64(priority), Valid: true},
		IsActive: sql.NullInt64{Int64: boolToInt64(active), Valid: true},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update entry rule: %w", err)
	}

	return nil, EntryRuleUpdateOutput{
		ID:       input.ID,
		Priority: priority,
		Active:   active,
		Updated:  true,
	}, nil
}

func handleEntryRuleDelete(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	// Get existing rule
	rule, err := toolCtx.DB().GetEntryRule(ctx, input.ID)
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("entry rule %s not found", input.ID))
	}

	// Verify sequence belongs to brand
	seq, err := toolCtx.DB().GetSequenceByID(ctx, rule.SequenceID)
	if err != nil || seq.OrgID.String != toolCtx.BrandID(ctx) {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("entry rule %s not found", input.ID))
	}

	err = toolCtx.DB().DeleteEntryRule(ctx, input.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to delete entry rule: %w", err)
	}

	return nil, DeleteOutput{
		Success: true,
		Message: fmt.Sprintf("Entry rule %s deleted successfully", input.ID),
	}, nil
}

// ============================================================================
// QUEUE HANDLERS
// ============================================================================

// QueueItem represents a queued email.
type QueueItem struct {
	ID           string `json:"id"`
	ContactID    string `json:"contact_id"`
	ContactEmail string `json:"contact_email"`
	ContactName  string `json:"contact_name,omitempty"`
	TemplateID   string `json:"template_id,omitempty"`
	Subject      string `json:"subject"`
	ScheduledFor string `json:"scheduled_for"`
	Status       string `json:"status"`
	SentAt       string `json:"sent_at,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// QueueListOutput defines output for queue.list.
type QueueListOutput struct {
	Emails []QueueItem `json:"emails"`
	Total  int         `json:"total"`
}

// QueueCancelOutput defines output for queue.cancel.
type QueueCancelOutput struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func handleQueue(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	switch input.Action {
	case "list":
		return handleQueueList(ctx, toolCtx, input)
	case "cancel":
		return handleQueueCancel(ctx, toolCtx, input)
	}
	return nil, nil, nil
}

func handleQueueList(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	var filterStatus interface{}
	if input.Status != "" {
		filterStatus = input.Status
	}

	var filterContactID interface{}
	if input.ContactID != "" {
		filterContactID = input.ContactID
	}

	emails, err := toolCtx.DB().ListEmailQueueByOrg(ctx, db.ListEmailQueueByOrgParams{
		OrgID:           sql.NullString{String: toolCtx.BrandID(ctx), Valid: true},
		FilterStatus:    filterStatus,
		FilterContactID: filterContactID,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list email queue: %w", err)
	}

	items := make([]QueueItem, 0, len(emails))
	for _, e := range emails {
		items = append(items, QueueItem{
			ID:           e.ID,
			ContactID:    e.ContactID.String,
			ContactEmail: e.ContactEmail,
			ContactName:  e.ContactName,
			TemplateID:   e.TemplateID.String,
			Subject:      e.Subject,
			ScheduledFor: e.ScheduledFor,
			Status:       e.Status.String,
			SentAt:       e.SentAt.String,
			ErrorMessage: e.ErrorMessage.String,
		})
	}

	return nil, QueueListOutput{
		Emails: items,
		Total:  len(items),
	}, nil
}

func handleQueueCancel(ctx context.Context, toolCtx *mcpctx.ToolContext, input EmailInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	if err := sameBrand(ctx, toolCtx, access.QueuedEmail, input.ID, "queued email"); err != nil {
		return nil, nil, err
	}

	err := toolCtx.DB().CancelEmail(ctx, input.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to cancel email: %w", err)
	}

	return nil, QueueCancelOutput{
		ID:      input.ID,
		Success: true,
		Message: fmt.Sprintf("Email %s cancelled successfully", input.ID),
	}, nil
}

// registerEmailToolToRegistry registers email tool to the direct-call registry.
func registerEmailToolToRegistry(registry *ToolRegistry, toolCtx *mcpctx.ToolContext) {
	registry.Register("email", func(ctx context.Context, args json.RawMessage) (interface{}, error) {
		var input EmailInput
		if err := json.Unmarshal(args, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		handler := emailHandler(toolCtx)
		_, output, err := handler(ctx, nil, input)
		return output, err
	})
}
