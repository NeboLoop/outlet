package gdpr

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetContactConsentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetContactConsentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetContactConsentLogic {
	return &GetContactConsentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetContactConsentLogic) GetContactConsent(req *types.GDPRConsentRequest) (resp *types.GDPRConsentInfo, err error) {
	// Get contact by ID
	contact, err := l.svcCtx.DB.GetContact(l.ctx, req.ContactId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("contact not found")
		}
		return nil, err
	}
	if _, err := access.Require(l.ctx, l.svcCtx.DB, contact.OrgID.String, access.Member); err != nil {
		return nil, err
	}

	return &types.GDPRConsentInfo{
		ContactId:          contact.ID,
		Email:              contact.Email,
		GDPRConsent:        contact.GdprConsent.Valid && contact.GdprConsent.Int64 == 1,
		GDPRConsentAt:      contact.GdprConsentAt.String,
		MarketingConsent:   contact.UnsubscribedAt.String == "", // If not unsubscribed, they consent to marketing
		MarketingConsentAt: contact.CreatedAt.String,            // When they subscribed
		CreatedAt:          contact.CreatedAt.String,
	}, nil
}
