package emailconfig

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteDomainIdentityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteDomainIdentityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteDomainIdentityLogic {
	return &DeleteDomainIdentityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteDomainIdentityLogic) DeleteDomainIdentity(req *types.DeleteDomainIdentityRequest) (resp *types.Response, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, req.OrgId, access.Admin); err != nil {
		return nil, err
	}

	// Get the domain identity to verify it exists and belongs to the org
	identity, err := l.svcCtx.DB.GetDomainIdentity(l.ctx, req.Id)
	if err != nil {
		return nil, errorx.NewNotFoundError("Domain identity not found")
	}

	// Verify the identity belongs to the requested org
	if identity.OrgID != req.OrgId {
		return nil, errorx.NewNotFoundError("Domain identity not found")
	}

	// A domain another brand still holds stays in SES: only this brand's
	// record goes.
	shared, err := email.DomainShared(l.ctx, l.svcCtx.DB, req.OrgId, identity.Domain)
	if err != nil {
		return nil, err
	}

	// The AWS account of the domain's pathway
	region, accessKey, secretKey, err := email.AWSCredentials(l.ctx, l.svcCtx.DB, l.svcCtx.CryptoService, req.OrgId, identity.PathwayID.String)
	if shared {
		l.Infof("Domain %s is still held by another brand; leaving it in SES", identity.Domain)
	} else if err != nil {
		// Log but continue - we still want to delete from our database
		l.Errorf("Failed to get AWS credentials: %v", err)
	} else {
		// Try to delete from AWS SES (don't fail if this doesn't work)
		if deleteErr := email.DeleteDomainIdentity(l.ctx, region, accessKey, secretKey, identity.Domain); deleteErr != nil {
			l.Errorf("Failed to delete domain identity from AWS SES: %v", deleteErr)
			// Continue anyway - the user may have already deleted it from SES
		}
	}

	// Delete from database
	if err := l.svcCtx.DB.DeleteDomainIdentity(l.ctx, req.Id); err != nil {
		l.Errorf("Failed to delete domain identity from database: %v", err)
		return nil, errorx.NewInternalError("Failed to delete domain identity")
	}

	return &types.Response{
		Success: true,
		Message: "Domain identity deleted successfully",
	}, nil
}
