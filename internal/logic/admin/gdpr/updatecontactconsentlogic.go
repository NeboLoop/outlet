package gdpr

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateContactConsentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateContactConsentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateContactConsentLogic {
	return &UpdateContactConsentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateContactConsentLogic) UpdateContactConsent(req *types.UpdateGDPRConsentRequest) (resp *types.GDPRConsentInfo, err error) {
	// Verify contact exists
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

	// Update GDPR consent
	consentVal := int64(0)
	if req.GDPRConsent {
		consentVal = 1
	}

	if err := l.svcCtx.DB.UpdateContactGDPR(l.ctx, db.UpdateContactGDPRParams{
		ID:      req.ContactId,
		Consent: sql.NullInt64{Int64: consentVal, Valid: true},
	}); err != nil {
		return nil, fmt.Errorf("failed to update GDPR consent: %w", err)
	}

	// Handle marketing consent (unsubscribe if no longer consenting)
	if !req.MarketingConsent {
		// Unsubscribe the contact
		if err := l.svcCtx.DB.UnsubscribeContact(l.ctx, req.ContactId); err != nil {
			l.Errorf("Failed to unsubscribe contact: %v", err)
		}
	}

	// Return updated consent info
	return NewGetContactConsentLogic(l.ctx, l.svcCtx).GetContactConsent(&types.GDPRConsentRequest{
		ContactId: req.ContactId,
	})
}
