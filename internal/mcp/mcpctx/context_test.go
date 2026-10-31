package mcpctx

import (
	"context"
	"sync"
	"testing"

	"github.com/outlet-sh/outlet/internal/db"

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

// ========== NewToolContext Tests ==========

func TestNewToolContext_APIKeyMode(t *testing.T) {
	brandID := uuid.New().String()
	brand := createTestBrand(brandID, "Test Brand")

	tc := NewToolContext(nil, brand, "req-123", "test-agent/1.0")

	assert.NotNil(t, tc, "ToolContext should not be nil")
	assert.Equal(t, AuthModeAPIKey, tc.AuthMode(), "AuthMode should be APIKey")
	assert.Equal(t, brandID, tc.SessionBrand().ID, "BrandID should match")
	assert.Equal(t, "req-123", tc.RequestID(), "RequestID should match")
	assert.Equal(t, "test-agent/1.0", tc.UserAgent(), "UserAgent should match")
	assert.True(t, (tc.SessionBrand().ID != ""), "HasBrand should be true for API key mode")
}

func TestNewUserToolContext_OAuthMode(t *testing.T) {
	userID := uuid.New().String()
	user := createTestUser(userID, "test@example.com", "admin")

	tc := NewUserToolContext(nil, user, "req-456", "test-agent/1.0", "session-123")

	assert.NotNil(t, tc, "ToolContext should not be nil")
	assert.Equal(t, AuthModeOAuth, tc.AuthMode(), "AuthMode should be OAuth")
	assert.Equal(t, userID, tc.UserID(), "UserID should match")
	assert.Equal(t, "", tc.SessionBrand().ID, "BrandID should be empty before selection")
	assert.False(t, (tc.SessionBrand().ID != ""), "HasBrand should be false before brand selection")
	assert.Equal(t, "session-123", tc.SessionID(), "SessionID should match")
}

// ========== Brand Selection Tests ==========

func TestToolContext_SelectBrand(t *testing.T) {
	userID := uuid.New().String()
	user := createTestUser(userID, "test@example.com", "admin")
	tc := NewUserToolContext(nil, user, "req-123", "test-agent/1.0", "session-123")

	brandID := uuid.New().String()
	brand := createTestBrand(brandID, "Selected Brand")

	// Before selection
	assert.False(t, (tc.SessionBrand().ID != ""), "HasBrand should be false before selection")
	assert.Equal(t, "", tc.SessionBrand().ID, "BrandID should be empty before selection")

	// Select brand
	tc.SelectBrand(brand)

	// After selection
	assert.True(t, (tc.SessionBrand().ID != ""), "HasBrand should be true after selection")
	assert.Equal(t, brandID, tc.SessionBrand().ID, "BrandID should match selected brand")
	assert.Equal(t, brand.Name, tc.SessionBrand().Name, "Brand name should match")
}

func TestToolContext_SelectBrand_CallsCallback(t *testing.T) {
	userID := uuid.New().String()
	user := createTestUser(userID, "test@example.com", "admin")
	tc := NewUserToolContext(nil, user, "req-123", "test-agent/1.0", "session-123")

	brandID := uuid.New().String()
	brand := createTestBrand(brandID, "Selected Brand")

	var callbackUserID string
	var callbackBrandID string
	callbackCalled := false

	tc.SetBrandSelectionCallback(func(uid, bid string) {
		callbackCalled = true
		callbackUserID = uid
		callbackBrandID = bid
	})

	tc.SelectBrand(brand)

	assert.True(t, callbackCalled, "Callback should be called")
	assert.Equal(t, userID, callbackUserID, "Callback should receive user ID")
	assert.Equal(t, brandID, callbackBrandID, "Callback should receive brand ID")
}

func TestToolContext_RestoreBrand_DoesNotCallCallback(t *testing.T) {
	userID := uuid.New().String()
	user := createTestUser(userID, "test@example.com", "admin")
	tc := NewUserToolContext(nil, user, "req-123", "test-agent/1.0", "session-123")

	brandID := uuid.New().String()
	brand := createTestBrand(brandID, "Restored Brand")

	callbackCalled := false
	tc.SetBrandSelectionCallback(func(uid, bid string) {
		callbackCalled = true
	})

	tc.RestoreBrand(brand)

	assert.False(t, callbackCalled, "Callback should not be called for RestoreBrand")
	assert.True(t, (tc.SessionBrand().ID != ""), "HasBrand should be true after restore")
	assert.Equal(t, brandID, tc.SessionBrand().ID, "BrandID should match restored brand")
}

func TestToolContext_ClearBrand(t *testing.T) {
	userID := uuid.New().String()
	user := createTestUser(userID, "test@example.com", "admin")
	tc := NewUserToolContext(nil, user, "req-123", "test-agent/1.0", "session-123")

	brandID := uuid.New().String()
	brand := createTestBrand(brandID, "Brand to Clear")
	tc.SelectBrand(brand)

	assert.True(t, (tc.SessionBrand().ID != ""), "HasBrand should be true after selection")

	tc.ClearBrand()

	assert.False(t, (tc.SessionBrand().ID != ""), "HasBrand should be false after clear")
	assert.Equal(t, "", tc.SessionBrand().ID, "BrandID should be empty after clear")
}

// ========== APIKey Mode Tests ==========

func TestToolContext_APIKeyMode_AlwaysHasBrand(t *testing.T) {
	brandID := uuid.New().String()
	brand := createTestBrand(brandID, "API Key Brand")

	tc := NewToolContext(nil, brand, "req-123", "test-agent/1.0")

	// In API key mode, brand is always set
	assert.True(t, (tc.SessionBrand().ID != ""), "HasBrand should always be true in API key mode")
	assert.Equal(t, brandID, tc.SessionBrand().ID, "BrandID should be set")
	assert.Equal(t, brand.Name, tc.SessionBrand().Name, "Brand should match")
}

func TestToolContext_APIKeyMode_NoUserID(t *testing.T) {
	brandID := uuid.New().String()
	brand := createTestBrand(brandID, "API Key Brand")

	tc := NewToolContext(nil, brand, "req-123", "test-agent/1.0")

	assert.Equal(t, "", tc.UserID(), "UserID should be empty in API key mode")
	assert.Nil(t, tc.User(), "User should be nil in API key mode")
	assert.False(t, tc.IsSuperAdmin(), "IsSuperAdmin should be false when no user")
}

// ========== User and Permissions Tests ==========

func TestToolContext_IsSuperAdmin(t *testing.T) {
	tests := []struct {
		name           string
		role           string
		expectedResult bool
	}{
		{"Super admin role", "super_admin", true},
		{"Admin role", "admin", false},
		{"User role", "user", false},
		{"Empty role", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := uuid.New().String()
			user := createTestUser(userID, "test@example.com", tt.role)
			tc := NewUserToolContext(nil, user, "req-123", "test-agent/1.0", "session-123")

			assert.Equal(t, tt.expectedResult, tc.IsSuperAdmin(), "IsSuperAdmin should return %v for role %s", tt.expectedResult, tt.role)
		})
	}
}

func TestToolContext_User(t *testing.T) {
	userID := uuid.New().String()
	user := createTestUser(userID, "test@example.com", "admin")

	tc := NewUserToolContext(nil, user, "req-123", "test-agent/1.0", "session-123")

	assert.NotNil(t, tc.User(), "User should not be nil")
	assert.Equal(t, userID, tc.User().ID, "User ID should match")
	assert.Equal(t, "test@example.com", tc.User().Email, "User email should match")
}

// ========== RequireBrand Tests ==========

func TestToolContext_RequireBrand_Success(t *testing.T) {
	brandID := uuid.New().String()
	brand := createTestBrand(brandID, "Test Brand")

	tc := NewToolContext(nil, brand, "req-123", "test-agent/1.0")

	err := tc.RequireBrand(WithCallBrand(context.Background(), tc.SessionBrand()))
	assert.NoError(t, err, "RequireBrand should not return error when brand is set")
}

func TestToolContext_RequireBrand_Failure(t *testing.T) {
	userID := uuid.New().String()
	user := createTestUser(userID, "test@example.com", "admin")

	tc := NewUserToolContext(nil, user, "req-123", "test-agent/1.0", "session-123")

	err := tc.RequireBrand(WithCallBrand(context.Background(), tc.SessionBrand()))
	require.Error(t, err, "RequireBrand should return error when no brand selected")
	assert.Equal(t, ErrNoBrandSelected, err, "Error should be ErrNoBrandSelected")
}

// ========== ToolError Tests ==========

func TestToolError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ToolError
		expected string
	}{
		{
			name:     "Error without field",
			err:      &ToolError{Code: "not_found", Message: "Resource not found"},
			expected: "not_found: Resource not found",
		},
		{
			name:     "Error with field",
			err:      &ToolError{Code: "validation", Message: "Invalid value", Field: "email"},
			expected: "validation: Invalid value (field: email)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("Invalid email format", "email")

	assert.Equal(t, "validation", err.Code, "Code should be 'validation'")
	assert.Equal(t, "Invalid email format", err.Message, "Message should match")
	assert.Equal(t, "email", err.Field, "Field should be 'email'")
}

func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError("User not found")

	assert.Equal(t, "not_found", err.Code, "Code should be 'not_found'")
	assert.Equal(t, "User not found", err.Message, "Message should match")
	assert.Empty(t, err.Field, "Field should be empty")
}

func TestNewConflictError(t *testing.T) {
	err := NewConflictError("Email already exists")

	assert.Equal(t, "conflict", err.Code, "Code should be 'conflict'")
	assert.Equal(t, "Email already exists", err.Message, "Message should match")
	assert.Empty(t, err.Field, "Field should be empty")
}

// ========== Context Value Tests ==========

func TestWithToolContext_And_FromContext(t *testing.T) {
	brandID := uuid.New().String()
	brand := createTestBrand(brandID, "Test Brand")
	tc := NewToolContext(nil, brand, "req-123", "test-agent/1.0")

	ctx := context.Background()
	ctx = WithToolContext(ctx, tc)

	retrieved := ToolContextFromContext(ctx)
	require.NotNil(t, retrieved, "ToolContext should be retrievable from context")
	assert.Equal(t, tc, retrieved, "Retrieved context should match original")
	assert.Equal(t, brandID, retrieved.SessionBrand().ID, "BrandID should match")
}

func TestToolContextFromContext_NilWhenNotSet(t *testing.T) {
	ctx := context.Background()

	retrieved := ToolContextFromContext(ctx)
	assert.Nil(t, retrieved, "Should return nil when ToolContext not set")
}

func TestToolContextFromContext_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), toolContextKey{}, "wrong type")

	retrieved := ToolContextFromContext(ctx)
	assert.Nil(t, retrieved, "Should return nil for wrong type")
}

// ========== Concurrent Access Tests ==========

func TestToolContext_ConcurrentBrandSelection(t *testing.T) {
	userID := uuid.New().String()
	user := createTestUser(userID, "test@example.com", "admin")
	tc := NewUserToolContext(nil, user, "req-123", "test-agent/1.0", "session-123")

	var wg sync.WaitGroup
	numGoroutines := 100

	// Create different brands to select
	brands := make([]db.Organization, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		brands[i] = createTestBrand(uuid.New().String(), "Brand "+string(rune('A'+i%26)))
	}

	// Concurrent brand selection
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			tc.SelectBrand(brands[idx])
			// Read immediately after write
			_ = (tc.SessionBrand().ID != "")
			_ = tc.SessionBrand().ID
		}(i)
	}

	wg.Wait()

	// After all concurrent selections, should have a brand
	assert.True(t, (tc.SessionBrand().ID != ""), "Should have a brand after concurrent selections")
}

func TestToolContext_ConcurrentReads(t *testing.T) {
	brandID := uuid.New().String()
	brand := createTestBrand(brandID, "Test Brand")
	tc := NewToolContext(nil, brand, "req-123", "test-agent/1.0")

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = (tc.SessionBrand().ID != "")
			_ = tc.SessionBrand().ID
			_ = tc.SessionBrand()
			_ = tc.AuthMode()
			_ = tc.RequestID()
			_ = tc.UserAgent()
		}()
	}

	wg.Wait()
	// Test passes if no race condition panic occurs
}

// ========== Empty/Nil Brand Tests ==========

func TestToolContext_Brand_ReturnsEmptyWhenNil(t *testing.T) {
	userID := uuid.New().String()
	user := createTestUser(userID, "test@example.com", "admin")
	tc := NewUserToolContext(nil, user, "req-123", "test-agent/1.0", "session-123")

	brand := tc.SessionBrand()
	assert.Equal(t, db.Organization{}, brand, "Brand should return empty struct when not set")
}

// ========== Session ID Tests ==========

func TestToolContext_SessionID(t *testing.T) {
	userID := uuid.New().String()
	user := createTestUser(userID, "test@example.com", "admin")

	tc := NewUserToolContext(nil, user, "req-123", "test-agent/1.0", "session-abc-123")

	assert.Equal(t, "session-abc-123", tc.SessionID(), "SessionID should match")
}

func TestToolContext_SessionID_EmptyForAPIKeyMode(t *testing.T) {
	brandID := uuid.New().String()
	brand := createTestBrand(brandID, "Test Brand")

	tc := NewToolContext(nil, brand, "req-123", "test-agent/1.0")

	assert.Empty(t, tc.SessionID(), "SessionID should be empty for API key mode")
}
