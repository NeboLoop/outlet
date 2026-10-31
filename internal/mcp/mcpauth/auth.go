package mcpauth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/access"
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
// Every request reads the key or token from the database, so a revoked key
// stops working at once, and the person behind it must still be active.
type Authenticator struct {
	svc *svc.ServiceContext
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
	// The query skips revoked and expired keys.
	keyRow, err := a.svc.DB.GetMCPAPIKeyByHash(ctx, hashToken(apiKey))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, auth.ErrInvalidToken
		}
		return nil, err
	}

	user, err := access.Active(ctx, a.svc.DB, keyRow.UserID)
	if err != nil {
		return nil, auth.ErrInvalidToken
	}

	var expiresAt time.Time
	if keyRow.ExpiresAt.Valid {
		expiresAt, _ = time.Parse(time.DateTime, keyRow.ExpiresAt.String)
	}

	// Update last used (async)
	go func() {
		_ = a.svc.DB.UpdateMCPAPIKeyLastUsed(context.Background(), keyRow.ID)
	}()

	return tokenInfo(user, AuthModeAPIKey, ParseScopes(keyRow.Scopes.String), expiresAt), nil
}

// verifyOAuthToken validates an OAuth access token and returns token info.
func (a *Authenticator) verifyOAuthToken(ctx context.Context, accessToken string) (*auth.TokenInfo, error) {
	tokenRow, err := a.svc.DB.GetMCPOAuthTokenByAccessHash(ctx, hashToken(accessToken))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, auth.ErrInvalidToken
		}
		return nil, err
	}

	user, err := access.Active(ctx, a.svc.DB, tokenRow.UserID)
	if err != nil {
		return nil, auth.ErrInvalidToken
	}

	expiresAt, _ := time.Parse(time.RFC3339, tokenRow.ExpiresAt)
	return tokenInfo(user, AuthModeOAuth, ParseScopes(tokenRow.Scopes), expiresAt), nil
}

func tokenInfo(user db.User, mode AuthMode, scopes []string, expiresAt time.Time) *auth.TokenInfo {
	userInfo := &UserInfo{
		UserID:    user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      user.Role,
		AuthMode:  mode,
		Scopes:    scopes,
		ExpiresAt: expiresAt,
	}
	return &auth.TokenInfo{
		Scopes:     scopes,
		Expiration: expiresAt,
		Extra: map[string]any{
			"user_id":   user.ID,
			"user_info": userInfo,
			"user":      &user,
		},
	}
}

// ScopeFull lets a key or token call every tool, with the person's own access.
const ScopeFull = "mcp:full"

// ParseScopes reads a stored scope list: OAuth space-separated, comma-separated,
// or a JSON array (the column default).
func ParseScopes(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == ',' || r == '[' || r == ']' || r == '"'
	})
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
