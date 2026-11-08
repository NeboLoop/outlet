package tracking

import (
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/events"
)

var (
	ErrInvalidToken = errors.New("invalid or missing token")
	ErrNotFound     = errors.New("record not found")
)

// Service handles email tracking operations
type Service struct {
	db  *db.Queries
	bus *events.Subject // optional; opens and clicks are published here
}

// New creates a new tracking service. bus may be nil.
func New(db *db.Queries, bus *events.Subject) *Service {
	return &Service{db: db, bus: bus}
}

// publish emits an open or click for the email on the event bus.
func (s *Service) publish(topic, status string, e db.GetEmailByTrackingTokenRow) {
	if s.bus == nil {
		return
	}
	_ = events.Emit(s.bus, topic, events.EmailEvent{
		EmailID:   e.ID,
		ContactID: e.ContactID.String,
		Email:     e.Email,
		Status:    status,
		Timestamp: time.Now(),
	})
}

// RecordOpen records an email open by tracking token: a sequence email, or
// a campaign send.
func (s *Service) RecordOpen(ctx context.Context, token string) error {
	if token == "" {
		return ErrInvalidToken
	}
	tok := sql.NullString{String: token, Valid: true}

	if emailRecord, err := s.db.GetEmailByTrackingToken(ctx, tok); err == nil {
		if err := s.db.RecordEmailOpen(ctx, emailRecord.ID); err != nil {
			return err
		}
		s.publish(events.TopicEmailOpened, "opened", emailRecord)
		return nil
	}

	send, err := s.db.GetCampaignSendByTrackingToken(ctx, tok)
	if err != nil {
		return ErrNotFound
	}
	if err := s.db.RecordCampaignOpen(ctx, send.ID); err != nil {
		return err
	}
	return s.db.RefreshCampaignEngagement(ctx, send.CampaignID)
}

// RecordClick records a click by tracking token: a sequence email, or a
// campaign send, where the link is also recorded for per-link stats.
func (s *Service) RecordClick(ctx context.Context, token, linkURL string) error {
	if token == "" {
		return ErrInvalidToken
	}
	tok := sql.NullString{String: token, Valid: true}

	if emailRecord, err := s.db.GetEmailByTrackingToken(ctx, tok); err == nil {
		if err := s.db.RecordEmailClick(ctx, emailRecord.ID); err != nil {
			return err
		}
		s.publish(events.TopicEmailClicked, "clicked", emailRecord)
		return nil
	}

	send, err := s.db.GetCampaignSendByTrackingToken(ctx, tok)
	if err != nil {
		return ErrNotFound
	}
	if err := s.db.RecordCampaignClick(ctx, send.ID); err != nil {
		return err
	}
	if linkURL != "" {
		_, _ = s.db.CreateCampaignClick(ctx, db.CreateCampaignClickParams{
			ID:             uuid.NewString(),
			CampaignSendID: send.ID,
			CampaignID:     send.CampaignID,
			ContactID:      send.ContactID,
			LinkUrl:        linkURL,
		})
	}
	return s.db.RefreshCampaignEngagement(ctx, send.CampaignID)
}

// Unsubscribe unsubscribes a contact by tracking token and cancels pending emails
func (s *Service) Unsubscribe(ctx context.Context, token string) error {
	if token == "" {
		return ErrInvalidToken
	}

	contact, err := s.db.GetContactByTrackingToken(ctx, sql.NullString{String: token, Valid: true})
	if err != nil {
		return ErrNotFound
	}

	if err := s.db.UnsubscribeContact(ctx, contact.ID); err != nil {
		return err
	}

	return s.db.CancelEmailsForContact(ctx, sql.NullString{String: contact.ID, Valid: true})
}
