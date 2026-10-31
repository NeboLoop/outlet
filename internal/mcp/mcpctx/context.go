package mcpctx

import (
	"context"
	"sync"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
)

// AuthMode indicates how the MCP session was authenticated.
type AuthMode int

const (
	// AuthModeAPIKey means authenticated via X-API-Key header (brand-scoped).
	AuthModeAPIKey AuthMode = iota
	// AuthModeOAuth means authenticated via Bearer token (user-scoped, needs brand selection).
	AuthModeOAuth
)

// BrandSelectionCallback is called when brand selection changes.
// The handler uses this to persist brand selection for session recovery.
// userID is included for DB persistence.
type BrandSelectionCallback func(userID, brandID string)

// ToolContext carries context for all MCP tools.
// It supports both API key auth (brand-scoped) and OAuth auth (user-scoped with brand selection).
type ToolContext struct {
	svc       *svc.ServiceContext
	requestID string
	userAgent string
	sessionID string

	// Auth mode
	authMode AuthMode

	// API key auth: brand is set directly
	brand *db.Organization

	// OAuth auth: user is set, brand must be selected
	user        *db.User
	selectedBrand *db.Organization
	mu          sync.RWMutex // protects selectedBrand

	// Callback for persisting brand selection (set by Handler)
	onBrandSelect BrandSelectionCallback
}

// NewToolContext creates a new brand-scoped tool context (API key auth).
func NewToolContext(svc *svc.ServiceContext, brand db.Organization, requestID, userAgent string) *ToolContext {
	return &ToolContext{
		svc:       svc,
		brand:     &brand,
		requestID: requestID,
		userAgent: userAgent,
		authMode:  AuthModeAPIKey,
	}
}

// NewUserToolContext creates a new user-scoped tool context (OAuth auth).
// The user must select a brand using brand.select before using other tools.
func NewUserToolContext(svc *svc.ServiceContext, user db.User, requestID, userAgent, sessionID string) *ToolContext {
	return &ToolContext{
		svc:       svc,
		user:      &user,
		requestID: requestID,
		userAgent: userAgent,
		sessionID: sessionID,
		authMode:  AuthModeOAuth,
	}
}

// SetBrandSelectionCallback sets the callback for persisting brand selection.
func (t *ToolContext) SetBrandSelectionCallback(cb BrandSelectionCallback) {
	t.onBrandSelect = cb
}

// SessionID returns the MCP session ID.
func (t *ToolContext) SessionID() string {
	return t.sessionID
}

// SessionBrand returns the brand this session works in: the API key's
// brand, or the one brand.select chose. It is the zero value when none is
// chosen. Only the gate reads it: it checks the caller's access to that
// brand and hands the brand to the call (WithCallBrand), and tool handlers
// use only that one.
func (t *ToolContext) SessionBrand() db.Organization {
	if t.authMode == AuthModeAPIKey {
		if t.brand == nil {
			return db.Organization{}
		}
		return *t.brand
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.selectedBrand == nil {
		return db.Organization{}
	}
	return *t.selectedBrand
}

type callBrandKey struct{}

// WithCallBrand records the brand the gate checked for one tool call.
func WithCallBrand(ctx context.Context, brand db.Organization) context.Context {
	return context.WithValue(ctx, callBrandKey{}, brand)
}

// Brand returns the brand this tool call acts in: the one the gate checked
// the caller's access to. A brand.select running at the same time on the
// same session doesn't change it. It is the zero value when the call has no
// brand.
func (t *ToolContext) Brand(ctx context.Context) db.Organization {
	brand, _ := ctx.Value(callBrandKey{}).(db.Organization)
	return brand
}

// BrandID returns the ID of the brand this tool call acts in, or "".
func (t *ToolContext) BrandID(ctx context.Context) string {
	return t.Brand(ctx).ID
}

// AuthMode returns the authentication mode.
func (t *ToolContext) AuthMode() AuthMode {
	return t.authMode
}

// User returns the authenticated user (OAuth mode only).
func (t *ToolContext) User() *db.User {
	return t.user
}

// UserID returns the authenticated user's ID (OAuth mode only).
// Returns empty string if not in OAuth mode.
func (t *ToolContext) UserID() string {
	if t.user == nil {
		return ""
	}
	return t.user.ID
}

// IsSuperAdmin returns true if the user has the super_admin role.
func (t *ToolContext) IsSuperAdmin() bool {
	if t.user == nil {
		return false
	}
	return t.user.Role == "super_admin"
}

// SelectBrand sets the current brand for OAuth sessions.
// Also calls the persistence callback if set.
func (t *ToolContext) SelectBrand(brand db.Organization) {
	t.mu.Lock()
	t.selectedBrand = &brand
	cb := t.onBrandSelect
	userID := t.UserID()
	t.mu.Unlock()

	// Call callback outside the lock to persist the selection
	if cb != nil {
		cb(userID, brand.ID)
	}
}

// RestoreBrand sets the current brand without triggering the callback.
// Used when restoring brand selection from persistent storage.
func (t *ToolContext) RestoreBrand(brand db.Organization) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.selectedBrand = &brand
}

// ClearBrand clears the selected brand (OAuth mode).
func (t *ToolContext) ClearBrand() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.selectedBrand = nil
}

// DB returns the database store for queries.
func (t *ToolContext) DB() *db.Store {
	return t.svc.DB
}

// Svc returns the full service context for advanced operations.
func (t *ToolContext) Svc() *svc.ServiceContext {
	return t.svc
}

// RequestID returns the request ID for tracing.
func (t *ToolContext) RequestID() string {
	return t.requestID
}

// UserAgent returns the client's user agent string.
func (t *ToolContext) UserAgent() string {
	return t.userAgent
}

// ToolError represents a structured error for MCP tool responses.
type ToolError struct {
	Code    string `json:"code"`    // "not_found", "validation", "conflict", "rate_limit"
	Message string `json:"message"` // Human-readable description
	Field   string `json:"field"`   // For validation errors
}

func (e *ToolError) Error() string {
	if e.Field != "" {
		return e.Code + ": " + e.Message + " (field: " + e.Field + ")"
	}
	return e.Code + ": " + e.Message
}

// NewValidationError creates a validation error for a specific field.
func NewValidationError(message, field string) *ToolError {
	return &ToolError{Code: "validation", Message: message, Field: field}
}

// NewNotFoundError creates a not found error.
func NewNotFoundError(message string) *ToolError {
	return &ToolError{Code: "not_found", Message: message}
}

// NewConflictError creates a conflict error (duplicate, already exists).
func NewConflictError(message string) *ToolError {
	return &ToolError{Code: "conflict", Message: message}
}

// ErrNoBrandSelected is returned when a brand-scoped operation is attempted without a brand selected.
var ErrNoBrandSelected = &ToolError{
	Code:    "no_brand_selected",
	Message: "No brand selected. Use brand.list to see available brands and brand.select to choose one.",
}

// RequireBrand returns an error if this tool call has no brand.
// Use this at the start of tools that need brand context.
func (t *ToolContext) RequireBrand(ctx context.Context) error {
	if t.BrandID(ctx) == "" {
		return ErrNoBrandSelected
	}
	return nil
}

// toolContextKey is used to store ToolContext in context.Context
type toolContextKey struct{}

// WithToolContext adds ToolContext to a context.
func WithToolContext(ctx context.Context, tc *ToolContext) context.Context {
	return context.WithValue(ctx, toolContextKey{}, tc)
}

// ToolContextFromContext retrieves ToolContext from a context.
func ToolContextFromContext(ctx context.Context) *ToolContext {
	if tc, ok := ctx.Value(toolContextKey{}).(*ToolContext); ok {
		return tc
	}
	return nil
}
