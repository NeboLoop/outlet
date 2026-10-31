package apikeys

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"time"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/mcp/mcpauth"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateMCPAPIKeyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateMCPAPIKeyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateMCPAPIKeyLogic {
	return &CreateMCPAPIKeyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateMCPAPIKeyLogic) CreateMCPAPIKey(req *types.CreateMCPAPIKeyRequest) (resp *types.CreateMCPAPIKeyResponse, err error) {
	// Get user ID from context
	user, ok := access.User(l.ctx)
	if !ok {
		return nil, errorx.NewUnauthorizedError("Unauthorized")
	}
	userID := user.ID

	// Validate name
	if req.Name == "" {
		return nil, errorx.NewBadRequestError("Name is required")
	}

	// Generate a secure API key: lv_<32 random bytes base64>
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		l.Errorf("Failed to generate random bytes: %v", err)
		return nil, errorx.NewInternalError("Failed to generate API key")
	}

	// Use URL-safe base64 encoding
	keyBody := base64.RawURLEncoding.EncodeToString(randomBytes)
	fullKey := "lv_" + keyBody
	keyPrefix := fullKey[:11] // "lv_" + first 8 chars of body

	// Hash the key for storage
	keyHash := mcpauth.HashToken(fullKey)

	// Parse expiry if provided
	var expiresAt sql.NullString
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			return nil, errorx.NewBadRequestError("Invalid expires_at format (use ISO 8601)")
		}
		expiresAt = sql.NullString{String: t.Format("2006-01-02 15:04:05"), Valid: true}
	}

	// Create the key in database
	key, err := l.svcCtx.DB.CreateMCPAPIKey(l.ctx, db.CreateMCPAPIKeyParams{
		ID:        uuid.New().String(),
		UserID:    userID,
		Name:      req.Name,
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		Scopes:    sql.NullString{String: "mcp:full", Valid: true},
		ExpiresAt: expiresAt,
	})
	if err != nil {
		l.Errorf("Failed to create API key: %v", err)
		return nil, errorx.NewInternalError("Failed to create API key")
	}

	// Format created_at for response
	createdAt := key.CreatedAt.String
	if createdAt == "" {
		createdAt = time.Now().Format("2006-01-02T15:04:05Z")
	}

	return &types.CreateMCPAPIKeyResponse{
		Id:        key.ID,
		Name:      key.Name,
		Key:       fullKey, // Only returned on creation!
		KeyPrefix: key.KeyPrefix,
		CreatedAt: createdAt,
	}, nil
}
