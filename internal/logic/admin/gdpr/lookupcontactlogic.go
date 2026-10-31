package gdpr

import (
	"context"
	"database/sql"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type LookupContactLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLookupContactLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LookupContactLogic {
	return &LookupContactLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LookupContactLogic) LookupContact(req *types.GDPRLookupRequest) (resp *types.GDPRLookupResponse, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	if req.Email == "" {
		return &types.GDPRLookupResponse{
			Found: false,
		}, nil
	}

	// Look up the contact by email in this brand
	contact, err := l.svcCtx.DB.GetContactByOrgAndEmail(l.ctx, db.GetContactByOrgAndEmailParams{
		OrgID: sql.NullString{String: orgID, Valid: true},
		Email: req.Email,
	})
	if err != nil {
		// Not found is not an error, just return found=false
		return &types.GDPRLookupResponse{
			Found: false,
		}, nil
	}

	return &types.GDPRLookupResponse{
		Found:     true,
		ContactId: contact.ID,
		Email:     contact.Email,
	}, nil
}
