package mcpauth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/modelcontextprotocol/go-sdk/auth"
)

// AuthMode indicates the type of authentication used.
type AuthMode string

const (
	AuthModeAPIKey AuthMode = "api_key"
	AuthModeOAuth  AuthMode = "oauth"
)

// UserInfo contains authenticated user information.
type UserInfo struct {
	UserID    string
	Email     string
	Name      string
	Role      string
	AuthMode  AuthMode
	Scopes    []string
	ExpiresAt time.Time
}

// userInfoKey is used to store UserInfo in context.
type userInfoKey struct{}

// tokenInfoKey is used to store TokenInfo in context.
type tokenInfoKey struct{}

// WithUserInfo adds UserInfo to a context.
func WithUserInfo(ctx context.Context, info *UserInfo) context.Context {
	return context.WithValue(ctx, userInfoKey{}, info)
}

// UserInfoFromContext retrieves UserInfo from a context.
func UserInfoFromContext(ctx context.Context) *UserInfo {
	if info, ok := ctx.Value(userInfoKey{}).(*UserInfo); ok {
		return info
	}
	return nil
}

// ContextWithTokenInfo adds TokenInfo to a context.
func ContextWithTokenInfo(ctx context.Context, info *auth.TokenInfo) context.Context {
	return context.WithValue(ctx, tokenInfoKey{}, info)
}

// TokenInfoFromContext retrieves TokenInfo from a context.
func TokenInfoFromContext(ctx context.Context) *auth.TokenInfo {
	if info, ok := ctx.Value(tokenInfoKey{}).(*auth.TokenInfo); ok {
		return info
	}
	return nil
}

// Authenticator handles MCP authentication for both API keys and OAuth tokens.
type Authenticator struct {
	svc *svc.ServiceContext

	// API key cache: hash -> cached result
	keyCache sync.Map // map[string]*cachedKey
}

// keyCacheTTL bounds how long a cached key is trusted, so a revoked key or a
// deactivated user stops working within a minute without a restart.
const keyCacheTTL = time.Minute

type cachedKey struct {
	cachedAt  time.Time
	userInfo  *UserInfo
	user      *db.User
	expiresAt time.Time
	revokedAt *time.Time
}

// NewAuthenticator creates a new MCP authenticator.
func NewAuthenticator(svc *svc.ServiceContext) *Authenticator {
	return &Authenticator{
		svc: svc,
	}
}

// TokenVerifier returns a token verifier function for use with auth.RequireBearerToken.
// It supports both API keys (lv_xxx format) and OAuth access tokens.
func (a *Authenticator) TokenVerifier() func(ctx context.Context, token string, req *http.Request) (*auth.TokenInfo, error) {
	return func(ctx context.Context, token string, req *http.Request) (*auth.TokenInfo, error) {
		// Determine token type by prefix
		if strings.HasPrefix(token, "lv_") {
			return a.verifyAPIKey(ctx, token)
		}
		// Assume OAuth token
		return a.verifyOAuthToken(ctx, token)
	}
}

// verifyAPIKey validates an API key and returns token info.
func (a *Authenticator) verifyAPIKey(ctx context.Context, apiKey string) (*auth.TokenInfo, error) {
	// Hash the key
	hash := hashToken(apiKey)

	// Check cache first
	if cached, ok := a.keyCache.Load(hash); ok && time.Since(cached.(*cachedKey).cachedAt) < keyCacheTTL {
		ck := cached.(*cachedKey)
		// Check if revoked
		if ck.revokedAt != nil {
			return nil, auth.ErrInvalidToken
		}
		// Check if expired
		if !ck.expiresAt.IsZero() && time.Now().After(ck.expiresAt) {
			return nil, auth.ErrInvalidToken
		}
		return &auth.TokenInfo{
			Scopes:     ck.userInfo.Scopes,
			Expiration: ck.expiresAt,
			Extra: map[string]any{
				"user_id":   ck.userInfo.UserID,
				"user_info": ck.userInfo,
				"user":      ck.user,
			},
		}, nil
	}

	// DB lookup
	keyRow, err := a.svc.DB.GetMCPAPIKeyByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, auth.ErrInvalidToken
		}
		return nil, err
	}

	// Build user info
	userInfo := &UserInfo{
		UserID:   keyRow.UserID,
		Email:    keyRow.UserEmail,
		Name:     keyRow.UserName,
		Role:     keyRow.UserRole,
		AuthMode: AuthModeAPIKey,
		Scopes:   parseScopes(keyRow.Scopes.String),
	}
	if keyRow.ExpiresAt.Valid {
		userInfo.ExpiresAt, _ = time.Parse(time.RFC3339, keyRow.ExpiresAt.String)
	}

	user := &db.User{
		ID:     keyRow.UserID,
		Email:  keyRow.UserEmail,
		Name:   keyRow.UserName,
		Role:   keyRow.UserRole,
		Status: keyRow.UserStatus,
	}

	// Cache the result
	ck := &cachedKey{
		cachedAt:  time.Now(),
		userInfo:  userInfo,
		user:      user,
		expiresAt: userInfo.ExpiresAt,
	}
	a.keyCache.Store(hash, ck)

	// Update last used (async)
	go func() {
		_ = a.svc.DB.UpdateMCPAPIKeyLastUsed(context.Background(), keyRow.ID)
	}()

	return &auth.TokenInfo{
		Scopes:     userInfo.Scopes,
		Expiration: userInfo.ExpiresAt,
		Extra: map[string]any{
			"user_id":   userInfo.UserID,
			"user_info": userInfo,
			"user":      user,
		},
	}, nil
}

// verifyOAuthToken validates an OAuth access token and returns token info.
func (a *Authenticator) verifyOAuthToken(ctx context.Context, accessToken string) (*auth.TokenInfo, error) {
	hash := hashToken(accessToken)

	tokenRow, err := a.svc.DB.GetMCPOAuthTokenByAccessHash(ctx, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, auth.ErrInvalidToken
		}
		return nil, err
	}

	expiresAt, _ := time.Parse(time.RFC3339, tokenRow.ExpiresAt)
	userInfo := &UserInfo{
		UserID:    tokenRow.UserID,
		Email:     tokenRow.UserEmail,
		Name:      tokenRow.UserName,
		Role:      tokenRow.UserRole,
		AuthMode:  AuthModeOAuth,
		Scopes:    parseScopes(tokenRow.Scopes),
		ExpiresAt: expiresAt,
	}

	user := &db.User{
		ID:     tokenRow.UserID,
		Email:  tokenRow.UserEmail,
		Name:   tokenRow.UserName,
		Role:   tokenRow.UserRole,
		Status: tokenRow.UserStatus,
	}

	return &auth.TokenInfo{
		Scopes:     userInfo.Scopes,
		Expiration: expiresAt,
		Extra: map[string]any{
			"user_id":   userInfo.UserID,
			"user_info": userInfo,
			"user":      user,
		},
	}, nil
}

func parseScopes(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

// InvalidateAPIKey removes an API key from the cache (call when revoked).
func (a *Authenticator) InvalidateAPIKey(apiKey string) {
	hash := hashToken(apiKey)
	a.keyCache.Delete(hash)
}

// InvalidateAPIKeyByHash removes an API key from cache by its hash.
func (a *Authenticator) InvalidateAPIKeyByHash(hash string) {
	a.keyCache.Delete(hash)
}

// hashToken creates a SHA-256 hash of a token.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// HashToken is exported for use when creating API keys.
func HashToken(token string) string {
	return hashToken(token)
}
