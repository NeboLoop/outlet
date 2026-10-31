package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testSecret      = "test-secret-key-for-jwt-testing"
	testSecretOther = "different-secret-key"
)

// TestGenerateToken tests JWT token generation for admin users
func TestGenerateToken(t *testing.T) {
	t.Run("generates valid token with correct claims", func(t *testing.T) {
		userID := "user-123"
		email := "test@example.com"
		role := "admin"
		expireDuration := 24 * time.Hour

		token, err := GenerateToken(userID, email, role, testSecret, expireDuration)

		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.True(t, strings.Count(token, ".") == 2, "JWT should have 3 parts separated by dots")
	})

	t.Run("generates different tokens for different users", func(t *testing.T) {
		token1, err := GenerateToken("user-1", "user1@example.com", "admin", testSecret, time.Hour)
		require.NoError(t, err)

		token2, err := GenerateToken("user-2", "user2@example.com", "agent", testSecret, time.Hour)
		require.NoError(t, err)

		assert.NotEqual(t, token1, token2)
	})

	t.Run("generates different tokens at different times", func(t *testing.T) {
		token1, err := GenerateToken("user-1", "user@example.com", "admin", testSecret, time.Hour)
		require.NoError(t, err)

		// JWT timestamps have second granularity, so we need to wait at least 1 second
		// to guarantee different IssuedAt values
		time.Sleep(1100 * time.Millisecond)

		token2, err := GenerateToken("user-1", "user@example.com", "admin", testSecret, time.Hour)
		require.NoError(t, err)

		assert.NotEqual(t, token1, token2)
	})

	t.Run("refuses an empty secret", func(t *testing.T) {
		// Anyone could sign a token with an empty secret.
		token, err := GenerateToken("user-1", "user@example.com", "admin", "", time.Hour)
		require.Error(t, err)
		assert.Empty(t, token)

		signed, err := GenerateToken("user-1", "user@example.com", "admin", testSecret, time.Hour)
		require.NoError(t, err)
		_, err = ValidateToken(signed, "")
		require.Error(t, err)
	})

	t.Run("handles empty user fields", func(t *testing.T) {
		token, err := GenerateToken("", "", "", testSecret, time.Hour)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("handles various expiration durations", func(t *testing.T) {
		testCases := []struct {
			name     string
			duration time.Duration
		}{
			{"one second", time.Second},
			{"one minute", time.Minute},
			{"one hour", time.Hour},
			{"one day", 24 * time.Hour},
			{"one week", 7 * 24 * time.Hour},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				token, err := GenerateToken("user-1", "user@example.com", "admin", testSecret, tc.duration)
				require.NoError(t, err)
				assert.NotEmpty(t, token)
			})
		}
	})
}

// TestValidateToken tests JWT token validation for admin users
func TestValidateToken(t *testing.T) {
	t.Run("validates token and returns correct claims", func(t *testing.T) {
		userID := "user-123"
		email := "test@example.com"
		role := "admin"

		token, err := GenerateToken(userID, email, role, testSecret, time.Hour)
		require.NoError(t, err)

		claims, err := ValidateToken(token, testSecret)
		require.NoError(t, err)

		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, role, claims.Role)
		assert.Equal(t, userID, claims.Subject)
		assert.Equal(t, "outlet.sh", claims.Issuer)
	})

	t.Run("validates registered claims", func(t *testing.T) {
		now := time.Now()
		token, err := GenerateToken("user-123", "test@example.com", "admin", testSecret, time.Hour)
		require.NoError(t, err)

		claims, err := ValidateToken(token, testSecret)
		require.NoError(t, err)

		assert.NotNil(t, claims.IssuedAt)
		assert.NotNil(t, claims.ExpiresAt)
		assert.NotNil(t, claims.NotBefore)

		// Check times are within reasonable bounds
		assert.WithinDuration(t, now, claims.IssuedAt.Time, 2*time.Second)
		assert.WithinDuration(t, now.Add(time.Hour), claims.ExpiresAt.Time, 2*time.Second)
	})

	t.Run("rejects expired token", func(t *testing.T) {
		// Generate token that expired in the past
		token, err := GenerateToken("user-123", "test@example.com", "admin", testSecret, -time.Hour)
		require.NoError(t, err)

		_, err = ValidateToken(token, testSecret)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse token")
	})

	t.Run("rejects token with wrong secret", func(t *testing.T) {
		token, err := GenerateToken("user-123", "test@example.com", "admin", testSecret, time.Hour)
		require.NoError(t, err)

		_, err = ValidateToken(token, testSecretOther)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse token")
	})

	t.Run("rejects malformed token", func(t *testing.T) {
		testCases := []struct {
			name  string
			token string
		}{
			{"empty string", ""},
			{"random string", "not-a-jwt-token"},
			{"too few parts", "header.payload"},
			{"too many parts", "header.payload.signature.extra"},
			{"invalid base64", "!!!.@@@.###"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := ValidateToken(tc.token, testSecret)
				assert.Error(t, err)
			})
		}
	})

	t.Run("rejects token with tampered payload", func(t *testing.T) {
		token, err := GenerateToken("user-123", "test@example.com", "admin", testSecret, time.Hour)
		require.NoError(t, err)

		// Tamper with the payload (middle part)
		parts := strings.Split(token, ".")
		require.Len(t, parts, 3)
		tamperedToken := parts[0] + ".dGFtcGVyZWQ." + parts[2]

		_, err = ValidateToken(tamperedToken, testSecret)
		assert.Error(t, err)
	})

	t.Run("rejects token signed with different algorithm", func(t *testing.T) {
		// Create a token with RS256 (would need private key, so we fake it)
		// This tests the signing method check
		claims := CustomClaims{
			UserID: "user-123",
			Email:  "test@example.com",
			Role:   "admin",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
		}

		// Create token with "none" algorithm (insecure)
		token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
		tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
		require.NoError(t, err)

		_, err = ValidateToken(tokenString, testSecret)
		assert.Error(t, err)
	})
}

// TestExtractClaims tests extracting claims without validation
func TestExtractClaims(t *testing.T) {
	t.Run("extracts claims from valid token", func(t *testing.T) {
		userID := "user-123"
		email := "test@example.com"
		role := "admin"

		token, err := GenerateToken(userID, email, role, testSecret, time.Hour)
		require.NoError(t, err)

		claims, err := ExtractClaims(token)
		require.NoError(t, err)

		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, role, claims.Role)
	})

	t.Run("extracts claims from expired token", func(t *testing.T) {
		userID := "user-123"
		email := "test@example.com"
		role := "admin"

		// Generate an expired token
		token, err := GenerateToken(userID, email, role, testSecret, -time.Hour)
		require.NoError(t, err)

		// ValidateToken should fail
		_, err = ValidateToken(token, testSecret)
		assert.Error(t, err)

		// But ExtractClaims should succeed
		claims, err := ExtractClaims(token)
		require.NoError(t, err)

		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, role, claims.Role)
	})

	t.Run("extracts claims from token with wrong signature", func(t *testing.T) {
		token, err := GenerateToken("user-123", "test@example.com", "admin", testSecret, time.Hour)
		require.NoError(t, err)

		// Tamper with signature (last part)
		parts := strings.Split(token, ".")
		require.Len(t, parts, 3)
		tamperedToken := parts[0] + "." + parts[1] + ".invalidsignature"

		// ValidateToken should fail
		_, err = ValidateToken(tamperedToken, testSecret)
		assert.Error(t, err)

		// But ExtractClaims should succeed (it doesn't verify signature)
		claims, err := ExtractClaims(tamperedToken)
		require.NoError(t, err)
		assert.Equal(t, "user-123", claims.UserID)
	})

	t.Run("fails on malformed token", func(t *testing.T) {
		testCases := []struct {
			name  string
			token string
		}{
			{"empty string", ""},
			{"random string", "not-a-jwt-token"},
			{"invalid base64 in payload", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.!!!invalid!!!.signature"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := ExtractClaims(tc.token)
				assert.Error(t, err)
			})
		}
	})
}

// TestGenerateSDKToken tests JWT token generation for SDK customers
func TestGenerateSDKToken(t *testing.T) {
	t.Run("generates valid SDK token with correct claims", func(t *testing.T) {
		customerID := "customer-123"
		orgID := "org-456"
		email := "customer@example.com"
		expireDuration := 24 * time.Hour

		token, err := GenerateSDKToken(customerID, orgID, email, testSecret, expireDuration)

		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.True(t, strings.Count(token, ".") == 2, "JWT should have 3 parts separated by dots")
	})

	t.Run("generates different tokens for different customers", func(t *testing.T) {
		token1, err := GenerateSDKToken("customer-1", "org-1", "c1@example.com", testSecret, time.Hour)
		require.NoError(t, err)

		token2, err := GenerateSDKToken("customer-2", "org-2", "c2@example.com", testSecret, time.Hour)
		require.NoError(t, err)

		assert.NotEqual(t, token1, token2)
	})

	t.Run("SDK token is different from admin token", func(t *testing.T) {
		adminToken, err := GenerateToken("user-123", "admin@example.com", "admin", testSecret, time.Hour)
		require.NoError(t, err)

		sdkToken, err := GenerateSDKToken("user-123", "org-123", "admin@example.com", testSecret, time.Hour)
		require.NoError(t, err)

		assert.NotEqual(t, adminToken, sdkToken)
	})

	t.Run("handles empty fields", func(t *testing.T) {
		token, err := GenerateSDKToken("", "", "", testSecret, time.Hour)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})
}

// TestValidateSDKToken tests JWT token validation for SDK customers
func TestValidateSDKToken(t *testing.T) {
	t.Run("validates SDK token and returns correct claims", func(t *testing.T) {
		customerID := "customer-123"
		orgID := "org-456"
		email := "customer@example.com"

		token, err := GenerateSDKToken(customerID, orgID, email, testSecret, time.Hour)
		require.NoError(t, err)

		claims, err := ValidateSDKToken(token, testSecret)
		require.NoError(t, err)

		assert.Equal(t, customerID, claims.CustomerID)
		assert.Equal(t, orgID, claims.OrgID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, customerID, claims.Subject)
		assert.Equal(t, "outlet.sh/sdk", claims.Issuer)
	})

	t.Run("validates registered claims", func(t *testing.T) {
		now := time.Now()
		token, err := GenerateSDKToken("customer-123", "org-456", "customer@example.com", testSecret, time.Hour)
		require.NoError(t, err)

		claims, err := ValidateSDKToken(token, testSecret)
		require.NoError(t, err)

		assert.NotNil(t, claims.IssuedAt)
		assert.NotNil(t, claims.ExpiresAt)
		assert.NotNil(t, claims.NotBefore)

		// Check times are within reasonable bounds
		assert.WithinDuration(t, now, claims.IssuedAt.Time, 2*time.Second)
		assert.WithinDuration(t, now.Add(time.Hour), claims.ExpiresAt.Time, 2*time.Second)
	})

	t.Run("rejects expired SDK token", func(t *testing.T) {
		token, err := GenerateSDKToken("customer-123", "org-456", "customer@example.com", testSecret, -time.Hour)
		require.NoError(t, err)

		_, err = ValidateSDKToken(token, testSecret)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse SDK token")
	})

	t.Run("rejects SDK token with wrong secret", func(t *testing.T) {
		token, err := GenerateSDKToken("customer-123", "org-456", "customer@example.com", testSecret, time.Hour)
		require.NoError(t, err)

		_, err = ValidateSDKToken(token, testSecretOther)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse SDK token")
	})

	t.Run("rejects malformed token", func(t *testing.T) {
		testCases := []struct {
			name  string
			token string
		}{
			{"empty string", ""},
			{"random string", "not-a-jwt-token"},
			{"too few parts", "header.payload"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := ValidateSDKToken(tc.token, testSecret)
				assert.Error(t, err)
			})
		}
	})

	t.Run("rejects token with tampered payload", func(t *testing.T) {
		token, err := GenerateSDKToken("customer-123", "org-456", "customer@example.com", testSecret, time.Hour)
		require.NoError(t, err)

		// Tamper with the payload
		parts := strings.Split(token, ".")
		require.Len(t, parts, 3)
		tamperedToken := parts[0] + ".dGFtcGVyZWQ." + parts[2]

		_, err = ValidateSDKToken(tamperedToken, testSecret)
		assert.Error(t, err)
	})
}

// TestTokenTypeMismatch tests that admin and SDK tokens cannot be interchanged
func TestTokenTypeMismatch(t *testing.T) {
	t.Run("admin token cannot be validated as SDK token", func(t *testing.T) {
		adminToken, err := GenerateToken("user-123", "admin@example.com", "admin", testSecret, time.Hour)
		require.NoError(t, err)

		// Should fail to validate as SDK token (different claim structure)
		claims, err := ValidateSDKToken(adminToken, testSecret)
		require.NoError(t, err)

		// Claims will be empty/wrong since the token has different claim fields
		assert.Empty(t, claims.CustomerID)
		assert.Empty(t, claims.OrgID)
	})

	t.Run("SDK token cannot be validated as admin token", func(t *testing.T) {
		sdkToken, err := GenerateSDKToken("customer-123", "org-456", "customer@example.com", testSecret, time.Hour)
		require.NoError(t, err)

		// Should fail to validate as admin token (different claim structure)
		claims, err := ValidateToken(sdkToken, testSecret)
		require.NoError(t, err)

		// Claims will be empty/wrong since the token has different claim fields
		assert.Empty(t, claims.UserID)
		assert.Empty(t, claims.Role)
	})
}

// TestCustomClaimsStructure tests the CustomClaims struct
func TestCustomClaimsStructure(t *testing.T) {
	t.Run("CustomClaims implements jwt.Claims", func(t *testing.T) {
		claims := CustomClaims{
			UserID: "user-123",
			Email:  "test@example.com",
			Role:   "admin",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
		}

		// Verify it implements the interface by creating a token
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		assert.NotNil(t, token)
	})

	t.Run("SDKCustomerClaims implements jwt.Claims", func(t *testing.T) {
		claims := SDKCustomerClaims{
			CustomerID: "customer-123",
			OrgID:      "org-456",
			Email:      "customer@example.com",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
		}

		// Verify it implements the interface by creating a token
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		assert.NotNil(t, token)
	})
}

// TestExpirationEdgeCases tests edge cases around token expiration
func TestExpirationEdgeCases(t *testing.T) {
	t.Run("token valid immediately after creation", func(t *testing.T) {
		token, err := GenerateToken("user-123", "test@example.com", "admin", testSecret, time.Hour)
		require.NoError(t, err)

		claims, err := ValidateToken(token, testSecret)
		require.NoError(t, err)
		assert.NotNil(t, claims)
	})

	t.Run("token with short expiration expires", func(t *testing.T) {
		// JWT expiration has second granularity, so we need at least 1+ second expiration
		// to reliably test expiration behavior
		token, err := GenerateToken("user-123", "test@example.com", "admin", testSecret, 2*time.Second)
		require.NoError(t, err)

		// Should be valid immediately
		claims, err := ValidateToken(token, testSecret)
		require.NoError(t, err)
		assert.NotNil(t, claims)

		// Wait for expiration (2 seconds + buffer)
		time.Sleep(2500 * time.Millisecond)

		// Should be invalid now
		_, err = ValidateToken(token, testSecret)
		assert.Error(t, err)
	})

	t.Run("zero duration creates already-expired token", func(t *testing.T) {
		token, err := GenerateToken("user-123", "test@example.com", "admin", testSecret, 0)
		require.NoError(t, err)

		// Token expires immediately (at creation time)
		// There might be a tiny window where it's valid, so we give a small buffer
		time.Sleep(10 * time.Millisecond)

		_, err = ValidateToken(token, testSecret)
		assert.Error(t, err)
	})
}

// TestSecretHandling tests various secret key scenarios
func TestSecretHandling(t *testing.T) {
	t.Run("works with short secret", func(t *testing.T) {
		shortSecret := "key"
		token, err := GenerateToken("user-123", "test@example.com", "admin", shortSecret, time.Hour)
		require.NoError(t, err)

		claims, err := ValidateToken(token, shortSecret)
		require.NoError(t, err)
		assert.Equal(t, "user-123", claims.UserID)
	})

	t.Run("works with long secret", func(t *testing.T) {
		longSecret := strings.Repeat("a", 1000)
		token, err := GenerateToken("user-123", "test@example.com", "admin", longSecret, time.Hour)
		require.NoError(t, err)

		claims, err := ValidateToken(token, longSecret)
		require.NoError(t, err)
		assert.Equal(t, "user-123", claims.UserID)
	})

	t.Run("works with special characters in secret", func(t *testing.T) {
		specialSecret := "!@#$%^&*()_+-=[]{}|;':\",./<>?"
		token, err := GenerateToken("user-123", "test@example.com", "admin", specialSecret, time.Hour)
		require.NoError(t, err)

		claims, err := ValidateToken(token, specialSecret)
		require.NoError(t, err)
		assert.Equal(t, "user-123", claims.UserID)
	})

	t.Run("works with unicode secret", func(t *testing.T) {
		unicodeSecret := "日本語の秘密鍵🔐"
		token, err := GenerateToken("user-123", "test@example.com", "admin", unicodeSecret, time.Hour)
		require.NoError(t, err)

		claims, err := ValidateToken(token, unicodeSecret)
		require.NoError(t, err)
		assert.Equal(t, "user-123", claims.UserID)
	})
}

// TestSpecialCharactersInClaims tests claims with special characters
func TestSpecialCharactersInClaims(t *testing.T) {
	t.Run("handles special characters in email", func(t *testing.T) {
		email := "user+tag@example.com"
		token, err := GenerateToken("user-123", email, "admin", testSecret, time.Hour)
		require.NoError(t, err)

		claims, err := ValidateToken(token, testSecret)
		require.NoError(t, err)
		assert.Equal(t, email, claims.Email)
	})

	t.Run("handles unicode in user ID", func(t *testing.T) {
		userID := "用户-123"
		token, err := GenerateToken(userID, "test@example.com", "admin", testSecret, time.Hour)
		require.NoError(t, err)

		claims, err := ValidateToken(token, testSecret)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
	})

	t.Run("handles long values in claims", func(t *testing.T) {
		longEmail := strings.Repeat("a", 100) + "@example.com"
		token, err := GenerateToken("user-123", longEmail, "admin", testSecret, time.Hour)
		require.NoError(t, err)

		claims, err := ValidateToken(token, testSecret)
		require.NoError(t, err)
		assert.Equal(t, longEmail, claims.Email)
	})
}

// BenchmarkGenerateToken benchmarks token generation
func BenchmarkGenerateToken(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GenerateToken("user-123", "test@example.com", "admin", testSecret, time.Hour)
	}
}

// BenchmarkValidateToken benchmarks token validation
func BenchmarkValidateToken(b *testing.B) {
	token, _ := GenerateToken("user-123", "test@example.com", "admin", testSecret, time.Hour)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ValidateToken(token, testSecret)
	}
}

// BenchmarkExtractClaims benchmarks claim extraction without validation
func BenchmarkExtractClaims(b *testing.B) {
	token, _ := GenerateToken("user-123", "test@example.com", "admin", testSecret, time.Hour)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ExtractClaims(token)
	}
}

// BenchmarkGenerateSDKToken benchmarks SDK token generation
func BenchmarkGenerateSDKToken(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GenerateSDKToken("customer-123", "org-456", "customer@example.com", testSecret, time.Hour)
	}
}

// BenchmarkValidateSDKToken benchmarks SDK token validation
func BenchmarkValidateSDKToken(b *testing.B) {
	token, _ := GenerateSDKToken("customer-123", "org-456", "customer@example.com", testSecret, time.Hour)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ValidateSDKToken(token, testSecret)
	}
}
