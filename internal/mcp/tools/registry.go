package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/outlet-sh/outlet/internal/mcp/mcpauth"
	"github.com/outlet-sh/outlet/internal/mcp/mcpctx"
)

// ToolFunc is a function that executes a tool with JSON args.
// This allows tools to be called directly without going through the MCP protocol.
type ToolFunc func(ctx context.Context, args json.RawMessage) (interface{}, error)

// ToolRegistry stores tool handlers for direct invocation. Calls pass the
// same gate as MCP calls, as the tool context's user.
type ToolRegistry struct {
	mu      sync.RWMutex
	tools   map[string]ToolFunc
	toolCtx *mcpctx.ToolContext
}

// NewToolRegistry creates a new tool registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]ToolFunc),
	}
}

// Register adds a tool to the registry.
func (r *ToolRegistry) Register(name string, fn ToolFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[name] = fn
}

// Call invokes a tool by name with the given arguments.
func (r *ToolRegistry) Call(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	r.mu.RLock()
	fn, ok := r.tools[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal args: %w", err)
	}

	if r.toolCtx == nil || r.toolCtx.User() == nil {
		return nil, fmt.Errorf("tool registry has no signed-in user")
	}
	user := r.toolCtx.User()
	caller := Caller{UserID: user.ID, Email: user.Email, Scopes: []string{mcpauth.ScopeFull}}
	var out interface{}
	var callErr error
	if err := gated(ctx, r.toolCtx, caller, name, argsJSON, func() string {
		out, callErr = fn(ctx, argsJSON)
		return outcome(nil, callErr)
	}); err != nil {
		return nil, err
	}
	return out, callErr
}

// Has returns true if the registry has a tool with the given name.
func (r *ToolRegistry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.tools[name]
	return ok
}

// List returns all registered tool names.
func (r *ToolRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// NewRegistryWithTools creates a registry with all tools registered.
// This is used by the agent executor to call tools directly.
func NewRegistryWithTools(toolCtx *mcpctx.ToolContext) *ToolRegistry {
	registry := NewToolRegistry()
	registry.toolCtx = toolCtx

	// Register all tool handlers (unified resource/action pattern)
	registerEmailToolToRegistry(registry, toolCtx)
	registerBrandToolToRegistry(registry, toolCtx)
	registerCampaignToolToRegistry(registry, toolCtx)
	registerContactToolToRegistry(registry, toolCtx)
	registerWebhookToolToRegistry(registry, toolCtx)
	registerDesignToolToRegistry(registry, toolCtx)
	registerTransactionalToolToRegistry(registry, toolCtx)
	registerStatsToolToRegistry(registry, toolCtx)
	registerBlocklistToolToRegistry(registry, toolCtx)
	registerGDPRToolToRegistry(registry, toolCtx)

	return registry
}
