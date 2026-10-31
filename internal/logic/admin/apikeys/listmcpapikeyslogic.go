package apikeys

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListMCPAPIKeysLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListMCPAPIKeysLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListMCPAPIKeysLogic {
	return &ListMCPAPIKeysLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListMCPAPIKeysLogic) ListMCPAPIKeys() (resp *types.ListMCPAPIKeysResponse, err error) {
	// Get user ID from context
	user, ok := access.User(l.ctx)
	if !ok {
		return nil, errorx.NewUnauthorizedError("Unauthorized")
	}
	userID := user.ID

	// List API keys for this user
	keys, err := l.svcCtx.DB.ListMCPAPIKeysByUser(l.ctx, userID)
	if err != nil {
		l.Errorf("Failed to list API keys: %v", err)
		return nil, errorx.NewInternalError("Failed to list API keys")
	}

	// Convert to response
	keyInfos := make([]types.MCPAPIKeyInfo, 0, len(keys))
	for _, key := range keys {
		// Skip revoked keys
		if key.RevokedAt.Valid {
			continue
		}

		keyInfo := types.MCPAPIKeyInfo{
			Id:        key.ID,
			Name:      key.Name,
			KeyPrefix: key.KeyPrefix,
			Scopes:    key.Scopes.String,
			CreatedAt: key.CreatedAt.String,
		}

		if key.LastUsedAt.Valid {
			keyInfo.LastUsed = key.LastUsedAt.String
		}
		if key.ExpiresAt.Valid {
			keyInfo.ExpiresAt = key.ExpiresAt.String
		}

		keyInfos = append(keyInfos, keyInfo)
	}

	return &types.ListMCPAPIKeysResponse{
		Keys:  keyInfos,
		Total: len(keyInfos),
	}, nil
}
