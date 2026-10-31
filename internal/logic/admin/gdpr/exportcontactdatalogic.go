package gdpr

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type ExportContactDataLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewExportContactDataLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ExportContactDataLogic {
	return &ExportContactDataLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ContactExport represents the full data export for a contact
type ContactExport struct {
	ExportedAt string                 `json:"exported_at"`
	ContactID  string                 `json:"contact_id"`
	Profile    map[string]interface{} `json:"profile"`
	Tags       []string               `json:"tags"`
	Activity   []map[string]interface{} `json:"activity,omitempty"`
	Consent    map[string]interface{} `json:"consent"`
}

func (l *ExportContactDataLogic) ExportContactData(req *types.GDPRExportRequest) (resp *types.GDPRExportResponse, err error) {
	// Get contact
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

	// Build export data
	export := ContactExport{
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		ContactID:  contact.ID,
		Profile: map[string]interface{}{
			"email":             contact.Email,
			"name":              contact.Name,
			"source":            contact.Source.String,
			"email_verified":    contact.EmailVerified == 1,
			"verified_at":       contact.VerifiedAt.String,
			"unsubscribed_at":   contact.UnsubscribedAt.String,
			"created_at":        contact.CreatedAt.String,
			"updated_at":        contact.UpdatedAt.String,
		},
		Consent: map[string]interface{}{
			"gdpr_consent":    contact.GdprConsent.Valid && contact.GdprConsent.Int64 == 1,
			"gdpr_consent_at": contact.GdprConsentAt.String,
			"is_subscribed":   contact.UnsubscribedAt.String == "",
		},
	}

	// Get tags
	tags, err := l.svcCtx.DB.GetContactTags(l.ctx, sql.NullString{String: contact.ID, Valid: true})
	if err == nil {
		export.Tags = make([]string, len(tags))
		for i, t := range tags {
			export.Tags[i] = t.Tag
		}
	}

	// Get email activity
	export.Activity = l.getEmailActivity(contact.ID)

	// Write export to file
	exportDir := filepath.Join(".", "exports", "gdpr")
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create export directory: %w", err)
	}

	exportID := uuid.New().String()
	filename := fmt.Sprintf("gdpr-export-%s-%s.json", contact.ID[:8], exportID[:8])
	filePath := filepath.Join(exportDir, filename)

	exportData, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize export: %w", err)
	}

	if err := os.WriteFile(filePath, exportData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write export file: %w", err)
	}

	// Generate download URL (expires in 24 hours)
	expiresAt := time.Now().Add(24 * time.Hour)

	return &types.GDPRExportResponse{
		DownloadURL: fmt.Sprintf("/api/admin/gdpr/export/%s/download", exportID),
		ExpiresAt:   expiresAt.Format(time.RFC3339),
		Format:      "json",
	}, nil
}

// getEmailActivity retrieves all email activity for a contact
func (l *ExportContactDataLogic) getEmailActivity(contactID string) []map[string]interface{} {
	var activity []map[string]interface{}
	contactIDNull := sql.NullString{String: contactID, Valid: true}

	// Get campaign sends
	campaignSends, err := l.svcCtx.DB.GetContactCampaignSends(l.ctx, contactID)
	if err == nil {
		for _, cs := range campaignSends {
			activity = append(activity, map[string]interface{}{
				"type":          "campaign",
				"campaign_id":   cs.CampaignID,
				"campaign_name": cs.CampaignName.String,
				"sent_at":       cs.SentAt.String,
				"opened_at":     cs.OpenedAt.String,
				"clicked_at":    cs.ClickedAt.String,
			})
		}
	}

	// Get transactional sends
	transactionalSends, err := l.svcCtx.DB.GetContactTransactionalSends(l.ctx, contactIDNull)
	if err == nil {
		for _, ts := range transactionalSends {
			activity = append(activity, map[string]interface{}{
				"type":          "transactional",
				"template_id":   ts.TemplateID,
				"template_name": ts.TemplateName.String,
				"to_email":      ts.ToEmail,
				"status":        ts.Status.String,
				"opened_at":     ts.OpenedAt.String,
				"clicked_at":    ts.ClickedAt.String,
				"created_at":    ts.CreatedAt.String,
			})
		}
	}

	// Get sequence emails
	sequenceEmails, err := l.svcCtx.DB.GetContactSequenceEmails(l.ctx, contactIDNull)
	if err == nil {
		for _, se := range sequenceEmails {
			activity = append(activity, map[string]interface{}{
				"type":          "sequence",
				"sequence_id":   se.SequenceID.String,
				"sequence_name": se.SequenceName.String,
				"step_number":   se.StepNumber.Int64,
				"subject":       se.Subject.String,
				"status":        se.Status.String,
				"sent_at":       se.SentAt.String,
				"opened_at":     se.OpenedAt.String,
				"clicked_at":    se.ClickedAt.String,
			})
		}
	}

	// Get email clicks
	clicks, err := l.svcCtx.DB.GetContactEmailClicks(l.ctx, contactIDNull)
	if err == nil {
		for _, c := range clicks {
			activity = append(activity, map[string]interface{}{
				"type":       "click",
				"link_url":   c.LinkUrl,
				"link_name":  c.LinkName.String,
				"clicked_at": c.ClickedAt,
				"user_agent": c.UserAgent.String,
				"ip_address": c.IpAddress.String,
			})
		}
	}

	return activity
}
