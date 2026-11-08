package subscribers

import (
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/types"
)

// contactToSubscriberInfo converts a database contact to a SubscriberInfo response
func contactToSubscriberInfo(c db.Contact) *types.SubscriberInfo {
	verifiedAt := ""
	if c.VerifiedAt.Valid {
		verifiedAt = c.VerifiedAt.String
	}

	unsubscribedAt := ""
	if c.UnsubscribedAt.Valid {
		unsubscribedAt = c.UnsubscribedAt.String
	}

	blockedAt := ""
	if c.BlockedAt.Valid {
		blockedAt = c.BlockedAt.String
	}

	createdAt := ""
	if c.CreatedAt.Valid {
		createdAt = c.CreatedAt.String
	}

	return &types.SubscriberInfo{
		Id:                  c.ID,
		Name:                c.Name,
		Email:               c.Email,
		Status:              c.Status.String,
		EmailVerified:       c.EmailVerified == 1,
		Source:              c.Source.String,
		CreatedAt:           createdAt,
		VerifiedAt:          verifiedAt,
		UnsubscribedAt:      unsubscribedAt,
		BlockedAt:           blockedAt,
		ValidationStatus:    c.ValidationStatus.String,
		ValidationSubStatus: c.ValidationSubStatus.String,
	}
}
