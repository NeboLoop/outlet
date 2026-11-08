package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/outlet-sh/outlet/internal/db"

	"github.com/zeromicro/go-zero/core/logx"
)

type contextKey string

const (
	OrgIDKey  contextKey = "org_id"
	OrgKey    contextKey = "org"
)

type APIKeyMiddleware struct {
	store *db.Store
	cache sync.Map // map[apiKey]cachedOrg
}

// orgCacheTTL bounds how long a cached brand is trusted, so a deleted brand or
// a rotated key stops working within a minute even where InvalidateCache is
// not called.
const orgCacheTTL = time.Minute

type cachedOrg struct {
	org      db.Organization
	cachedAt time.Time
}

func NewAPIKeyMiddleware(store *db.Store) *APIKeyMiddleware {
	return &APIKeyMiddleware{store: store}
}

// getOrg retrieves an organization by API key, using cache when available.
func (m *APIKeyMiddleware) getOrg(ctx context.Context, apiKey string) (db.Organization, error) {
	if cached, ok := m.cache.Load(apiKey); ok && time.Since(cached.(cachedOrg).cachedAt) < orgCacheTTL {
		return cached.(cachedOrg).org, nil
	}

	// Lookup from database
	org, err := m.store.GetOrganizationByAPIKey(ctx, apiKey)
	if err != nil {
		return db.Organization{}, err
	}

	// Cache the result
	m.cache.Store(apiKey, cachedOrg{org: org, cachedAt: time.Now()})

	return org, nil
}

// InvalidateCache removes an API key from the cache (call when org is updated)
func (m *APIKeyMiddleware) InvalidateCache(apiKey string) {
	m.cache.Delete(apiKey)
}

func (m *APIKeyMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Missing API key",
			})
			return
		}

		org, err := m.getOrg(r.Context(), apiKey)
		if err != nil {
			logx.Errorf("API key lookup failed: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Invalid API key",
			})
			return
		}

		// Add org info to context
		ctx := context.WithValue(r.Context(), OrgIDKey, org.ID)
		ctx = context.WithValue(ctx, OrgKey, org)

		next(w, r.WithContext(ctx))
	}
}
