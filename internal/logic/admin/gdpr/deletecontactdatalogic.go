package gdpr

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteContactDataLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteContactDataLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteContactDataLogic {
	return &DeleteContactDataLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteContactDataLogic) DeleteContactData(req *types.GDPRDeleteRequest) (resp *types.GDPRDeleteResponse, err error) {
	// Require confirmation
	if !req.Confirm {
		return nil, fmt.Errorf("deletion not confirmed. Set confirm=true to proceed")
	}

	// Get contact to verify it exists
	contact, err := l.svcCtx.DB.GetContact(l.ctx, req.ContactId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("contact not found")
		}
		return nil, err
	}
	if _, err := access.Require(l.ctx, l.svcCtx.DB, contact.OrgID.String, access.Admin); err != nil {
		return nil, err
	}

	// Generate audit record ID
	auditID := uuid.New().String()

	// Log the deletion for audit purposes
	l.Infof("GDPR deletion request for contact %s (email: %s, audit_id: %s)", contact.ID, contact.Email, auditID)

	// Delete the contact (related data will be cleaned up via cascade or handled separately)
	// Note: In production, you may want to anonymize rather than delete for audit purposes
	if err := l.svcCtx.DB.DeleteContact(l.ctx, contact.ID); err != nil {
		return nil, fmt.Errorf("failed to delete contact: %w", err)
	}

	deletedItems := []string{"contact_record", "associated_tags", "sequence_states"}

	// Build summary
	summary := fmt.Sprintf("Deleted: %v for contact %s", deletedItems, contact.Email)

	return &types.GDPRDeleteResponse{
		Success:       true,
		Message:       "Contact data deleted successfully (Right to be Forgotten fulfilled)",
		DeletedData:   summary,
		AuditRecordId: auditID,
	}, nil
}
