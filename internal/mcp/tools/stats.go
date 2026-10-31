package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/mcp/mcpctx"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// statsActions defines valid actions for each stats resource.
var statsActions = map[string][]string{
	"overview": {"get"},
	"email":    {"get"},
	"contact":  {"get"},
}

// StatsInput defines input for the stats tool.
type StatsInput struct {
	Resource string `json:"resource" jsonschema:"required,Resource type: overview, email, or contact"`
	Action   string `json:"action" jsonschema:"required,Action to perform: get"`

	// Overview/Email filters
	Days       int    `json:"days,omitempty" jsonschema:"Number of days to include (default: 30, max: 90)"`
	SequenceID string `json:"sequence_id,omitempty" jsonschema:"Filter by sequence ID (email.get)"`

	// Contact filters
	ContactID string `json:"contact_id,omitempty" jsonschema:"Contact ID (contact.get)"`
}

// RegisterStatsTool registers the stats tool.
func RegisterStatsTool(server *mcp.Server, toolCtx *mcpctx.ToolContext) {
	mcp.AddTool(server, &mcp.Tool{
		Name:  "stats",
		Title: "Statistics and Analytics",
		Description: `Get statistics and analytics for your brand.

PREREQUISITE: You must first select a brand using brand(resource: brand, action: select).

Resources and Actions:
- overview.get: Get overall brand statistics
- email.get: Get email performance statistics (optional: sequence_id)
- contact.get: Get statistics for a specific contact (requires: contact_id)

Examples:
  stats(resource: overview, action: get)
  stats(resource: email, action: get)
  stats(resource: email, action: get, sequence_id: "uuid")
  stats(resource: contact, action: get, contact_id: "uuid")`,
	}, statsHandler(toolCtx))
}

func statsHandler(toolCtx *mcpctx.ToolContext) func(ctx context.Context, req *mcp.CallToolRequest, input StatsInput) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input StatsInput) (*mcp.CallToolResult, any, error) {
		// Validate resource
		validActions, ok := statsActions[input.Resource]
		if !ok {
			return nil, nil, mcpctx.NewValidationError(
				fmt.Sprintf("invalid resource '%s', must be: overview, email, or contact", input.Resource),
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
		case "overview":
			return handleOverviewStats(ctx, toolCtx, input)
		case "email":
			return handleEmailStats(ctx, toolCtx, input)
		case "contact":
			return handleContactStats(ctx, toolCtx, input)
		}
		return nil, nil, nil // unreachable
	}
}

// ============================================================================
// OUTPUT TYPES
// ============================================================================

// OverviewStatsOutput defines output for overview.get.
type OverviewStatsOutput struct {
	TotalContacts   int64 `json:"total_contacts"`
	NewContacts30d  int64 `json:"new_contacts_30d"`
	NewContacts7d   int64 `json:"new_contacts_7d"`
	EmailsSent30d   int64 `json:"emails_sent_30d"`
	Opens30d        int64 `json:"opens_30d"`
	Clicks30d       int64 `json:"clicks_30d"`
}

// EmailStatsOutput defines output for email.get.
type EmailStatsOutput struct {
	Sent      int64   `json:"sent"`
	Opened    int64   `json:"opened"`
	Clicked   int64   `json:"clicked"`
	OpenRate  float64 `json:"open_rate"`
	ClickRate float64 `json:"click_rate"`
}

// ContactStatsOutput defines output for contact.get.
type ContactStatsOutput struct {
	ContactID         string `json:"contact_id"`
	Email             string `json:"email"`
	CampaignsSent     int64  `json:"campaigns_sent"`
	SequencesEnrolled int64  `json:"sequences_enrolled"`
	TransactionalSent int64  `json:"transactional_sent"`
	TotalOpens        int64  `json:"total_opens"`
	TotalClicks       int64  `json:"total_clicks"`
}

// ============================================================================
// HANDLERS
// ============================================================================

func handleOverviewStats(ctx context.Context, toolCtx *mcpctx.ToolContext, input StatsInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	brandID := toolCtx.BrandID(ctx)

	// Get subscriber stats
	var totalContacts, new30d, new7d int64
	subscriberStats, err := toolCtx.DB().GetDashboardSubscriberStats(ctx, sql.NullString{String: brandID, Valid: true})
	if err == nil {
		totalContacts = subscriberStats.Total
		if subscriberStats.New30d.Valid {
			new30d = int64(subscriberStats.New30d.Float64)
		}
		if subscriberStats.New7d.Valid {
			new7d = int64(subscriberStats.New7d.Float64)
		}
	}

	// Get email stats (30 days)
	var emailsSent, opens, clicks int64
	emailStats, err := toolCtx.DB().GetDashboardEmailStats30Days(ctx, brandID)
	if err == nil {
		emailsSent = interfaceToInt64(emailStats.EmailsSent)
		opens = interfaceToInt64(emailStats.EmailsOpened)
		clicks = interfaceToInt64(emailStats.EmailsClicked)
	}

	return nil, OverviewStatsOutput{
		TotalContacts:   totalContacts,
		NewContacts30d:  new30d,
		NewContacts7d:   new7d,
		EmailsSent30d:   emailsSent,
		Opens30d:        opens,
		Clicks30d:       clicks,
	}, nil
}

func handleEmailStats(ctx context.Context, toolCtx *mcpctx.ToolContext, input StatsInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	var sent, opened, clicked int64

	// Get stats based on filters
	if input.SequenceID != "" {
		if err := sameBrand(ctx, toolCtx, access.Sequence, input.SequenceID, "sequence"); err != nil {
			return nil, nil, err
		}
		// Get sequence-specific stats
		stats, err := toolCtx.DB().GetEmailStatsForSequence(ctx, sql.NullString{String: input.SequenceID, Valid: true})
		if err == nil {
			if stats.SentCount.Valid {
				sent = int64(stats.SentCount.Float64)
			}
			if stats.OpenedCount.Valid {
				opened = int64(stats.OpenedCount.Float64)
			}
			if stats.ClickedCount.Valid {
				clicked = int64(stats.ClickedCount.Float64)
			}
		}
	} else {
		// Get overall email stats for date range
		emailStats, err := toolCtx.DB().GetDashboardEmailStats30Days(ctx, toolCtx.BrandID(ctx))
		if err == nil {
			sent = interfaceToInt64(emailStats.EmailsSent)
			opened = interfaceToInt64(emailStats.EmailsOpened)
			clicked = interfaceToInt64(emailStats.EmailsClicked)
		}
	}

	// Calculate rates
	var openRate, clickRate float64
	if sent > 0 {
		openRate = float64(opened) / float64(sent) * 100
		clickRate = float64(clicked) / float64(sent) * 100
	}

	return nil, EmailStatsOutput{
		Sent:      sent,
		Opened:    opened,
		Clicked:   clicked,
		OpenRate:  openRate,
		ClickRate: clickRate,
	}, nil
}

func handleContactStats(ctx context.Context, toolCtx *mcpctx.ToolContext, input StatsInput) (*mcp.CallToolResult, any, error) {
	if err := toolCtx.RequireBrand(ctx); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.ContactID) == "" {
		return nil, nil, mcpctx.NewValidationError("contact_id is required", "contact_id")
	}

	// Get contact to verify access and get email
	contact, err := toolCtx.DB().GetContactByOrgID(ctx, db.GetContactByOrgIDParams{
		ID:    input.ContactID,
		OrgID: sql.NullString{String: toolCtx.BrandID(ctx), Valid: true},
	})
	if err != nil {
		return nil, nil, mcpctx.NewNotFoundError(fmt.Sprintf("contact %s not found", input.ContactID))
	}

	// Get campaign sends count
	campaignSends, _ := toolCtx.DB().GetContactCampaignSends(ctx, contact.ID)
	campaignsSent := int64(len(campaignSends))

	// Get sequence enrollments count
	sequenceStates, _ := toolCtx.DB().ListContactSequenceStatesWithDetails(ctx, db.ListContactSequenceStatesWithDetailsParams{
		ContactID: sql.NullString{String: contact.ID, Valid: true},
		OrgID:     sql.NullString{String: toolCtx.BrandID(ctx), Valid: true},
	})
	sequencesEnrolled := int64(len(sequenceStates))

	// Get transactional sends count
	transactionalSends, _ := toolCtx.DB().GetContactTransactionalSends(ctx, sql.NullString{String: contact.ID, Valid: true})
	transactionalSent := int64(len(transactionalSends))

	// Get click/open counts
	clicks, _ := toolCtx.DB().GetContactEmailClicks(ctx, sql.NullString{String: contact.ID, Valid: true})
	totalClicks := int64(len(clicks))

	// Estimate opens from various sources
	var totalOpens int64
	for _, send := range campaignSends {
		if send.OpenedAt.Valid {
			totalOpens++
		}
	}
	for _, send := range transactionalSends {
		if send.OpenedAt.Valid {
			totalOpens++
		}
	}

	return nil, ContactStatsOutput{
		ContactID:         contact.ID,
		Email:             contact.Email,
		CampaignsSent:     campaignsSent,
		SequencesEnrolled: sequencesEnrolled,
		TransactionalSent: transactionalSent,
		TotalOpens:        totalOpens,
		TotalClicks:       totalClicks,
	}, nil
}

// interfaceToInt64 safely converts interface{} to int64.
func interfaceToInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case float64:
		return int64(val)
	case float32:
		return int64(val)
	default:
		return 0
	}
}

// registerStatsToolToRegistry registers stats tool to the direct-call registry.
func registerStatsToolToRegistry(registry *ToolRegistry, toolCtx *mcpctx.ToolContext) {
	registry.Register("stats", func(ctx context.Context, args json.RawMessage) (interface{}, error) {
		var input StatsInput
		if err := json.Unmarshal(args, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		handler := statsHandler(toolCtx)
		_, output, err := handler(ctx, nil, input)
		return output, err
	})
}
