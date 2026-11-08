package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/outlet-sh/outlet/internal/utils"
	"github.com/zeromicro/go-zero/core/logx"
)

// AuthMiddleware admits admin requests that carry an access token signed with
// the configured access secret.
type AuthMiddleware struct {
	accessSecret string
}

func NewAuthMiddleware(accessSecret string) *AuthMiddleware {
	return &AuthMiddleware{accessSecret: accessSecret}
}

func (m *AuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// Try to get from cookie
			cookie, err := r.Cookie("auth_token")
			if err == nil && cookie.Value != "" {
				authHeader = "Bearer " + cookie.Value
			}
		}

		if authHeader == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "No authentication token provided",
			})
			return
		}

		// Extract token from "Bearer <token>" format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Invalid authorization header format",
			})
			return
		}

		tokenString := parts[1]

		// Verify the signature, signing method and expiry before trusting any claim
		claims, err := utils.ValidateToken(tokenString, m.accessSecret)
		if err == nil && claims.UserID != "" {
			// Add user info to context for use in handlers
			ctx := context.WithValue(r.Context(), "userId", claims.UserID)
			if claims.Email != "" {
				ctx = context.WithValue(ctx, "email", claims.Email)
			}
			if claims.Role != "" {
				ctx = context.WithValue(ctx, "role", claims.Role)
			}
			ctx = context.WithValue(ctx, "token", tokenString)

			// Read X-Org-Id header for org-scoped admin operations
			orgIDStr := r.Header.Get("X-Org-Id")
			if orgIDStr != "" {
				// Validate it's a proper UUID, but store as string (logic files expect string)
				if _, err := uuid.Parse(orgIDStr); err == nil {
					ctx = context.WithValue(ctx, OrgIDKey, orgIDStr)
				}
			}

			next(w, r.WithContext(ctx))
			return
		}
		if err != nil {
			logx.Errorf("Rejected admin token: %v", err)
		}

		// If we get here, token was invalid
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid or expired token",
		})
	}
}
