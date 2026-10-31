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

// contactActions defines valid actions for contacts.
var contactActions = []string{"create", "list", "get", "update", "add_tags", "remove_tags", "unsubscribe", "block", "unblock", "activity"}

// ContactInput defines input for the contact tool.
type ContactInput struct {
	Action string `json:"action" jsonschema:"required,Action to perform: create, list, get, update, add_tags, remove_tags, unsubscribe, block, unblock, activity"`

	// Common
	ID string `json:"id,omitempty" jsonschema:"Contact ID (for get, update, add_tags, remove_tags, unsubscribe, block, unblock, activity)"`

	// Create fields
	Email  string `json:"email,omitempty" jsonschema:"Contact email address (required for create)"`
	Name   string `json:"name,omitempty" jsonschema:"Contact name"`
	Source string `json:"source,omitempty" jsonschema:"How the contact was acquired (e.g., 'website', 'import', 'api')"`

	// List filters
	Tag string `json:"tag,omitempty" jsonschema:"Filter by tag (for list)"`

	// Pagination
	Page     int64 `json:"page,omitempty" jsonschema:"Page number (1-based, default: 1)"`
	PageSize int64 `json:"page_size,omitempty" jsonschema:"Items per page (default: 50)"`

	// Tag management
	Tags string `json:"tags,omitempty" jsonschema:"Comma-separated tags to add or remove"`
}

// ContactItem represents a contact in list output.
type ContactItem struct {
	ID            string   `json:"id"`
	Email         string   `json:"email"`
	Name          string   `json:"name,omitempty"`
	Source        string   `json:"source,omitempty"`
	EmailVerified bool     `json:"email_verified"`
	Status        string   `json:"status,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	CreatedAt     string   `json:"created_at"`
}

// ContactListOutput defines output for contact list.
type ContactListOutput struct {
	Contacts []ContactItem `json:"contacts"`
	Total    int           `json:"total"`
	Page     int64         `json:"page"`
	PageSize int64         `json:"page_size"`
}

// ContactCreateOutput defines output for contact create.
type ContactCreateOutput struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name,omitempty"`
	Created bool   `json:"created"`
}

// ContactGetOutput defines output for contact get.
type ContactGetOutput struct {
	ID             string   `json:"id"`
	Email          string   `json:"email"`
	Name           string   `json:"name,omitempty"`
	Source         string   `json:"source,omitempty"`
	EmailVerified  bool     `json:"email_verified"`
	VerifiedAt     string   `json:"verified_at,omitempty"`
	Status         string   `json:"status,omitempty"`
	UnsubscribedAt string   `json:"unsubscribed_at,omitempty"`
	BlockedAt      string   `json:"blocked_at,omitempty"`
	GdprConsent    bool     `json:"gdpr_consent"`
	GdprConsentAt  string   `json:"gdpr_consent_at,omitempty"`
	Tags           []string `json:"tags"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
}

// ContactUpdateOutput defines output for contact update.
type ContactUpdateOutput struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Updated bool   `json:"updated"`
}

// ContactTagsOutput defines output for tag operations.
type ContactTagsOutput struct {
	ID      string   `json:"id"`
	Tags    []string `json:"tags"`
	Success bool     `json:"success"`
	Message string   `json:"message"`
}

// ContactActivityItem represents an activity event.
type ContactActivityItem struct {
	Type      string `json:"type"` // sequence_email, campaign_send, transactional, click
	Subject   string `json:"subject,omitempty"`
	Name      string `json:"name,omitempty"` // sequence/campaign/template name
	Status    string `json:"status,omitempty"`
	SentAt    string `json:"sent_at,omitempty"`
	OpenedAt  string `json:"opened_at,omitempty"`
	ClickedAt string `json:"clicked_at,omitempty"`
	LinkURL   string `json:"link_url,omitempty"`
}

// ContactActivityOutput defines output for activity.
type ContactActivityOutput struct {
	ID         string                `json:"id"`
	Email      string                `json:"email"`
	Activities []ContactActivityItem `json:"activities"`
}

// RegisterContactTool registers the contact tool.
func RegisterContactTool(server *mcp.Server, toolCtx *mcpctx.ToolContext) {
	mcp.AddTool(server, &mcp.Tool{
		Name:  "contact",
		Title: "Contact Management",
		Description: `Manage contacts (subscribers/recipients).

PREREQUISITE: You must first select a brand using brand(resource: brand, action: select).

Actions and Required Fields:
- create: Create a new contact (requires: email)
- list: List contacts (optional: tag filter, page, page_size)
- get: Get contact details with tags (requires: id)
- update: Update contact name (requires: id, name)
- add_tags: Add tags to a contact (requires: id, tags)
- remove_tags: Remove tags from a contact (requires: id, tags)
- unsubscribe: Globally unsubscribe a contact (requires: id)
- block: Block a contact from receiving emails (requires: id)
- unblock: Unblock a contact (requires: id)
- activity: Get contact activity history (requires: id)

Examples:
  contact(action: create, email: "user@example.com", name: "John Doe", source: "website")
  contact(action: list, page: 1, page_size: 20)
  contact(action: list, tag: "premium")
  contact(action: get, id: "uuid")
  contact(action: update, id: "uuid", name: "Jane Doe")
  contact(action: add_tags, id: "uuid", tags: "vip,newsletter")
  contact(action: remove_tags, id: "uuid", tags: "inactive")
  contact(action: unsubscribe, id: "uuid")
  contact(action: block, id: "uuid")
  contact(action: activity, id: "uuid")`,
	}, contactHandler(toolCtx))
}

func contactHandler(toolCtx *mcpctx.ToolContext) func(ctx context.Context, req *mcp.CallToolRequest, input ContactInput) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input ContactInput) (*mcp.CallToolResult, any, error) {
		// Validate action
		if !slices.Contains(contactActions, input.Action) {
			return nil, nil, mcpctx.NewValidationError(
				fmt.Sprintf("invalid action '%s', must be: %s", input.Action, strings.Join(contactActions, ", ")),
				"action")
		}

		switch input.Action {
		case "create":
			return handleContactCreate(ctx, toolCtx, input)
		case "list":
			return handleContactList(ctx, toolCtx, input)
		case "get":
			return handleContactGet(ctx, toolCtx, input)
		case "update":
			return handleContactUpdate(ctx, toolCtx, input)
		case "add_tags":
			return handleContactAddTags(ctx, toolCtx, input)
		case "remove_tags":
			return handleContactRemoveTags(ctx, toolCtx, input)
		case "unsubscribe":
			return handleContactUnsubscribe(ctx, toolCtx, input)
		case "block":
			return handleContactBlock(ctx, toolCtx, input)
		case "unblock":
			return handleContactUnblock(ctx, toolCtx, input)
		case "activity":
			return handleContactActivity(ctx, toolCtx, input)
		}
		return nil, nil, nil
	}
}

func handleContactCreate(ctx context.Context, toolCtx *mcpctx.ToolContext, input ContactInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.Email) == "" {
		return nil, nil, mcpctx.NewValidationError("email is required", "email")
	}

	// Check if contact already exists
	existing, err := toolCtx.DB().GetContactByOrgAndEmail(ctx, db.GetContactByOrgAndEmailParams{
		OrgID: sql.NullString{String: toolCtx.BrandID(ctx), Valid: true},
		Email: input.Email,
	})
	if err == nil {
		// Contact exists
		return nil, ContactCreateOutput{
			ID:      existing.ID,
			Email:   existing.Email,
			Name:    existing.Name,
			Created: false,
		}, nil
	}

	contactID := uuid.New().String()
	contact, err := toolCtx.DB().CreateContact(ctx, db.CreateContactParams{
		ID:     contactID,
		OrgID:  sql.NullString{String: toolCtx.BrandID(ctx), Valid: true},
		Email:  input.Email,
		Name:   input.Name,
		Source: sql.NullString{String: input.Source, Valid: input.Source != ""},
		Status: "new",
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create contact: %w", err)
	}

	return nil, ContactCreateOutput{
		ID:      contact.ID,
		Email:   contact.Email,
		Name:    contact.Name,
		Created: true,
	}, nil
}

func handleContactList(ctx context.Context, toolCtx *mcpctx.ToolContext, input ContactInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	page := input.Page
	if page < 1 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	var contacts []db.Contact
	var err error

	if input.Tag != "" {
		// Filter by tag
		contacts, err = toolCtx.DB().GetContactsByTag(ctx, db.GetContactsByTagParams{
			Tag:        input.Tag,
			PageOffset: offset,
			PageSize:   pageSize,
		})
	} else {
		contacts, err = toolCtx.DB().ListContactsByOrg(ctx, db.ListContactsByOrgParams{
			OrgID:      sql.NullString{String: toolCtx.BrandID(ctx), Valid: true},
			PageOffset: offset,
			PageSize:   pageSize,
		})
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list contacts: %w", err)
	}

	items := make([]ContactItem, 0, len(contacts))
	for _, c := range contacts {
		// Get tags for each contact
		tags, _ := toolCtx.DB().GetContactTags(ctx, sql.NullString{String: c.ID, Valid: true})
		tagNames := make([]string, 0, len(tags))
		for _, t := range tags {
			tagNames = append(tagNames, t.Tag)
		}

		items = append(items, ContactItem{
			ID:            c.ID,
			Email:         c.Email,
			Name:          c.Name,
			Source:        c.Source.String,
			EmailVerified: c.EmailVerified == 1,
			Status:        c.Status.String,
			Tags:          tagNames,
			CreatedAt:     c.CreatedAt.String,
		})
	}

	return nil, ContactListOutput{
		Contacts: items,
		Total:    len(items),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func handleContactGet(ctx context.Context, toolCtx *mcpctx.ToolContext, input ContactInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	contact, err := toolCtx.DB().GetContactByOrgID(ctx, db.GetContactByOrgIDParams{
		ID:    input.ID,
		OrgID: sql.NullString{String: toolCtx.BrandID(ctx), Valid: true},
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("contact %s not found", input.ID))
	}

	// Get tags
	tags, _ := toolCtx.DB().GetContactTags(ctx, sql.NullString{String: contact.ID, Valid: true})
	tagNames := make([]string, 0, len(tags))
	for _, t := range tags {
		tagNames = append(tagNames, t.Tag)
	}

	return nil, ContactGetOutput{
		ID:             contact.ID,
		Email:          contact.Email,
		Name:           contact.Name,
		Source:         contact.Source.String,
		EmailVerified:  contact.EmailVerified == 1,
		VerifiedAt:     contact.VerifiedAt.String,
		Status:         contact.Status.String,
		UnsubscribedAt: contact.UnsubscribedAt.String,
		BlockedAt:      contact.BlockedAt.String,
		GdprConsent:    int64ToBool(contact.GdprConsent),
		GdprConsentAt:  contact.GdprConsentAt.String,
		Tags:           tagNames,
		CreatedAt:      contact.CreatedAt.String,
		UpdatedAt:      contact.UpdatedAt.String,
	}, nil
}

func handleContactUpdate(ctx context.Context, toolCtx *mcpctx.ToolContext, input ContactInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}
	if strings.TrimSpace(input.Name) == "" {
		return nil, nil, mcpctx.NewValidationError("name is required", "name")
	}

	// Verify contact exists
	_, err := toolCtx.DB().GetContactByOrgID(ctx, db.GetContactByOrgIDParams{
		ID:    input.ID,
		OrgID: sql.NullString{String: toolCtx.BrandID(ctx), Valid: true},
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("contact %s not found", input.ID))
	}

	err = toolCtx.DB().UpdateContactName(ctx, db.UpdateContactNameParams{
		ID:   input.ID,
		Name: input.Name,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update contact: %w", err)
	}

	return nil, ContactUpdateOutput{
		ID:      input.ID,
		Name:    input.Name,
		Updated: true,
	}, nil
}

func handleContactAddTags(ctx context.Context, toolCtx *mcpctx.ToolContext, input ContactInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}
	if strings.TrimSpace(input.Tags) == "" {
		return nil, nil, mcpctx.NewValidationError("tags is required (comma-separated)", "tags")
	}

	// Verify contact exists
	_, err := toolCtx.DB().GetContactByOrgID(ctx, db.GetContactByOrgIDParams{
		ID:    input.ID,
		OrgID: sql.NullString{String: toolCtx.BrandID(ctx), Valid: true},
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("contact %s not found", input.ID))
	}

	// Add tags
	tagList := strings.Split(input.Tags, ",")
	for _, tag := range tagList {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		toolCtx.DB().AddContactTag(ctx, db.AddContactTagParams{
			ContactID: sql.NullString{String: input.ID, Valid: true},
			Tag:       tag,
		})
	}

	// Get updated tags
	tags, _ := toolCtx.DB().GetContactTags(ctx, sql.NullString{String: input.ID, Valid: true})
	tagNames := make([]string, 0, len(tags))
	for _, t := range tags {
		tagNames = append(tagNames, t.Tag)
	}

	return nil, ContactTagsOutput{
		ID:      input.ID,
		Tags:    tagNames,
		Success: true,
		Message: fmt.Sprintf("Added %d tag(s)", len(tagList)),
	}, nil
}

func handleContactRemoveTags(ctx context.Context, toolCtx *mcpctx.ToolContext, input ContactInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}
	if strings.TrimSpace(input.Tags) == "" {
		return nil, nil, mcpctx.NewValidationError("tags is required (comma-separated)", "tags")
	}

	// Verify contact exists
	_, err := toolCtx.DB().GetContactByOrgID(ctx, db.GetContactByOrgIDParams{
		ID:    input.ID,
		OrgID: sql.NullString{String: toolCtx.BrandID(ctx), Valid: true},
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("contact %s not found", input.ID))
	}

	// Remove tags
	tagList := strings.Split(input.Tags, ",")
	removed := 0
	for _, tag := range tagList {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		err := toolCtx.DB().RemoveContactTag(ctx, db.RemoveContactTagParams{
			ContactID: sql.NullString{String: input.ID, Valid: true},
			Tag:       tag,
		})
		if err == nil {
			removed++
		}
	}

	// Get updated tags
	tags, _ := toolCtx.DB().GetContactTags(ctx, sql.NullString{String: input.ID, Valid: true})
	tagNames := make([]string, 0, len(tags))
	for _, t := range tags {
		tagNames = append(tagNames, t.Tag)
	}

	return nil, ContactTagsOutput{
		ID:      input.ID,
		Tags:    tagNames,
		Success: true,
		Message: fmt.Sprintf("Removed %d tag(s)", removed),
	}, nil
}

func handleContactUnsubscribe(ctx context.Context, toolCtx *mcpctx.ToolContext, input ContactInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	// Verify contact exists
	contact, err := toolCtx.DB().GetContactByOrgID(ctx, db.GetContactByOrgIDParams{
		ID:    input.ID,
		OrgID: sql.NullString{String: toolCtx.BrandID(ctx), Valid: true},
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("contact %s not found", input.ID))
	}

	err = toolCtx.DB().GlobalUnsubscribeByOrgAndEmail(ctx, db.GlobalUnsubscribeByOrgAndEmailParams{
		OrgID: sql.NullString{String: toolCtx.BrandID(ctx), Valid: true},
		Email: contact.Email,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unsubscribe contact: %w", err)
	}

	return nil, DeleteOutput{
		Success: true,
		Message: fmt.Sprintf("Contact %s has been unsubscribed globally", contact.Email),
	}, nil
}

func handleContactBlock(ctx context.Context, toolCtx *mcpctx.ToolContext, input ContactInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	// Verify contact exists
	contact, err := toolCtx.DB().GetContactByOrgID(ctx, db.GetContactByOrgIDParams{
		ID:    input.ID,
		OrgID: sql.NullString{String: toolCtx.BrandID(ctx), Valid: true},
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("contact %s not found", input.ID))
	}

	err = toolCtx.DB().BlockContact(ctx, input.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to block contact: %w", err)
	}

	return nil, DeleteOutput{
		Success: true,
		Message: fmt.Sprintf("Contact %s has been blocked", contact.Email),
	}, nil
}

func handleContactUnblock(ctx context.Context, toolCtx *mcpctx.ToolContext, input ContactInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	// Verify contact exists
	contact, err := toolCtx.DB().GetContactByOrgID(ctx, db.GetContactByOrgIDParams{
		ID:    input.ID,
		OrgID: sql.NullString{String: toolCtx.BrandID(ctx), Valid: true},
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("contact %s not found", input.ID))
	}

	err = toolCtx.DB().UnblockContact(ctx, input.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unblock contact: %w", err)
	}

	return nil, DeleteOutput{
		Success: true,
		Message: fmt.Sprintf("Contact %s has been unblocked", contact.Email),
	}, nil
}

func handleContactActivity(ctx context.Context, toolCtx *mcpctx.ToolContext, input ContactInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	// Verify contact exists
	contact, err := toolCtx.DB().GetContactByOrgID(ctx, db.GetContactByOrgIDParams{
		ID:    input.ID,
		OrgID: sql.NullString{String: toolCtx.BrandID(ctx), Valid: true},
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("contact %s not found", input.ID))
	}

	activities := make([]ContactActivityItem, 0)

	// Get sequence emails
	seqEmails, err := toolCtx.DB().GetContactSequenceEmails(ctx, sql.NullString{String: input.ID, Valid: true})
	if err == nil {
		for _, e := range seqEmails {
			activities = append(activities, ContactActivityItem{
				Type:      "sequence_email",
				Subject:   e.Subject.String,
				Name:      e.SequenceName.String,
				Status:    e.Status.String,
				SentAt:    e.SentAt.String,
				OpenedAt:  e.OpenedAt.String,
				ClickedAt: e.ClickedAt.String,
			})
		}
	}

	// Get campaign sends
	campaignSends, err := toolCtx.DB().GetContactCampaignSends(ctx, input.ID)
	if err == nil {
		for _, c := range campaignSends {
			activities = append(activities, ContactActivityItem{
				Type:      "campaign_send",
				Name:      c.CampaignName.String,
				SentAt:    c.SentAt.String,
				OpenedAt:  c.OpenedAt.String,
				ClickedAt: c.ClickedAt.String,
			})
		}
	}

	// Get transactional sends
	txSends, err := toolCtx.DB().GetContactTransactionalSends(ctx, sql.NullString{String: input.ID, Valid: true})
	if err == nil {
		for _, t := range txSends {
			activities = append(activities, ContactActivityItem{
				Type:      "transactional",
				Name:      t.TemplateName.String,
				Status:    t.Status.String,
				SentAt:    t.CreatedAt.String,
				OpenedAt:  t.OpenedAt.String,
				ClickedAt: t.ClickedAt.String,
			})
		}
	}

	// Get link clicks
	clicks, err := toolCtx.DB().GetContactEmailClicks(ctx, sql.NullString{String: input.ID, Valid: true})
	if err == nil {
		for _, c := range clicks {
			activities = append(activities, ContactActivityItem{
				Type:      "click",
				LinkURL:   c.LinkUrl,
				ClickedAt: c.ClickedAt,
			})
		}
	}

	return nil, ContactActivityOutput{
		ID:         contact.ID,
		Email:      contact.Email,
		Activities: activities,
	}, nil
}

// registerContactToolToRegistry registers contact tool to the direct-call registry.
func registerContactToolToRegistry(registry *ToolRegistry, toolCtx *mcpctx.ToolContext) {
	registry.Register("contact", func(ctx context.Context, args json.RawMessage) (interface{}, error) {
		var input ContactInput
		if err := json.Unmarshal(args, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		handler := contactHandler(toolCtx)
		_, output, err := handler(ctx, nil, input)
		return output, err
	})
}
