package gdpr

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/services/gdpr"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
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
	caller, err := access.Require(l.ctx, l.svcCtx.DB, contact.OrgID.String, access.Admin)
	if err != nil {
		return nil, err
	}

	deleted, err := gdpr.Delete(l.ctx, l.svcCtx.DB, l.svcCtx.EmailService, gdpr.DeleteRequest{
		OrgID:            contact.OrgID.String,
		ContactID:        contact.ID,
		ProviderLogs:     req.DeleteEmail,
		RequestedBy:      caller.User.ID,
		RequestedByEmail: caller.User.Email,
		Via:              "admin",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to delete contact: %w", err)
	}
	l.Infof("GDPR deletion of contact %s (audit_id: %s)", contact.ID, deleted.AuditID)

	providers := make([]types.GDPRProviderResult, 0, len(deleted.Providers))
	for _, p := range deleted.Providers {
		providers = append(providers, types.GDPRProviderResult{Pathway: p.Pathway, Provider: p.Provider, Result: p.Result, Failed: p.Failed})
	}
	return &types.GDPRDeleteResponse{
		Success:         true,
		Message:         "Contact data deleted successfully (Right to be Forgotten fulfilled)",
		DeletedData:     fmt.Sprintf("Deleted: %v for contact %s", []string{"contact_record", "associated_tags", "sequence_states"}, deleted.Email),
		AuditRecordId:   deleted.AuditID,
		ProviderResults: providers,
	}, nil
}
