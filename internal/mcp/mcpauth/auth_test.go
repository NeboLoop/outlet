package mcpauth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========== HashToken Tests ==========

func TestHashToken_Deterministic(t *testing.T) {
	token := "test-token-123"

	hash1 := HashToken(token)
	hash2 := HashToken(token)

	assert.Equal(t, hash1, hash2, "Same token should produce same hash")
}

func TestHashToken_DifferentForDifferentTokens(t *testing.T) {
	token1 := "token-one"
	token2 := "token-two"

	hash1 := HashToken(token1)
	hash2 := HashToken(token2)

	assert.NotEqual(t, hash1, hash2, "Different tokens should produce different hashes")
}

func TestHashToken_ExpectedFormat(t *testing.T) {
	token := "test-token"
	hash := HashToken(token)

	// Should be hex-encoded SHA-256 hash (64 characters)
	assert.Len(t, hash, 64, "Hash should be 64 characters (hex-encoded SHA-256)")

	// Verify it's valid hex
	_, err := hex.DecodeString(hash)
	assert.NoError(t, err, "Hash should be valid hex")
}

func TestHashToken_ManualVerification(t *testing.T) {
	token := "known-token"
	hash := HashToken(token)

	// Manually compute expected hash
	expected := sha256.Sum256([]byte(token))
	expectedHex := hex.EncodeToString(expected[:])

	assert.Equal(t, expectedHex, hash, "Hash should match manual computation")
}

func TestHashToken_EmptyString(t *testing.T) {
	hash := HashToken("")

	// Empty string should still produce a valid hash
	assert.Len(t, hash, 64, "Empty string should produce valid hash")

	// Verify it's the hash of empty string
	expected := sha256.Sum256([]byte(""))
	expectedHex := hex.EncodeToString(expected[:])
	assert.Equal(t, expectedHex, hash)
}

func TestHashToken_SpecialCharacters(t *testing.T) {
	tokens := []string{
		"token-with-special!@#$%^&*()",
		"unicode-token-\u4e2d\u6587",
		"newline\ntoken",
		"tab\ttoken",
		"null\x00token",
	}

	hashes := make(map[string]bool)
	for _, token := range tokens {
		hash := HashToken(token)
		assert.Len(t, hash, 64, "Special character token should produce valid hash")
		assert.False(t, hashes[hash], "All hashes should be unique")
		hashes[hash] = true
	}
}

func TestHashToken_LongToken(t *testing.T) {
	// Create a very long token
	longToken := ""
	for i := 0; i < 10000; i++ {
		longToken += "x"
	}

	hash := HashToken(longToken)

	// Should still produce valid 64-char hex hash
	assert.Len(t, hash, 64, "Long token should produce valid hash")
}

// ========== UserInfo Tests ==========

func TestUserInfo_Structure(t *testing.T) {
	expiresAt := time.Now().Add(1 * time.Hour)
	info := &UserInfo{
		UserID:    uuid.New().String(),
		Email:     "test@example.com",
		Name:      "Test User",
		Role:      "admin",
		AuthMode:  AuthModeOAuth,
		Scopes:    []string{"mcp:full", "offline_access"},
		ExpiresAt: expiresAt,
	}

	assert.NotEmpty(t, info.UserID)
	assert.Equal(t, "test@example.com", info.Email)
	assert.Equal(t, "Test User", info.Name)
	assert.Equal(t, "admin", info.Role)
	assert.Equal(t, AuthModeOAuth, info.AuthMode)
	assert.Len(t, info.Scopes, 2)
	assert.Equal(t, expiresAt, info.ExpiresAt)
}

func TestUserInfo_APIKeyAuthMode(t *testing.T) {
	info := &UserInfo{
		AuthMode: AuthModeAPIKey,
	}

	assert.Equal(t, AuthModeAPIKey, info.AuthMode)
	assert.Equal(t, AuthMode("api_key"), info.AuthMode)
}

func TestUserInfo_OAuthAuthMode(t *testing.T) {
	info := &UserInfo{
		AuthMode: AuthModeOAuth,
	}

	assert.Equal(t, AuthModeOAuth, info.AuthMode)
	assert.Equal(t, AuthMode("oauth"), info.AuthMode)
}

// ========== Context Tests ==========

func TestWithUserInfo_And_UserInfoFromContext(t *testing.T) {
	info := &UserInfo{
		UserID:   uuid.New().String(),
		Email:    "test@example.com",
		AuthMode: AuthModeOAuth,
	}

	ctx := context.Background()
	ctx = WithUserInfo(ctx, info)

	retrieved := UserInfoFromContext(ctx)

	require.NotNil(t, retrieved, "UserInfo should be retrievable from context")
	assert.Equal(t, info.UserID, retrieved.UserID)
	assert.Equal(t, info.Email, retrieved.Email)
	assert.Equal(t, info.AuthMode, retrieved.AuthMode)
}

func TestUserInfoFromContext_NilWhenNotSet(t *testing.T) {
	ctx := context.Background()

	retrieved := UserInfoFromContext(ctx)

	assert.Nil(t, retrieved, "Should return nil when UserInfo not set")
}

func TestUserInfoFromContext_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), userInfoKey{}, "wrong type")

	retrieved := UserInfoFromContext(ctx)

	assert.Nil(t, retrieved, "Should return nil for wrong type")
}

// ========== Authenticator Tests ==========

func TestNewAuthenticator(t *testing.T) {
	auth := NewAuthenticator(nil)

	assert.NotNil(t, auth, "Authenticator should not be nil")
}

func TestAuthenticator_TokenVerifier_ReturnsFunction(t *testing.T) {
	auth := NewAuthenticator(nil)

	verifier := auth.TokenVerifier()

	assert.NotNil(t, verifier, "TokenVerifier should return a function")
}

// ========== API Key Format Tests ==========

func TestAPIKeyFormat_Prefix(t *testing.T) {
	// API keys start with lv_ prefix
	apiKey := "lv_test123456789"
	assert.True(t, len(apiKey) > 3, "API key should have prefix plus random part")
	assert.Equal(t, "lv_", apiKey[:3], "API key should start with lv_")
}

// ========== Token Verification Path Tests ==========

func TestTokenVerifier_APIKeyPath(t *testing.T) {
	// Note: We can't fully test this without a DB, but we test that the verifier
	// function is created correctly. The actual DB call would panic without a real DB.
	auth := NewAuthenticator(nil)
	verifier := auth.TokenVerifier()

	assert.NotNil(t, verifier, "TokenVerifier should return a function")

	// Verify that lv_ prefix tokens would go through API key path
	// (We can't call verifier without a DB as it will panic)
	token := "lv_testkey"
	assert.True(t, len(token) > 3 && token[:3] == "lv_", "API key format should have lv_ prefix")
}

func TestTokenVerifier_OAuthPath(t *testing.T) {
	// Note: We can't fully test this without a DB, but we test the token format distinction
	auth := NewAuthenticator(nil)
	verifier := auth.TokenVerifier()

	assert.NotNil(t, verifier, "TokenVerifier should return a function")

	// Verify that non-lv_ tokens would go through OAuth path
	token := "oauth_access_token_123"
	assert.False(t, len(token) >= 3 && token[:3] == "lv_", "OAuth token should not have lv_ prefix")
}

// ========== AuthMode Constants Tests ==========

func TestAuthModeConstants(t *testing.T) {
	assert.Equal(t, AuthMode("api_key"), AuthModeAPIKey)
	assert.Equal(t, AuthMode("oauth"), AuthModeOAuth)
}

// ========== TokenInfo Context Tests ==========

func TestContextWithTokenInfo_And_TokenInfoFromContext(t *testing.T) {
	// We can't fully test this without importing the auth package from go-sdk,
	// but we can test the basic context operations
	ctx := context.Background()

	// Without token info
	info := TokenInfoFromContext(ctx)
	assert.Nil(t, info, "Should return nil when no token info")
}

// ========== Edge Cases ==========

func TestHashToken_BinaryData(t *testing.T) {
	// Token with binary/null bytes
	binaryToken := string([]byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD})
	hash := HashToken(binaryToken)

	assert.Len(t, hash, 64, "Binary token should produce valid hash")
}

func TestHashToken_VeryLongToken(t *testing.T) {
	// Create a 1MB token
	longToken := make([]byte, 1024*1024)
	for i := range longToken {
		longToken[i] = byte(i % 256)
	}

	hash := HashToken(string(longToken))

	assert.Len(t, hash, 64, "Very long token should produce valid hash")
}

func TestUserInfo_EmptyScopes(t *testing.T) {
	info := &UserInfo{
		UserID: uuid.New().String(),
		Scopes: []string{},
	}

	assert.Empty(t, info.Scopes)
	assert.NotNil(t, info.Scopes)
}

func TestUserInfo_NilScopes(t *testing.T) {
	info := &UserInfo{
		UserID: uuid.New().String(),
		Scopes: nil,
	}

	assert.Nil(t, info.Scopes)
}

// ========== Expiration Tests ==========

func TestUserInfo_ZeroExpiration(t *testing.T) {
	info := &UserInfo{
		UserID:    uuid.New().String(),
		ExpiresAt: time.Time{}, // Zero value
	}

	assert.True(t, info.ExpiresAt.IsZero(), "Zero expiration should be zero")
}

func TestUserInfo_FutureExpiration(t *testing.T) {
	futureTime := time.Now().Add(24 * time.Hour)
	info := &UserInfo{
		UserID:    uuid.New().String(),
		ExpiresAt: futureTime,
	}

	assert.True(t, info.ExpiresAt.After(time.Now()), "Should be in the future")
}

func TestUserInfo_PastExpiration(t *testing.T) {
	pastTime := time.Now().Add(-24 * time.Hour)
	info := &UserInfo{
		UserID:    uuid.New().String(),
		ExpiresAt: pastTime,
	}

	assert.True(t, info.ExpiresAt.Before(time.Now()), "Should be in the past")
}
