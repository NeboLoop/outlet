package utils

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// CustomClaims represents JWT claims with user information (admin users)
type CustomClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// SDKCustomerClaims represents JWT claims for SDK customer authentication
type SDKCustomerClaims struct {
	CustomerID string `json:"customer_id"`
	OrgID      string `json:"org_id"`
	Email      string `json:"email"`
	jwt.RegisteredClaims
}

// errNoSecret refuses tokens without a signing secret: anyone could sign one.
var errNoSecret = errors.New("JWT secret is not set (JWT_SECRET, JWT_REFRESH_SECRET)")

// GenerateToken creates a new JWT token for a user
func GenerateToken(userID, email, role, secret string, expireDuration time.Duration) (string, error) {
	if secret == "" {
		return "", errNoSecret
	}
	// Create claims with user data and expiration
	claims := CustomClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expireDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "outlet.sh",
			Subject:   userID,
		},
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates and parses a JWT token
func ValidateToken(tokenString, secret string) (*CustomClaims, error) {
	if secret == "" {
		return nil, errNoSecret
	}
	// Parse token
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Extract claims
	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

// ExtractClaims extracts user information from a token without full validation
// Useful for getting user info from a token that might be expired
func ExtractClaims(tokenString string) (*CustomClaims, error) {
	// Parse without validation
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, &CustomClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

// GenerateSDKToken creates a new JWT token for an SDK customer
func GenerateSDKToken(customerID, orgID, email, secret string, expireDuration time.Duration) (string, error) {
	claims := SDKCustomerClaims{
		CustomerID: customerID,
		OrgID:      orgID,
		Email:      email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expireDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "outlet.sh/sdk",
			Subject:   customerID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign SDK token: %w", err)
	}

	return tokenString, nil
}

// ValidateSDKToken validates and parses an SDK customer JWT token
func ValidateSDKToken(tokenString, secret string) (*SDKCustomerClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &SDKCustomerClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse SDK token: %w", err)
	}

	claims, ok := token.Claims.(*SDKCustomerClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid SDK token claims")
	}

	return claims, nil
}
