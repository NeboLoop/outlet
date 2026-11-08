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
	db  db.Querier
	bus *events.Subject // optional; opens and clicks are published here
}

// New creates a new tracking service. Pass the *db.Store so unsubscribes are
// published too. bus may be nil.
func New(db db.Querier, bus *events.Subject) *Service {
	return &Service{db: db, bus: bus}
}

// RecordOpen records an email open by tracking token: a sequence email, or
// a campaign send.
func (s *Service) RecordOpen(ctx context.Context, token string) error {
	if token == "" {
		return ErrInvalidToken
	}
	tok := sql.NullString{String: token, Valid: true}

	if emailRecord, err := s.db.GetEmailByTrackingToken(ctx, tok); err == nil {
		if emailRecord.Kind == "outreach" {
			now := time.Now().UTC()
			_, err := s.db.RecordOutreachOpen(ctx, db.RecordOutreachOpenParams{
				At: sql.NullString{String: now.Format(time.DateTime), Valid: true}, Token: tok,
				CountedAfter: sql.NullString{String: now.Add(-scannerWindow).Format(time.DateTime), Valid: true},
			})
			return err
		}
		if err := s.db.RecordEmailOpen(ctx, emailRecord.ID); err != nil {
			return err
		}
		db.PublishEmail(ctx, s.db, s.bus, events.TopicEmailOpened, db.FromQueue, emailRecord.ID, "")
		return nil
	}

	send, err := s.db.GetCampaignSendByTrackingToken(ctx, tok)
	if err != nil {
		return ErrNotFound
	}
	if err := s.db.RecordCampaignOpen(ctx, send.ID); err != nil {
		return err
	}
	db.PublishEmail(ctx, s.db, s.bus, events.TopicEmailOpened, db.FromCampaign, send.ID, "")
	return s.db.RefreshCampaignEngagement(ctx, send.CampaignID)
}

// scannerWindow: mail scanners open links and images within seconds of
// delivery. On outreach (cold) email, where a click or an open is a signal
// worth trusting, one this soon after sending isn't counted.
const scannerWindow = time.Minute

// RecordClick records a click by tracking token: a sequence or outreach
// email, or a campaign send, where the link is also recorded for per-link stats.
func (s *Service) RecordClick(ctx context.Context, token, linkURL string) error {
	if token == "" {
		return ErrInvalidToken
	}
	tok := sql.NullString{String: token, Valid: true}

	if emailRecord, err := s.db.GetEmailByTrackingToken(ctx, tok); err == nil {
		if emailRecord.Kind == "outreach" {
			now := time.Now().UTC()
			_, err := s.db.RecordOutreachClick(ctx, db.RecordOutreachClickParams{
				At: sql.NullString{String: now.Format(time.DateTime), Valid: true}, Token: tok,
				CountedAfter: sql.NullString{String: now.Add(-scannerWindow).Format(time.DateTime), Valid: true},
			})
			return err
		}
		if err := s.db.RecordEmailClick(ctx, emailRecord.ID); err != nil {
			return err
		}
		db.PublishEmail(ctx, s.db, s.bus, events.TopicEmailClicked, db.FromQueue, emailRecord.ID, linkURL)
		return nil
	}

	send, err := s.db.GetCampaignSendByTrackingToken(ctx, tok)
	if err != nil {
		return ErrNotFound
	}
	if err := s.db.RecordCampaignClick(ctx, send.ID); err != nil {
		return err
	}
	db.PublishEmail(ctx, s.db, s.bus, events.TopicEmailClicked, db.FromCampaign, send.ID, linkURL)
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
