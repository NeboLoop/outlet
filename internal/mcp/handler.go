package mcp

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/mcp/mcpauth"
	"github.com/outlet-sh/outlet/internal/mcp/mcpctx"
	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Handler handles MCP HTTP requests with Bearer token authentication.
// Supports both API keys (lv_xxx) and OAuth access tokens.
type Handler struct {
	svc             *svc.ServiceContext
	authenticator   *mcpauth.Authenticator
	httpHandler     http.Handler
	resourceMetaURL string

	// sessionCache stores MCP servers + ToolContext by session ID (in-memory).
	// Each session has its own server with its own ToolContext (brand selection).
	// This allows multiple clients per user with different brands.
	sessionCache sync.Map // map[sessionID]*sessionData

	// brandSelectionCache is an in-memory cache of brand selections.
	// Backed by mcp_sessions DB table for persistence across restarts.
	brandSelectionCache sync.Map // map[sessionID]brandSelection
}

// brandSelection is the brand a session's owner chose.
type brandSelection struct {
	userID  string
	brandID string
}

// sessionData holds cached session data.
type sessionData struct {
	server  *mcp.Server
	toolCtx *mcpctx.ToolContext
	userID  string // owner of this session, validated on cache hits
}

// NewHandler creates a new MCP handler with authentication.
// baseURL is used to construct the resource metadata URL for OAuth discovery.
func NewHandler(svc *svc.ServiceContext, baseURL string) *Handler {
	baseURL = strings.TrimSuffix(baseURL, "/")

	h := &Handler{
		svc:             svc,
		authenticator:   mcpauth.NewAuthenticator(svc),
		// Point to root URL - ChatGPT may expect root discovery then follow to /mcp
		resourceMetaURL: baseURL + "/.well-known/oauth-protected-resource",
	}

	// Create the streamable HTTP handler in STATELESS mode.
	// Stateless mode means the SDK doesn't validate session IDs - we handle it ourselves.
	// This allows sessions to survive server restarts because:
	// 1. Client continues using their old session ID after restart
	// 2. SDK doesn't reject it (stateless)
	// 3. We restore brand selection from our persistent brandSelectionStore
	streamHandler := mcp.NewStreamableHTTPHandler(
		h.getServerForRequest,
		&mcp.StreamableHTTPOptions{
			Stateless: true, // Don't track sessions in SDK - we do it ourselves
		},
	)

	// Wrap with our custom auth middleware that properly formats WWW-Authenticate
	h.httpHandler = h.authMiddleware(streamHandler)

	return h
}

// authMiddleware validates Bearer tokens and returns proper OAuth challenge on 401.
// This custom implementation ensures WWW-Authenticate header is RFC 9728 compliant
// with quoted resource_metadata value.
// It also ensures session IDs are generated and communicated back to the client.
func (h *Handler) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for session ID - generate one if not provided
		sessionID := r.Header.Get("Mcp-Session-Id")
		newSession := false
		if sessionID == "" {
			sessionID = uuid.New().String()
			newSession = true
			// Add session ID to request so getServerForRequest can use it
			r.Header.Set("Mcp-Session-Id", sessionID)
		}

		fmt.Printf("[MCP DEBUG] %s %s | Session: %q (new: %v) | Accept: %s\n",
			r.Method, r.URL.Path, sessionID, newSession, r.Header.Get("Accept"))

		// Extract Bearer token
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			h.writeUnauthorized(w, "missing bearer token")
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			h.writeUnauthorized(w, "empty bearer token")
			return
		}

		// Verify token using our authenticator (TokenVerifier returns a func)
		tokenInfo, err := h.authenticator.TokenVerifier()(r.Context(), token, r)
		if err != nil {
			h.writeUnauthorized(w, "invalid token")
			return
		}

		// Always set session ID in response header so client can use it for subsequent requests
		// This is critical for MCP session continuity - the client needs to know the session ID
		w.Header().Set("Mcp-Session-Id", sessionID)

		// Add token info to request context and continue
		ctx := mcpauth.ContextWithTokenInfo(r.Context(), tokenInfo)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// writeUnauthorized sends a 401 response with WWW-Authenticate header for OAuth discovery.
// Uses simple format matching working MCP OAuth implementations.
func (h *Handler) writeUnauthorized(w http.ResponseWriter, msg string) {
	// Bearer challenge with resource_metadata URL and scope
	wwwAuth := fmt.Sprintf(`Bearer resource_metadata="%s", scope="mcp:full"`, h.resourceMetaURL)
	w.Header().Set("WWW-Authenticate", wwwAuth)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

// getServerForRequest returns a cached server for the session, or creates a new one.
// We cache by SESSION ID to support multiple clients per user with different brands.
// In stateless mode, the SDK doesn't track sessions - we do it ourselves.
// The authMiddleware ensures a session ID is always present in the request header.
// This allows brand selection to survive server restarts.
func (h *Handler) getServerForRequest(r *http.Request) *mcp.Server {
	// Session ID is always set by authMiddleware (generated if not provided by client)
	sessionID := r.Header.Get("Mcp-Session-Id")

	// Get token info from context (set by authMiddleware)
	tokenInfo := mcpauth.TokenInfoFromContext(r.Context())
	if tokenInfo == nil {
		fmt.Println("[MCP] ERROR: No token info in context for getServerForRequest")
		return NewServer(h.svc, r)
	}

	// Extract requesting user's ID from token
	requestingUserID, _ := tokenInfo.Extra["user_id"].(string)

	// Check cache first - if we have this session, reuse it
	if cached, ok := h.sessionCache.Load(sessionID); ok {
		data := cached.(*sessionData)
		// Validate that the requesting user owns this session
		if data.userID != "" && requestingUserID != "" && data.userID != requestingUserID {
			fmt.Printf("[MCP] Session %s belongs to user %s but requested by %s - invalidating\n", sessionID, data.userID, requestingUserID)
			h.sessionCache.Delete(sessionID)
		} else {
			fmt.Printf("[MCP] Using cached session: %s (hasBrand: %v)\n", sessionID, data.toolCtx != nil && data.toolCtx.HasBrand())
			return data.server
		}
	}

	// Session not in cache - this is either a new session or after server restart
	fmt.Printf("[MCP] Session %s not in cache - creating new server\n", sessionID)

	// Create callback that persists brand selection for this session
	onBrandSelect := func(userID, brandID string) {
		h.StoreBrandSelection(sessionID, userID, brandID)
	}
	server, toolCtx := NewServerWithContext(h.svc, r, onBrandSelect)

	// Restore this session's own brand choice (from memory, or from the DB
	// after a restart). A new session never inherits a brand from another
	// session: the caller chooses one with brand.select.
	var brandIDToRestore string
	var found bool

	if stored, ok := h.brandSelectionCache.Load(sessionID); ok && stored.(brandSelection).userID == requestingUserID {
		brandIDToRestore = stored.(brandSelection).brandID
		found = true
		fmt.Printf("[MCP] Found brand in memory cache for session %s\n", sessionID)
	} else {
		// Not in memory - try DB by session ID (this happens after server restart)
		dbSession, err := h.svc.DB.GetMCPSession(r.Context(), sessionID)
		if err == nil && dbSession.OrgID.Valid && dbSession.UserID == requestingUserID {
			brandIDToRestore = dbSession.OrgID.String
			found = true
			// Cache it for next time
			h.brandSelectionCache.Store(sessionID, brandSelection{userID: requestingUserID, brandID: brandIDToRestore})
			fmt.Printf("[MCP] Found brand in DB for session %s\n", sessionID)
		}
	}

	if found && toolCtx != nil {
		// Fetch the brand from DB and restore selection
		brand, err := h.svc.DB.GetOrganizationByID(r.Context(), brandIDToRestore)
		if err == nil {
			// Set the brand directly without callback (it's already persisted)
			toolCtx.RestoreBrand(brand)
			fmt.Printf("[MCP] Restored brand selection for session %s: brand=%s\n", sessionID, brand.Name)
		} else {
			fmt.Printf("[MCP] Failed to restore brand %s for session %s: %v\n", brandIDToRestore, sessionID, err)
		}
	}

	// Cache the session with user identity for security validation
	h.sessionCache.Store(sessionID, &sessionData{server: server, toolCtx: toolCtx, userID: requestingUserID})

	return server
}

// StoreBrandSelection persists brand selection for a session.
// Called by brand.select tool to ensure selection survives server restarts.
// Stores in both in-memory cache and DB for persistence.
func (h *Handler) StoreBrandSelection(sessionID string, userID, brandID string) {
	// Store in memory cache
	h.brandSelectionCache.Store(sessionID, brandSelection{userID: userID, brandID: brandID})

	// Persist to DB (async to not block tool execution)
	go func() {
		ctx := context.Background()
		err := h.svc.DB.UpsertMCPSession(ctx, db.UpsertMCPSessionParams{
			SessionID: sessionID,
			UserID:    userID,
			OrgID:     sql.NullString{String: brandID, Valid: brandID != ""},
		})
		if err != nil {
			fmt.Printf("[MCP] Failed to persist session to DB: %v\n", err)
		} else {
			fmt.Printf("[MCP] Persisted brand selection to DB: session=%s brand=%s\n", sessionID, brandID)
		}
	}()
}

// ClearBrandSelection removes brand selection for a session.
func (h *Handler) ClearBrandSelection(sessionID string) {
	h.brandSelectionCache.Delete(sessionID)

	// Remove from DB (async)
	go func() {
		ctx := context.Background()
		_ = h.svc.DB.DeleteMCPSession(ctx, sessionID)
	}()
}

// ServeHTTP handles all MCP HTTP requests.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.httpHandler.ServeHTTP(w, r)
}

// Authenticator returns the authenticator for cache invalidation.
func (h *Handler) Authenticator() *mcpauth.Authenticator {
	return h.authenticator
}
