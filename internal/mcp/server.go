package mcp

import (
	"fmt"
	"net/http"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/mcp/mcpauth"
	"github.com/outlet-sh/outlet/internal/mcp/mcpctx"
	"github.com/outlet-sh/outlet/internal/mcp/tools"
	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewServer creates a new user-scoped MCP server with all tools registered.
// The user must use brand.select to choose which brand to work with.
// This is a convenience wrapper around NewServerWithContext that discards the toolCtx.
func NewServer(svc *svc.ServiceContext, r *http.Request) *mcp.Server {
	server, _ := NewServerWithContext(svc, r, nil)
	return server
}

// NewServerWithContext creates a new MCP server and returns both the server and the ToolContext.
// The ToolContext is returned so the caller can cache it for session persistence.
// The onBrandSelect callback is called when brand.select is used, for persisting the selection.
func NewServerWithContext(svc *svc.ServiceContext, r *http.Request, onBrandSelect mcpctx.BrandSelectionCallback) (*mcp.Server, *mcpctx.ToolContext) {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "outlet",
		Version: "1.0.0",
	}, nil)

	// Get token info from our custom auth middleware
	tokenInfo := mcpauth.TokenInfoFromContext(r.Context())
	if tokenInfo == nil {
		// No auth context - return server without tools
		fmt.Println("[MCP] No token info in context - returning server without tools")
		return server, nil
	}

	// Extract user info from token
	userInfo, ok := tokenInfo.Extra["user_info"].(*mcpauth.UserInfo)
	if !ok {
		fmt.Println("[MCP] Failed to extract user_info from token")
		return server, nil
	}

	user, ok := tokenInfo.Extra["user"].(*db.User)
	if !ok {
		fmt.Println("[MCP] Failed to extract user from token")
		return server, nil
	}

	sessionID := r.Header.Get("Mcp-Session-Id")
	fmt.Printf("[MCP] Creating server for user: %s (ID: %s, AuthMode: %s, Session: %s)\n", user.Email, user.ID, userInfo.AuthMode, sessionID)

	// Generate request ID for tracing
	requestID := uuid.New().String()
	userAgent := r.Header.Get("User-Agent")

	// Create user-scoped tool context
	// User must call brand.select to choose a brand - state persists in the session
	toolCtx := mcpctx.NewUserToolContext(svc, *user, requestID, userAgent, sessionID)

	// Set callback for persisting brand selection
	if onBrandSelect != nil {
		toolCtx.SetBrandSelectionCallback(onBrandSelect)
	}

	// Store additional user info in context for tools that need it
	_ = userInfo // Available via toolCtx.User() methods

	// Register all tools (unified resource/action pattern)
	tools.RegisterEmailTool(server, toolCtx)
	tools.RegisterBrandTool(server, toolCtx)
	tools.RegisterCampaignTool(server, toolCtx)
	tools.RegisterContactTool(server, toolCtx)
	tools.RegisterWebhookTool(server, toolCtx)
	tools.RegisterDesignTool(server, toolCtx)
	tools.RegisterTransactionalTool(server, toolCtx)
	tools.RegisterStatsTool(server, toolCtx)
	tools.RegisterBlocklistTool(server, toolCtx)
	tools.RegisterGDPRTool(server, toolCtx)

	// Every tool call passes the gate: scopes, brand access, audit.
	server.AddReceivingMiddleware(tools.Gate(toolCtx))

	return server, toolCtx
}
