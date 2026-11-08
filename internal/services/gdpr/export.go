package gdpr

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
)

// Data is everything Outlet keeps about one contact, for a GDPR access
// request.
type Data struct {
	ExportedAt         string              `json:"exported_at"`
	Contact            Profile             `json:"contact"`
	Consent            Consent             `json:"consent"`
	Tags               []string            `json:"tags"`
	CampaignSends      []CampaignSend      `json:"campaign_sends"`
	SequenceEmails     []SequenceEmail     `json:"sequence_emails"`
	TransactionalSends []TransactionalSend `json:"transactional_sends"`
	Clicks             []Click             `json:"clicks"`
}

// Profile is the contact's profile.
type Profile struct {
	ID             string `json:"id"`
	Email          string `json:"email"`
	Name           string `json:"name"`
	Status         string `json:"status"`
	Source         string `json:"source"`
	EmailVerified  bool   `json:"email_verified"`
	VerifiedAt     string `json:"verified_at"`
	UnsubscribedAt string `json:"unsubscribed_at"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// Consent is what the contact agreed to.
type Consent struct {
	GDPRConsent   bool   `json:"gdpr_consent"`
	GDPRConsentAt string `json:"gdpr_consent_at"`
	IsSubscribed  bool   `json:"is_subscribed"`
}

// CampaignSend is one campaign sent to the contact.
type CampaignSend struct {
	CampaignID   string `json:"campaign_id"`
	CampaignName string `json:"campaign_name"`
	SentAt       string `json:"sent_at"`
	OpenedAt     string `json:"opened_at"`
	ClickedAt    string `json:"clicked_at"`
}

// SequenceEmail is one sequence email queued or sent to the contact.
type SequenceEmail struct {
	SequenceID   string `json:"sequence_id"`
	SequenceName string `json:"sequence_name"`
	StepNumber   int64  `json:"step_number"`
	Subject      string `json:"subject"`
	Status       string `json:"status"`
	SentAt       string `json:"sent_at"`
	OpenedAt     string `json:"opened_at"`
	ClickedAt    string `json:"clicked_at"`
}

// TransactionalSend is one transactional email sent to the contact.
type TransactionalSend struct {
	TemplateID   string `json:"template_id"`
	TemplateName string `json:"template_name"`
	ToEmail      string `json:"to_email"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
	OpenedAt     string `json:"opened_at"`
	ClickedAt    string `json:"clicked_at"`
}

// Click is one link the contact clicked.
type Click struct {
	URL       string `json:"url"`
	LinkName  string `json:"link_name"`
	ClickedAt string `json:"clicked_at"`
	UserAgent string `json:"user_agent"`
	IPAddress string `json:"ip_address"`
}

// Export gathers everything kept about a brand's contact. It is
// the one export, for the admin screen (which hands it over as a file) and
// MCP; it writes nothing. A contact the brand doesn't have is ErrNotFound.
func Export(ctx context.Context, store *db.Store, orgID, contactID string) (Data, error) {
	contact, err := store.GetContactByOrgID(ctx, db.GetContactByOrgIDParams{
		ID:    contactID,
		OrgID: sql.NullString{String: orgID, Valid: true},
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Data{}, ErrNotFound
	}
	if err != nil {
		return Data{}, err
	}
	id := sql.NullString{String: contact.ID, Valid: true}

	out := Data{
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Contact: Profile{
			ID:             contact.ID,
			Email:          contact.Email,
			Name:           contact.Name,
			Status:         contact.Status.String,
			Source:         contact.Source.String,
			EmailVerified:  contact.EmailVerified == 1,
			VerifiedAt:     contact.VerifiedAt.String,
			UnsubscribedAt: contact.UnsubscribedAt.String,
			CreatedAt:      contact.CreatedAt.String,
			UpdatedAt:      contact.UpdatedAt.String,
		},
		Consent: Consent{
			GDPRConsent:   contact.GdprConsent.Valid && contact.GdprConsent.Int64 == 1,
			GDPRConsentAt: contact.GdprConsentAt.String,
			IsSubscribed:  contact.UnsubscribedAt.String == "",
		},
		Tags:               []string{},
		CampaignSends:      []CampaignSend{},
		SequenceEmails:     []SequenceEmail{},
		TransactionalSends: []TransactionalSend{},
		Clicks:             []Click{},
	}

	tags, err := store.GetContactTags(ctx, id)
	if err != nil {
		return Data{}, err
	}
	for _, t := range tags {
		out.Tags = append(out.Tags, t.Tag)
	}

	campaigns, err := store.GetContactCampaignSends(ctx, contact.ID)
	if err != nil {
		return Data{}, err
	}
	for _, c := range campaigns {
		out.CampaignSends = append(out.CampaignSends, CampaignSend{
			CampaignID: c.CampaignID, CampaignName: c.CampaignName.String,
			SentAt: c.SentAt.String, OpenedAt: c.OpenedAt.String, ClickedAt: c.ClickedAt.String,
		})
	}

	sequences, err := store.GetContactSequenceEmails(ctx, id)
	if err != nil {
		return Data{}, err
	}
	for _, s := range sequences {
		out.SequenceEmails = append(out.SequenceEmails, SequenceEmail{
			SequenceID: s.SequenceID.String, SequenceName: s.SequenceName.String, StepNumber: s.StepNumber.Int64,
			Subject: s.Subject.String, Status: s.Status.String,
			SentAt: s.SentAt.String, OpenedAt: s.OpenedAt.String, ClickedAt: s.ClickedAt.String,
		})
	}

	transactional, err := store.GetContactTransactionalSends(ctx, id)
	if err != nil {
		return Data{}, err
	}
	for _, t := range transactional {
		out.TransactionalSends = append(out.TransactionalSends, TransactionalSend{
			TemplateID: t.TemplateID, TemplateName: t.TemplateName.String, ToEmail: t.ToEmail, Status: t.Status.String,
			CreatedAt: t.CreatedAt.String, OpenedAt: t.OpenedAt.String, ClickedAt: t.ClickedAt.String,
		})
	}

	clicks, err := store.GetContactEmailClicks(ctx, id)
	if err != nil {
		return Data{}, err
	}
	for _, c := range clicks {
		out.Clicks = append(out.Clicks, Click{
			URL: c.LinkUrl, LinkName: c.LinkName.String, ClickedAt: c.ClickedAt,
			UserAgent: c.UserAgent.String, IPAddress: c.IpAddress.String,
		})
	}
	return out, nil
}
