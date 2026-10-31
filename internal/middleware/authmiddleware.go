package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/google/uuid"
)

// AuthMiddleware signs people in to the admin REST API. It checks the
// token's signature and expiry, then that the account is still active, on
// every request. What the person may do in a brand is decided by
// access.Check in each handler's logic.
type AuthMiddleware struct {
	secret string
	store  db.Querier
}

func NewAuthMiddleware(secret string, store db.Querier) *AuthMiddleware {
	return &AuthMiddleware{secret: secret, store: store}
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
			writeAuthError(w, http.StatusUnauthorized, "No authentication token provided")
			return
		}

		// Extract token from "Bearer <token>" format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			writeAuthError(w, http.StatusUnauthorized, "Invalid authorization header format")
			return
		}

		claims, err := utils.ValidateToken(parts[1], m.secret)
		if err != nil || claims.UserID == "" {
			writeAuthError(w, http.StatusUnauthorized, "Your session is not valid. Sign in again.")
			return
		}

		user, err := access.Active(r.Context(), m.store, claims.UserID)
		if err != nil {
			status := http.StatusInternalServerError
			msg := "Could not check your account. Try again."
			var ce *errorx.CodeError
			if errors.As(err, &ce) {
				status, msg = ce.Code, ce.Message
			}
			writeAuthError(w, status, msg)
			return
		}

		ctx := access.WithUser(r.Context(), user)
		// X-Org-Id names the brand for brand-scoped admin calls. Naming it
		// grants nothing: access.RequireBrand checks it in the logic.
		if orgID := r.Header.Get("X-Org-Id"); orgID != "" {
			if _, err := uuid.Parse(orgID); err == nil {
				ctx = access.WithBrand(ctx, orgID)
			}
		}

		next(w, r.WithContext(ctx))
	}
}

func writeAuthError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
