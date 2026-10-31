package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"strconv"
	"time"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/mcp/mcpctx"
	"github.com/outlet-sh/outlet/internal/services/email"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// campaignActions defines valid actions for campaigns.
var campaignActions = []string{"create", "list", "get", "update", "delete", "schedule", "send", "stats"}

// CampaignInput defines input for the campaign tool.
type CampaignInput struct {
	Action string `json:"action" jsonschema:"required,Action to perform: create, list, get, update, delete, schedule, send, stats"`

	// Common
	ID string `json:"id,omitempty" jsonschema:"Campaign ID (for get, update, delete, schedule, send, stats)"`

	// List filter
	Status string `json:"status,omitempty" jsonschema:"Filter by status: draft, scheduled, sending, sent (for list)"`

	// Create/Update fields
	Name           string `json:"name,omitempty" jsonschema:"Campaign name"`
	Subject        string `json:"subject,omitempty" jsonschema:"Email subject line"`
	PreviewText    string `json:"preview_text,omitempty" jsonschema:"Preview text shown in email clients"`
	FromName       string `json:"from_name,omitempty" jsonschema:"Sender name"`
	FromEmail      string `json:"from_email,omitempty" jsonschema:"Sender email address"`
	ReplyTo        string `json:"reply_to,omitempty" jsonschema:"Reply-to email address"`
	HTMLBody       string `json:"html_body,omitempty" jsonschema:"HTML content of the email"`
	PlainText      string `json:"plain_text,omitempty" jsonschema:"Plain text version of the email"`
	ListIDs        string `json:"list_ids,omitempty" jsonschema:"Comma-separated list IDs to send to"`
	ExcludeListIDs string `json:"exclude_list_ids,omitempty" jsonschema:"Comma-separated list IDs to exclude"`
	TrackOpens     *bool  `json:"track_opens,omitempty" jsonschema:"Track email opens (default: true)"`
	TrackClicks    *bool  `json:"track_clicks,omitempty" jsonschema:"Track link clicks (default: true)"`

	// Schedule fields
	ScheduledAt string `json:"scheduled_at,omitempty" jsonschema:"ISO 8601 datetime to schedule the campaign (for schedule action)"`
}

// CampaignItem represents a campaign in list output.
type CampaignItem struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Subject         string `json:"subject"`
	Status          string `json:"status"`
	RecipientsCount int64  `json:"recipients_count"`
	SentCount       int64  `json:"sent_count"`
	OpenedCount     int64  `json:"opened_count"`
	ClickedCount    int64  `json:"clicked_count"`
	ScheduledAt     string `json:"scheduled_at,omitempty"`
	CreatedAt       string `json:"created_at"`
}

// CampaignListOutput defines output for campaign list.
type CampaignListOutput struct {
	Campaigns []CampaignItem `json:"campaigns"`
	Total     int            `json:"total"`
}

// CampaignCreateOutput defines output for campaign create.
type CampaignCreateOutput struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Subject string `json:"subject"`
	Status  string `json:"status"`
	Created bool   `json:"created"`
}

// CampaignGetOutput defines output for campaign get.
type CampaignGetOutput struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Subject         string `json:"subject"`
	PreviewText     string `json:"preview_text,omitempty"`
	FromName        string `json:"from_name,omitempty"`
	FromEmail       string `json:"from_email,omitempty"`
	ReplyTo         string `json:"reply_to,omitempty"`
	HTMLBody        string `json:"html_body"`
	PlainText       string `json:"plain_text,omitempty"`
	ListIDs         string `json:"list_ids,omitempty"`
	ExcludeListIDs  string `json:"exclude_list_ids,omitempty"`
	Status          string `json:"status"`
	ScheduledAt     string `json:"scheduled_at,omitempty"`
	StartedAt       string `json:"started_at,omitempty"`
	CompletedAt     string `json:"completed_at,omitempty"`
	TrackOpens      bool   `json:"track_opens"`
	TrackClicks     bool   `json:"track_clicks"`
	RecipientsCount int64  `json:"recipients_count"`
	SentCount       int64  `json:"sent_count"`
	DeliveredCount  int64  `json:"delivered_count"`
	OpenedCount     int64  `json:"opened_count"`
	ClickedCount    int64  `json:"clicked_count"`
	BouncedCount    int64  `json:"bounced_count"`
	CreatedAt       string `json:"created_at"`
}

// CampaignUpdateOutput defines output for campaign update.
type CampaignUpdateOutput struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Subject string `json:"subject"`
	Status  string `json:"status"`
	Updated bool   `json:"updated"`
}

// CampaignScheduleOutput defines output for campaign schedule.
type CampaignScheduleOutput struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	ScheduledAt string `json:"scheduled_at"`
	Scheduled   bool   `json:"scheduled"`
}

// CampaignSendOutput defines output for campaign send (immediate).
// Sending starts when the scheduler picks the campaign up; Status says
// where it is now.
type CampaignSendOutput struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	ScheduledAt string `json:"scheduled_at"`
	Message     string `json:"message"`
}

// CampaignStatsOutput defines output for campaign stats.
type CampaignStatsOutput struct {
	ID               string  `json:"id"`
	RecipientsCount  int64   `json:"recipients_count"`
	SentCount        int64   `json:"sent_count"`
	DeliveredCount   int64   `json:"delivered_count"`
	OpenedCount      int64   `json:"opened_count"`
	ClickedCount     int64   `json:"clicked_count"`
	BouncedCount     int64   `json:"bounced_count"`
	ComplainedCount  int64   `json:"complained_count"`
	UnsubscribedCnt  int64   `json:"unsubscribed_count"`
	OpenRate         float64 `json:"open_rate"`
	ClickRate        float64 `json:"click_rate"`
	BounceRate       float64 `json:"bounce_rate"`
}

// RegisterCampaignTool registers the campaign tool.
func RegisterCampaignTool(server *mcp.Server, toolCtx *mcpctx.ToolContext) {
	mcp.AddTool(server, &mcp.Tool{
		Name:  "campaign",
		Title: "Campaign Management",
		Description: `Manage email broadcast campaigns.

PREREQUISITE: You must first select a brand using brand(resource: brand, action: select).

Actions and Required Fields:
- create: Create a new campaign (requires: name, subject, html_body, list_ids)
- list: List all campaigns (optional: status filter)
- get: Get campaign details with full content (requires: id)
- update: Update a draft campaign (requires: id)
- delete: Delete a draft campaign (requires: id)
- schedule: Schedule a campaign for future sending (requires: id, scheduled_at)
- send: Send a campaign now: queues it for the scheduler, which starts within seconds (requires: id)
- stats: Get campaign statistics (requires: id)

Status Values:
- draft: Campaign is being composed
- scheduled: Campaign is scheduled for future sending
- sending: Campaign is currently being sent
- sent: Campaign has been sent

Examples:
  campaign(action: create, name: "January Newsletter", subject: "Happy New Year!", html_body: "<h1>Hello</h1>", list_ids: "1,2")
  campaign(action: list, status: "draft")
  campaign(action: get, id: "uuid")
  campaign(action: update, id: "uuid", subject: "Updated Subject")
  campaign(action: schedule, id: "uuid", scheduled_at: "2024-01-15T10:00:00Z")
  campaign(action: send, id: "uuid")
  campaign(action: stats, id: "uuid")`,
	}, campaignHandler(toolCtx))
}

func campaignHandler(toolCtx *mcpctx.ToolContext) func(ctx context.Context, req *mcp.CallToolRequest, input CampaignInput) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input CampaignInput) (*mcp.CallToolResult, any, error) {
		// Validate action
		if !slices.Contains(campaignActions, input.Action) {
			return nil, nil, mcpctx.NewValidationError(
				fmt.Sprintf("invalid action '%s', must be: %s", input.Action, strings.Join(campaignActions, ", ")),
				"action")
		}

		switch input.Action {
		case "create":
			return handleCampaignCreate(ctx, toolCtx, input)
		case "list":
			return handleCampaignList(ctx, toolCtx, input)
		case "get":
			return handleCampaignGet(ctx, toolCtx, input)
		case "update":
			return handleCampaignUpdate(ctx, toolCtx, input)
		case "delete":
			return handleCampaignDelete(ctx, toolCtx, input)
		case "schedule":
			return handleCampaignSchedule(ctx, toolCtx, input)
		case "send":
			return handleCampaignSend(ctx, toolCtx, input)
		case "stats":
			return handleCampaignStats(ctx, toolCtx, input)
		}
		return nil, nil, nil
	}
}

func handleCampaignCreate(ctx context.Context, toolCtx *mcpctx.ToolContext, input CampaignInput) (*mcp.CallToolResult, any, error) {
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
	if strings.TrimSpace(input.ListIDs) == "" {
		return nil, nil, mcpctx.NewValidationError("list_ids is required", "list_ids")
	}

	if err := email.CheckSender(ctx, toolCtx.DB(), toolCtx.BrandID(ctx), input.FromEmail); err != nil {
		return nil, nil, err
	}
	listIDs, err := access.BrandLists(ctx, toolCtx.DB(), toolCtx.BrandID(ctx), splitIDs(input.ListIDs))
	if err != nil {
		return nil, nil, err
	}
	excludeListIDs, err := access.BrandLists(ctx, toolCtx.DB(), toolCtx.BrandID(ctx), splitIDs(input.ExcludeListIDs))
	if err != nil {
		return nil, nil, err
	}

	trackOpens := true
	if input.TrackOpens != nil {
		trackOpens = *input.TrackOpens
	}
	trackClicks := true
	if input.TrackClicks != nil {
		trackClicks = *input.TrackClicks
	}

	campaignID := uuid.New().String()
	campaign, err := toolCtx.DB().CreateCampaign(ctx, db.CreateCampaignParams{
		ID:             campaignID,
		OrgID:          toolCtx.BrandID(ctx),
		Name:           input.Name,
		Subject:        input.Subject,
		PreviewText:    sql.NullString{String: input.PreviewText, Valid: input.PreviewText != ""},
		FromName:       sql.NullString{String: input.FromName, Valid: input.FromName != ""},
		FromEmail:      sql.NullString{String: input.FromEmail, Valid: input.FromEmail != ""},
		ReplyTo:        sql.NullString{String: input.ReplyTo, Valid: input.ReplyTo != ""},
		HtmlBody:       input.HTMLBody,
		PlainText:      sql.NullString{String: input.PlainText, Valid: input.PlainText != ""},
		ListIds:        sql.NullString{String: db.EncodeListIDs(listIDs), Valid: true},
		ExcludeListIds: sql.NullString{String: db.EncodeListIDs(excludeListIDs), Valid: len(excludeListIDs) > 0},
		Status:         sql.NullString{String: "draft", Valid: true},
		TrackOpens:     sql.NullInt64{Int64: boolToInt64(trackOpens), Valid: true},
		TrackClicks:    sql.NullInt64{Int64: boolToInt64(trackClicks), Valid: true},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create campaign: %w", err)
	}

	return nil, CampaignCreateOutput{
		ID:      campaign.ID,
		Name:    campaign.Name,
		Subject: campaign.Subject,
		Status:  campaign.Status.String,
		Created: true,
	}, nil
}

func handleCampaignList(ctx context.Context, toolCtx *mcpctx.ToolContext, input CampaignInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	var campaigns []db.EmailCampaign
	var err error

	if input.Status != "" {
		campaigns, err = toolCtx.DB().ListCampaignsByStatus(ctx, db.ListCampaignsByStatusParams{
			OrgID:      toolCtx.BrandID(ctx),
			Status:     sql.NullString{String: input.Status, Valid: true},
			PageOffset: 0,
			PageSize:   100,
		})
	} else {
		campaigns, err = toolCtx.DB().ListCampaigns(ctx, db.ListCampaignsParams{
			OrgID:      toolCtx.BrandID(ctx),
			PageOffset: 0,
			PageSize:   100,
		})
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list campaigns: %w", err)
	}

	items := make([]CampaignItem, 0, len(campaigns))
	for _, c := range campaigns {
		items = append(items, CampaignItem{
			ID:              c.ID,
			Name:            c.Name,
			Subject:         c.Subject,
			Status:          c.Status.String,
			RecipientsCount: c.RecipientsCount.Int64,
			SentCount:       c.SentCount.Int64,
			OpenedCount:     c.OpenedCount.Int64,
			ClickedCount:    c.ClickedCount.Int64,
			ScheduledAt:     c.ScheduledAt.String,
			CreatedAt:       c.CreatedAt.String,
		})
	}

	return nil, CampaignListOutput{
		Campaigns: items,
		Total:     len(items),
	}, nil
}

func handleCampaignGet(ctx context.Context, toolCtx *mcpctx.ToolContext, input CampaignInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	campaign, err := toolCtx.DB().GetCampaign(ctx, db.GetCampaignParams{
		ID:    input.ID,
		OrgID: toolCtx.BrandID(ctx),
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("campaign %s not found", input.ID))
	}

	return nil, CampaignGetOutput{
		ID:              campaign.ID,
		Name:            campaign.Name,
		Subject:         campaign.Subject,
		PreviewText:     campaign.PreviewText.String,
		FromName:        campaign.FromName.String,
		FromEmail:       campaign.FromEmail.String,
		ReplyTo:         campaign.ReplyTo.String,
		HTMLBody:        campaign.HtmlBody,
		PlainText:       campaign.PlainText.String,
		ListIDs:         joinIDs(db.DecodeListIDs(campaign.ListIds.String)),
		ExcludeListIDs:  joinIDs(db.DecodeListIDs(campaign.ExcludeListIds.String)),
		Status:          campaign.Status.String,
		ScheduledAt:     campaign.ScheduledAt.String,
		StartedAt:       campaign.StartedAt.String,
		CompletedAt:     campaign.CompletedAt.String,
		TrackOpens:      int64ToBool(campaign.TrackOpens),
		TrackClicks:     int64ToBool(campaign.TrackClicks),
		RecipientsCount: campaign.RecipientsCount.Int64,
		SentCount:       campaign.SentCount.Int64,
		DeliveredCount:  campaign.DeliveredCount.Int64,
		OpenedCount:     campaign.OpenedCount.Int64,
		ClickedCount:    campaign.ClickedCount.Int64,
		BouncedCount:    campaign.BouncedCount.Int64,
		CreatedAt:       campaign.CreatedAt.String,
	}, nil
}

func handleCampaignUpdate(ctx context.Context, toolCtx *mcpctx.ToolContext, input CampaignInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	// Check campaign exists and is in draft status
	campaign, err := toolCtx.DB().GetCampaign(ctx, db.GetCampaignParams{
		ID:    input.ID,
		OrgID: toolCtx.BrandID(ctx),
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("campaign %s not found", input.ID))
	}

	if campaign.Status.String != "draft" {
		return nil, nil, mcpctx.NewValidationError("can only update draft campaigns", "id")
	}

	if err := email.CheckSender(ctx, toolCtx.DB(), toolCtx.BrandID(ctx), input.FromEmail); err != nil {
		return nil, nil, err
	}
	var listIDs, excludeListIDs interface{}
	if strings.TrimSpace(input.ListIDs) != "" {
		ids, err := access.BrandLists(ctx, toolCtx.DB(), toolCtx.BrandID(ctx), splitIDs(input.ListIDs))
		if err != nil {
			return nil, nil, err
		}
		listIDs = db.EncodeListIDs(ids)
	}
	if strings.TrimSpace(input.ExcludeListIDs) != "" {
		ids, err := access.BrandLists(ctx, toolCtx.DB(), toolCtx.BrandID(ctx), splitIDs(input.ExcludeListIDs))
		if err != nil {
			return nil, nil, err
		}
		excludeListIDs = db.EncodeListIDs(ids)
	}

	var trackOpens, trackClicks sql.NullInt64
	if input.TrackOpens != nil {
		trackOpens = sql.NullInt64{Int64: boolToInt64(*input.TrackOpens), Valid: true}
	}
	if input.TrackClicks != nil {
		trackClicks = sql.NullInt64{Int64: boolToInt64(*input.TrackClicks), Valid: true}
	}

	updated, err := toolCtx.DB().UpdateCampaign(ctx, db.UpdateCampaignParams{
		ID:             input.ID,
		OrgID:          toolCtx.BrandID(ctx),
		Name:           input.Name,
		Subject:        input.Subject,
		PreviewText:    input.PreviewText,
		FromName:       input.FromName,
		FromEmail:      input.FromEmail,
		ReplyTo:        input.ReplyTo,
		HtmlBody:       input.HTMLBody,
		PlainText:      input.PlainText,
		ListIds:        listIDs,
		ExcludeListIds: excludeListIDs,
		TrackOpens:     trackOpens,
		TrackClicks:    trackClicks,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update campaign: %w", err)
	}

	return nil, CampaignUpdateOutput{
		ID:      updated.ID,
		Name:    updated.Name,
		Subject: updated.Subject,
		Status:  updated.Status.String,
		Updated: true,
	}, nil
}

func handleCampaignDelete(ctx context.Context, toolCtx *mcpctx.ToolContext, input CampaignInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	// Check campaign exists
	_, err := toolCtx.DB().GetCampaign(ctx, db.GetCampaignParams{
		ID:    input.ID,
		OrgID: toolCtx.BrandID(ctx),
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("campaign %s not found", input.ID))
	}

	err = toolCtx.DB().DeleteCampaign(ctx, db.DeleteCampaignParams{
		ID:    input.ID,
		OrgID: toolCtx.BrandID(ctx),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to delete campaign (only draft campaigns can be deleted): %w", err)
	}

	return nil, DeleteOutput{
		Success: true,
		Message: fmt.Sprintf("Campaign %s deleted successfully", input.ID),
	}, nil
}

func handleCampaignSchedule(ctx context.Context, toolCtx *mcpctx.ToolContext, input CampaignInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}
	if strings.TrimSpace(input.ScheduledAt) == "" {
		return nil, nil, mcpctx.NewValidationError("scheduled_at is required (ISO 8601 format)", "scheduled_at")
	}

	scheduledAt, err := time.Parse(time.RFC3339, input.ScheduledAt)
	if err != nil {
		return nil, nil, mcpctx.NewValidationError("scheduled_at must be ISO 8601, like 2026-01-15T10:00:00Z", "scheduled_at")
	}

	campaign, err := toolCtx.DB().ScheduleCampaign(ctx, db.ScheduleCampaignParams{
		ID:          input.ID,
		OrgID:       toolCtx.BrandID(ctx),
		ScheduledAt: sql.NullString{String: scheduledAt.UTC().Format(time.RFC3339), Valid: true},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to schedule campaign (only draft campaigns can be scheduled): %w", err)
	}

	return nil, CampaignScheduleOutput{
		ID:          campaign.ID,
		Status:      campaign.Status.String,
		ScheduledAt: campaign.ScheduledAt.String,
		Scheduled:   true,
	}, nil
}

func handleCampaignSend(ctx context.Context, toolCtx *mcpctx.ToolContext, input CampaignInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	// Get campaign and verify it's in draft status
	campaign, err := toolCtx.DB().GetCampaign(ctx, db.GetCampaignParams{
		ID:    input.ID,
		OrgID: toolCtx.BrandID(ctx),
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("campaign %s not found", input.ID))
	}

	if campaign.Status.String != "draft" {
		return nil, nil, mcpctx.NewValidationError(
			fmt.Sprintf("campaign is in '%s' status, only draft campaigns can be sent", campaign.Status.String),
			"id")
	}

	// Schedule it for now: the scheduler is the one sending path.
	scheduled, err := toolCtx.DB().ScheduleCampaign(ctx, db.ScheduleCampaignParams{
		ID:    input.ID,
		OrgID: toolCtx.BrandID(ctx),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to queue campaign: %w", err)
	}

	return nil, CampaignSendOutput{
		ID:          scheduled.ID,
		Status:      scheduled.Status.String,
		ScheduledAt: scheduled.ScheduledAt.String,
		Message:     "Queued to send now. The scheduler starts sending within seconds; check stats for progress.",
	}, nil
}

func handleCampaignStats(ctx context.Context, toolCtx *mcpctx.ToolContext, input CampaignInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ID) == "" {
		return nil, nil, mcpctx.NewValidationError("id is required", "id")
	}

	campaign, err := toolCtx.DB().GetCampaign(ctx, db.GetCampaignParams{
		ID:    input.ID,
		OrgID: toolCtx.BrandID(ctx),
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("campaign %s not found", input.ID))
	}

	// Calculate rates
	var openRate, clickRate, bounceRate float64
	if campaign.SentCount.Int64 > 0 {
		openRate = float64(campaign.OpenedCount.Int64) / float64(campaign.SentCount.Int64) * 100
		clickRate = float64(campaign.ClickedCount.Int64) / float64(campaign.SentCount.Int64) * 100
		bounceRate = float64(campaign.BouncedCount.Int64) / float64(campaign.SentCount.Int64) * 100
	}

	return nil, CampaignStatsOutput{
		ID:              campaign.ID,
		RecipientsCount: campaign.RecipientsCount.Int64,
		SentCount:       campaign.SentCount.Int64,
		DeliveredCount:  campaign.DeliveredCount.Int64,
		OpenedCount:     campaign.OpenedCount.Int64,
		ClickedCount:    campaign.ClickedCount.Int64,
		BouncedCount:    campaign.BouncedCount.Int64,
		ComplainedCount: campaign.ComplainedCount.Int64,
		UnsubscribedCnt: campaign.UnsubscribedCount.Int64,
		OpenRate:        openRate,
		ClickRate:       clickRate,
		BounceRate:      bounceRate,
	}, nil
}

// registerCampaignToolToRegistry registers campaign tool to the direct-call registry.
func registerCampaignToolToRegistry(registry *ToolRegistry, toolCtx *mcpctx.ToolContext) {
	registry.Register("campaign", func(ctx context.Context, args json.RawMessage) (interface{}, error) {
		var input CampaignInput
		if err := json.Unmarshal(args, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		handler := campaignHandler(toolCtx)
		_, output, err := handler(ctx, nil, input)
		return output, err
	})
}

// splitIDs reads a comma-separated list of IDs, as the MCP tools take them.
func splitIDs(s string) []string {
	var ids []string
	for _, id := range strings.Split(s, ",") {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// joinIDs writes IDs comma-separated, as the MCP tools return them.
func joinIDs(ids []int64) string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, strconv.FormatInt(id, 10))
	}
	return strings.Join(out, ",")
}
