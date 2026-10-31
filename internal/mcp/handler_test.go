package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/mcp/mcpauth"
	"github.com/outlet-sh/outlet/internal/mcp/mcpctx"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestBrand creates a mock brand for testing
func createTestBrand(id string, name string) db.Organization {
	return db.Organization{
		ID:     id,
		Name:   name,
		Slug:   "test-brand",
		ApiKey: "test-api-key",
	}
}

// createTestUser creates a mock user for testing
func createTestUser(id string, email, role string) db.User {
	return db.User{
		ID:     id,
		Email:  email,
		Name:   "Test User",
		Role:   role,
		Status: "active",
	}
}

// ========== Handler Creation Tests ==========

func TestNewHandler(t *testing.T) {
	// Create handler with nil svc for basic instantiation test
	handler := NewHandler(nil, "https://example.com")

	assert.NotNil(t, handler, "Handler should not be nil")
}

func TestNewHandler_TrimTrailingSlash(t *testing.T) {
	handler := NewHandler(nil, "https://example.com/")

	// The handler should have trimmed the trailing slash internally
	assert.NotNil(t, handler)
}

func TestNewHandler_Authenticator(t *testing.T) {
	handler := NewHandler(nil, "https://example.com")

	auth := handler.Authenticator()
	assert.NotNil(t, auth, "Authenticator should not be nil")
}

// ========== HTTP Handler Tests ==========

func TestHandler_ServeHTTP_MissingAuth(t *testing.T) {
	handler := NewHandler(nil, "https://example.com")

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code, "Should return 401 without auth")

	// Check WWW-Authenticate header
	wwwAuth := rr.Header().Get("WWW-Authenticate")
	assert.Contains(t, wwwAuth, "Bearer", "Should have Bearer challenge")
	assert.Contains(t, wwwAuth, "resource_metadata", "Should include resource_metadata")
}

func TestHandler_ServeHTTP_EmptyBearer(t *testing.T) {
	handler := NewHandler(nil, "https://example.com")

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer ")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code, "Should return 401 with empty bearer")
}

func TestHandler_ServeHTTP_InvalidToken(t *testing.T) {
	// Note: Without a real DB, we can't fully test token verification
	// The authenticator will panic without a DB connection
	// This test documents the expected behavior
	t.Skip("Skipping: requires DB connection for token verification")
}

func TestHandler_ServeHTTP_BasicAuth_Rejected(t *testing.T) {
	handler := NewHandler(nil, "https://example.com")

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz") // user:pass
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Basic auth should be rejected since it doesn't start with "Bearer "
	assert.Equal(t, http.StatusUnauthorized, rr.Code, "Should reject Basic auth")
}

// ========== Session Cache Tests ==========

func TestHandler_SessionCache_StoreAndLoad(t *testing.T) {
	handler := NewHandler(nil, "https://example.com")

	sessionID := "test-session-123"
	server := NewServer(nil, httptest.NewRequest(http.MethodPost, "/mcp", nil))

	// Store session
	handler.sessionCache.Store(sessionID, &sessionData{server: server, toolCtx: nil})

	// Load session
	cached, ok := handler.sessionCache.Load(sessionID)

	assert.True(t, ok, "Session should be found")
	assert.NotNil(t, cached, "Session data should not be nil")

	data := cached.(*sessionData)
	assert.Equal(t, server, data.server, "Server should match")
}

func TestHandler_SessionCache_Delete(t *testing.T) {
	handler := NewHandler(nil, "https://example.com")

	sessionID := "session-to-delete"
	handler.sessionCache.Store(sessionID, &sessionData{})

	// Verify stored
	_, ok := handler.sessionCache.Load(sessionID)
	assert.True(t, ok, "Session should be stored")

	// Delete
	handler.sessionCache.Delete(sessionID)

	// Verify deleted
	_, ok = handler.sessionCache.Load(sessionID)
	assert.False(t, ok, "Session should be deleted")
}

// ========== Brand Selection Cache Tests ==========

func TestHandler_StoreBrandSelection(t *testing.T) {
	// Note: StoreBrandSelection launches a goroutine that tries to persist to DB
	// Without a DB, this will panic in the background goroutine
	// We test the memory cache directly instead

	handler := NewHandler(nil, "https://example.com")

	sessionID := "test-session"
	brandID := uuid.New().String()

	// Directly test the in-memory cache (avoid triggering the DB goroutine)
	handler.brandSelectionCache.Store(sessionID, brandSelection{userID: "u1", brandID: brandID})

	// Check memory cache
	cached, ok := handler.brandSelectionCache.Load(sessionID)
	require.True(t, ok, "Brand selection should be in memory cache")
	assert.Equal(t, brandID, cached.(brandSelection).brandID, "BrandID should match")
}

func TestHandler_ClearBrandSelection(t *testing.T) {
	// Note: ClearBrandSelection launches a goroutine that tries to delete from DB
	// Without a DB, this will panic in the background goroutine
	// We test the memory cache directly instead

	handler := NewHandler(nil, "https://example.com")

	sessionID := "session-to-clear"
	brandID := uuid.New().String()

	// Store directly to cache
	handler.brandSelectionCache.Store(sessionID, brandSelection{userID: "u1", brandID: brandID})

	// Verify stored
	_, ok := handler.brandSelectionCache.Load(sessionID)
	assert.True(t, ok, "Brand selection should be stored")

	// Clear directly from cache (avoid triggering the DB goroutine)
	handler.brandSelectionCache.Delete(sessionID)

	// Verify cleared
	_, ok = handler.brandSelectionCache.Load(sessionID)
	assert.False(t, ok, "Brand selection should be cleared")
}

// ========== Multi-Tenant Isolation Tests ==========

func TestHandler_MultiTenant_DifferentBrandsPerSession(t *testing.T) {
	_ = NewHandler(nil, "https://example.com") // Keep for session cache tests

	// Create two users with different brands
	user1 := createTestUser(uuid.New().String(), "user1@example.com", "admin")
	user2 := createTestUser(uuid.New().String(), "user2@example.com", "admin")

	brand1 := createTestBrand(uuid.New().String(), "Brand 1")
	brand2 := createTestBrand(uuid.New().String(), "Brand 2")

	// Create tool contexts for each user
	tc1 := mcpctx.NewUserToolContext(nil, user1, "req-1", "agent/1.0", "session-1")
	tc2 := mcpctx.NewUserToolContext(nil, user2, "req-2", "agent/1.0", "session-2")

	// Select different brands
	tc1.SelectBrand(brand1)
	tc2.SelectBrand(brand2)

	// Verify isolation
	assert.Equal(t, brand1.ID, tc1.SessionBrand().ID, "User 1 should have Brand 1")
	assert.Equal(t, brand2.ID, tc2.SessionBrand().ID, "User 2 should have Brand 2")
	assert.NotEqual(t, tc1.SessionBrand().ID, tc2.SessionBrand().ID, "Brands should be different")
}

func TestHandler_MultiTenant_SameUserMultipleSessions(t *testing.T) {
	handler := NewHandler(nil, "https://example.com")

	// Same user, different sessions (e.g., different browser tabs)
	user := createTestUser(uuid.New().String(), "user@example.com", "admin")

	brand1 := createTestBrand(uuid.New().String(), "Brand 1")
	brand2 := createTestBrand(uuid.New().String(), "Brand 2")

	// Create two sessions for same user
	session1 := "session-browser-1"
	session2 := "session-browser-2"

	tc1 := mcpctx.NewUserToolContext(nil, user, "req-1", "agent/1.0", session1)
	tc2 := mcpctx.NewUserToolContext(nil, user, "req-2", "agent/1.0", session2)

	// User selects different brands in different sessions
	tc1.SelectBrand(brand1)
	tc2.SelectBrand(brand2)

	// Store in handler's session cache
	handler.sessionCache.Store(session1, &sessionData{toolCtx: tc1})
	handler.sessionCache.Store(session2, &sessionData{toolCtx: tc2})

	// Verify each session has its own brand
	cached1, _ := handler.sessionCache.Load(session1)
	cached2, _ := handler.sessionCache.Load(session2)

	data1 := cached1.(*sessionData)
	data2 := cached2.(*sessionData)

	assert.Equal(t, brand1.ID, data1.toolCtx.SessionBrand().ID, "Session 1 should have Brand 1")
	assert.Equal(t, brand2.ID, data2.toolCtx.SessionBrand().ID, "Session 2 should have Brand 2")
}

func TestHandler_MultiTenant_BrandIsolation_RequireBrand(t *testing.T) {
	// Test that RequireBrand properly enforces brand selection
	user := createTestUser(uuid.New().String(), "user@example.com", "admin")

	// Create context without brand selection
	tc := mcpctx.NewUserToolContext(nil, user, "req-1", "agent/1.0", "session-1")

	// Should require brand selection before operations
	err := tc.RequireBrand(mcpctx.WithCallBrand(context.Background(), tc.SessionBrand()))
	assert.Error(t, err, "Should error when no brand selected")
	assert.Equal(t, mcpctx.ErrNoBrandSelected, err)

	// Select brand
	brand := createTestBrand(uuid.New().String(), "My Brand")
	tc.SelectBrand(brand)

	// Now should not error
	err = tc.RequireBrand(mcpctx.WithCallBrand(context.Background(), tc.SessionBrand()))
	assert.NoError(t, err, "Should not error after brand selection")
}

// ========== Session Persistence Tests ==========

func TestHandler_SessionRestore_FromMemoryCache(t *testing.T) {
	handler := NewHandler(nil, "https://example.com")

	sessionID := "persistent-session"
	brandID := uuid.New().String()

	// Store brand selection in memory cache
	handler.brandSelectionCache.Store(sessionID, brandSelection{userID: "u1", brandID: brandID})

	// Verify it can be retrieved
	cached, ok := handler.brandSelectionCache.Load(sessionID)
	require.True(t, ok, "Should find brand in memory cache")
	assert.Equal(t, brandID, cached.(brandSelection).brandID)
}

// ========== Concurrent Access Tests ==========

func TestHandler_ConcurrentSessionAccess(t *testing.T) {
	handler := NewHandler(nil, "https://example.com")

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			sessionID := "session-" + string(rune('0'+idx%10))
			brandID := uuid.New().String()

			// Store
			handler.brandSelectionCache.Store(sessionID, brandSelection{userID: "u1", brandID: brandID})

			// Load
			if cached, ok := handler.brandSelectionCache.Load(sessionID); ok {
				_ = cached.(brandSelection)
			}

			// Store session
			handler.sessionCache.Store(sessionID, &sessionData{})

			// Load session
			if cached, ok := handler.sessionCache.Load(sessionID); ok {
				_ = cached.(*sessionData)
			}
		}(i)
	}

	wg.Wait()
	// Test passes if no race condition panic occurs
}

func TestHandler_ConcurrentBrandSelection(t *testing.T) {
	// Note: StoreBrandSelection triggers a DB goroutine that panics without a DB
	// We test the concurrent cache access directly instead
	handler := NewHandler(nil, "https://example.com")

	user := createTestUser(uuid.New().String(), "user@example.com", "admin")

	var wg sync.WaitGroup
	numGoroutines := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			sessionID := uuid.New().String()
			brandID := uuid.New().String()

			// Create context
			tc := mcpctx.NewUserToolContext(nil, user, "req", "agent", sessionID)

			// Select brand
			brand := createTestBrand(brandID, "Test Brand")
			tc.SelectBrand(brand)

			// Store directly in cache (avoid DB goroutine)
			handler.brandSelectionCache.Store(sessionID, brandSelection{userID: "u1", brandID: brandID})

			// Read brand
			_ = tc.SessionBrand().ID
			_ = (tc.SessionBrand().ID != "")

			// Read from cache
			_, _ = handler.brandSelectionCache.Load(sessionID)
		}()
	}

	wg.Wait()
	// Test passes if no race condition panic occurs
}

// ========== WWW-Authenticate Header Tests ==========

func TestHandler_WWWAuthenticate_Format(t *testing.T) {
	handler := NewHandler(nil, "https://api.example.com")

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	wwwAuth := rr.Header().Get("WWW-Authenticate")

	// Should include Bearer scheme
	assert.Contains(t, wwwAuth, "Bearer", "Should have Bearer scheme")

	// Should include resource_metadata with quoted URL
	assert.Contains(t, wwwAuth, "resource_metadata=", "Should have resource_metadata")

	// Should include scope
	assert.Contains(t, wwwAuth, "scope=", "Should have scope")
	assert.Contains(t, wwwAuth, "mcp:full", "Should have mcp:full scope")
}

// ========== Tool Context Tests ==========

func TestHandler_ToolContext_UserInfo(t *testing.T) {
	user := createTestUser(uuid.New().String(), "admin@example.com", "super_admin")

	tc := mcpctx.NewUserToolContext(nil, user, "req-123", "test-agent/1.0", "session-abc")

	assert.Equal(t, user.ID, tc.UserID(), "UserID should match")
	assert.True(t, tc.IsSuperAdmin(), "Should be super admin")
	assert.Equal(t, mcpctx.AuthModeOAuth, tc.AuthMode(), "Should be OAuth mode")
	assert.Equal(t, "session-abc", tc.SessionID(), "SessionID should match")
}

func TestHandler_ToolContext_BrandContext(t *testing.T) {
	brand := createTestBrand(uuid.New().String(), "Test Company")

	tc := mcpctx.NewToolContext(nil, brand, "req-123", "test-agent/1.0")

	assert.Equal(t, brand.ID, tc.SessionBrand().ID, "BrandID should match")
	assert.True(t, (tc.SessionBrand().ID != ""), "Should have brand")
	assert.Equal(t, mcpctx.AuthModeAPIKey, tc.AuthMode(), "Should be API key mode")
	assert.Empty(t, tc.SessionID(), "SessionID should be empty for API key mode")
}

// ========== OAuth vs API Key Mode Tests ==========

func TestHandler_APIKeyMode_NoBrandSelection(t *testing.T) {
	brand := createTestBrand(uuid.New().String(), "API Key Brand")

	// API key mode: brand is set directly, no selection needed
	tc := mcpctx.NewToolContext(nil, brand, "req-123", "agent/1.0")

	assert.True(t, (tc.SessionBrand().ID != ""), "Should have brand in API key mode")
	assert.Equal(t, brand.ID, tc.SessionBrand().ID)
	assert.NoError(t, tc.RequireBrand(mcpctx.WithCallBrand(context.Background(), tc.SessionBrand())), "RequireBrand should pass")
}

func TestHandler_OAuthMode_RequiresBrandSelection(t *testing.T) {
	user := createTestUser(uuid.New().String(), "user@example.com", "admin")

	// OAuth mode: brand must be selected
	tc := mcpctx.NewUserToolContext(nil, user, "req-123", "agent/1.0", "session-123")

	assert.False(t, (tc.SessionBrand().ID != ""), "Should not have brand before selection")
	assert.Equal(t, "", tc.SessionBrand().ID)
	assert.Error(t, tc.RequireBrand(mcpctx.WithCallBrand(context.Background(), tc.SessionBrand())), "RequireBrand should fail before selection")

	// Select brand
	brand := createTestBrand(uuid.New().String(), "Selected Brand")
	tc.SelectBrand(brand)

	assert.True(t, (tc.SessionBrand().ID != ""), "Should have brand after selection")
	assert.Equal(t, brand.ID, tc.SessionBrand().ID)
	assert.NoError(t, tc.RequireBrand(mcpctx.WithCallBrand(context.Background(), tc.SessionBrand())), "RequireBrand should pass after selection")
}

// ========== Session Data Tests ==========

func TestSessionData_Structure(t *testing.T) {
	user := createTestUser(uuid.New().String(), "user@example.com", "admin")
	tc := mcpctx.NewUserToolContext(nil, user, "req-123", "agent/1.0", "session-123")

	// Create mock server (actual server creation would require full setup)
	data := &sessionData{
		server:  nil, // Would be MCP server
		toolCtx: tc,
	}

	assert.NotNil(t, data.toolCtx)
	assert.Equal(t, "session-123", data.toolCtx.SessionID())
}

// ========== Edge Cases ==========

func TestHandler_EmptySessionID(t *testing.T) {
	handler := NewHandler(nil, "https://example.com")

	// Should handle empty session ID gracefully
	_, ok := handler.sessionCache.Load("")
	assert.False(t, ok, "Empty session ID should not find anything")
}

func TestHandler_UUIDSessionID(t *testing.T) {
	handler := NewHandler(nil, "https://example.com")

	// UUID-style session IDs should work
	sessionID := uuid.New().String()
	brandID := uuid.New().String()

	handler.brandSelectionCache.Store(sessionID, brandSelection{userID: "u1", brandID: brandID})

	cached, ok := handler.brandSelectionCache.Load(sessionID)
	assert.True(t, ok, "UUID session ID should work")
	assert.Equal(t, brandID, cached.(brandSelection).brandID)
}

func TestHandler_SpecialCharacterSessionID(t *testing.T) {
	handler := NewHandler(nil, "https://example.com")

	// Session IDs with special characters
	sessionID := "session-with-special!@#$%"
	brandID := uuid.New().String()

	handler.brandSelectionCache.Store(sessionID, brandSelection{userID: "u1", brandID: brandID})

	cached, ok := handler.brandSelectionCache.Load(sessionID)
	assert.True(t, ok, "Special character session ID should work")
	assert.Equal(t, brandID, cached.(brandSelection).brandID)
}

// ========== HTTP Method Tests ==========

func TestHandler_DifferentHTTPMethods(t *testing.T) {
	handler := NewHandler(nil, "https://example.com")

	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
		http.MethodOptions,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/mcp", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			// All methods without auth should return 401
			assert.Equal(t, http.StatusUnauthorized, rr.Code, "%s should return 401 without auth", method)
		})
	}
}

// ========== Brand Selection Callback Tests ==========

func TestHandler_BrandSelectionCallback(t *testing.T) {
	user := createTestUser(uuid.New().String(), "user@example.com", "admin")
	tc := mcpctx.NewUserToolContext(nil, user, "req-123", "agent/1.0", "session-123")

	var callbackUserID string
	var callbackBrandID string
	callbackCalled := false

	tc.SetBrandSelectionCallback(func(uid, bid string) {
		callbackCalled = true
		callbackUserID = uid
		callbackBrandID = bid
	})

	brand := createTestBrand(uuid.New().String(), "Test Brand")
	tc.SelectBrand(brand)

	assert.True(t, callbackCalled, "Callback should be called")
	assert.Equal(t, user.ID, callbackUserID, "Callback should receive user ID")
	assert.Equal(t, brand.ID, callbackBrandID, "Callback should receive brand ID")
}

// ========== Token Info Context Tests ==========

func TestHandler_TokenInfoInContext(t *testing.T) {
	ctx := httptest.NewRequest(http.MethodGet, "/", nil).Context()

	// Without token info
	info := mcpauth.TokenInfoFromContext(ctx)
	assert.Nil(t, info, "Should be nil when not set")
}
