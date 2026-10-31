package apikeys

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RevokeMCPAPIKeyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRevokeMCPAPIKeyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevokeMCPAPIKeyLogic {
	return &RevokeMCPAPIKeyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RevokeMCPAPIKeyLogic) RevokeMCPAPIKey(req *types.RevokeMCPAPIKeyRequest) (resp *types.RevokeMCPAPIKeyResponse, err error) {
	// Get user ID from context
	user, ok := access.User(l.ctx)
	if !ok {
		return nil, errorx.NewUnauthorizedError("Unauthorized")
	}

	// Validate key ID is not empty
	if req.Id == "" {
		return nil, errorx.NewBadRequestError("Invalid API key ID")
	}

	// Revoke the key (this sets revoked_at timestamp). MCP reads keys from
	// the database on every request, so it stops working at once.
	revoked, err := l.svcCtx.DB.RevokeMCPAPIKey(l.ctx, db.RevokeMCPAPIKeyParams{ID: req.Id, UserID: user.ID})
	if err != nil {
		l.Errorf("Failed to revoke API key: %v", err)
		return nil, errorx.NewInternalError("Failed to revoke API key")
	}
	if revoked == 0 {
		return nil, errorx.NewNotFoundError("API key not found")
	}

	return &types.RevokeMCPAPIKeyResponse{
		Success: true,
		Message: "API key revoked successfully",
	}, nil
}
