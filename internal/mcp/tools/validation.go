package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/outlet-sh/outlet/internal/mcp/mcpctx"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// validationActions defines valid actions for each validation resource.
var validationActions = map[string][]string{
	"email": {"validate"},
	"job":   {"create", "get"},
}

// ValidationInput defines input for the validation tool.
type ValidationInput struct {
	Resource string `json:"resource" jsonschema:"required,Resource type: email or job"`
	Action   string `json:"action" jsonschema:"required,Action to perform: validate, create, get"`

	Email  string `json:"email,omitempty" jsonschema:"Email address to validate (email.validate)"`
	ListID string `json:"list_id,omitempty" jsonschema:"List ID whose active subscribers to validate (job.create)"`
	Name   string `json:"name,omitempty" jsonschema:"Optional job name (job.create)"`
	JobID  string `json:"job_id,omitempty" jsonschema:"Validation job ID (job.get)"`
}

// RegisterValidationTool registers the validation tool.
func RegisterValidationTool(server *mcp.Server, toolCtx *mcpctx.ToolContext) {
	mcp.AddTool(server, &mcp.Tool{
		Name:  "validation",
		Title: "Email Validation",
		Description: `Validate email addresses (ZeroBounce-compatible statuses: valid, invalid, catch-all, unknown, spamtrap, abuse, do_not_mail).

PREREQUISITE: You must first select a brand using brand(resource: brand, action: select).

Resources and Actions:
- email.validate: Validate one address now (requires: email). Counts toward validation usage.
- job.create: Start a background job validating a list's active subscribers (requires: list_id, optional: name)
- job.get: Get a job's status and per-status breakdown (requires: job_id)

Examples:
  validation(resource: email, action: validate, email: "jane@example.com")
  validation(resource: job, action: create, list_id: "12")
  validation(resource: job, action: get, job_id: "uuid")`,
	}, validationHandler(toolCtx))
}

func validationHandler(toolCtx *mcpctx.ToolContext) func(ctx context.Context, req *mcp.CallToolRequest, input ValidationInput) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input ValidationInput) (*mcp.CallToolResult, any, error) {
		validActions, ok := validationActions[input.Resource]
		if !ok {
			return nil, nil, mcpctx.NewValidationError(
				fmt.Sprintf("invalid resource '%s', must be: email or job", input.Resource),
				"resource")
		}
		if !slices.Contains(validActions, input.Action) {
			return nil, nil, mcpctx.NewValidationError(
				fmt.Sprintf("invalid action '%s' for resource '%s', must be: %s",
					input.Action, input.Resource, strings.Join(validActions, ", ")),
				"action")
		}
		if err := toolCtx.RequireBrand(ctx); err != nil {
			return nil, nil, err
		}

		svc := toolCtx.Svc().Validation
		brandID := toolCtx.BrandID(ctx)
		var (
			out any
			err error
		)
		switch input.Resource + "." + input.Action {
		case "email.validate":
			if input.Email == "" {
				return nil, nil, mcpctx.NewValidationError("email is required", "email")
			}
			out, err = svc.Validate(ctx, brandID, input.Email, "mcp")
		case "job.create":
			if input.ListID == "" {
				return nil, nil, mcpctx.NewValidationError("list_id is required", "list_id")
			}
			out, err = svc.CreateJob(ctx, brandID, input.Name, input.ListID, nil)
		case "job.get":
			if input.JobID == "" {
				return nil, nil, mcpctx.NewValidationError("job_id is required", "job_id")
			}
			out, err = svc.Job(ctx, brandID, input.JobID)
		}
		if err != nil {
			return nil, nil, err
		}
		return nil, out, nil
	}
}

func registerValidationToolToRegistry(registry *ToolRegistry, toolCtx *mcpctx.ToolContext) {
	registry.Register("validation", func(ctx context.Context, args json.RawMessage) (interface{}, error) {
		var input ValidationInput
		if err := json.Unmarshal(args, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		handler := validationHandler(toolCtx)
		_, output, err := handler(ctx, nil, input)
		return output, err
	})
}
