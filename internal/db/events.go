package db

import (
	"context"
	"database/sql"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/events"
)

// The Store publishes contact and email events on the event bus from the
// writes that make them happen, so every path that creates a contact,
// unsubscribes one, or records a send is heard by webhooks, whichever handler,
// worker or tool made the change. Writes inside ExecTx are not published.

// SetEvents makes the Store publish on bus. Without it nothing is published.
func (s *Store) SetEvents(bus *events.Subject) {
	s.events = bus
}

// EmailSource is the table that records a send.
type EmailSource int

const (
	FromQueue         EmailSource = iota // email_queue: sequence and queued mail
	FromCampaign                         // campaign_sends
	FromTransactional                    // transactional_sends
)

// PublishEmail publishes topic (email.sent, email.opened, email.clicked) on
// bus for the send with this id, looked up through q. clickedURL is set for
// clicks only. A nil bus publishes nothing.
func PublishEmail(ctx context.Context, q Querier, bus *events.Subject, topic string, source EmailSource, id, clickedURL string) {
	if bus == nil {
		return
	}
	evt := events.EmailEvent{
		EmailID:    id,
		Status:     strings.TrimPrefix(topic, "email."),
		ClickedURL: clickedURL,
		Timestamp:  time.Now().UTC(),
	}
	switch source {
	case FromQueue:
		row, err := q.GetQueuedEmailEvent(ctx, id)
		if err != nil {
			publishFailed(topic, id, err)
			return
		}
		evt.OrgID, evt.ContactID, evt.Email = row.OrgID.String, row.ContactID, row.Email
		evt.Subject, evt.SequenceID = row.Subject, row.SequenceID
	case FromCampaign:
		row, err := q.GetCampaignSendEvent(ctx, id)
		if err != nil {
			publishFailed(topic, id, err)
			return
		}
		evt.OrgID, evt.ContactID, evt.Email = row.OrgID, row.ContactID, row.Email
		evt.Subject, evt.CampaignID = row.Subject, row.CampaignID
		if row.ListID != 0 {
			evt.ListID = strconv.FormatInt(row.ListID, 10)
		}
	case FromTransactional:
		row, err := q.GetTransactionalSendEvent(ctx, id)
		if err != nil {
			publishFailed(topic, id, err)
			return
		}
		evt.OrgID, evt.ContactID, evt.Email, evt.Subject = row.OrgID, row.ContactID, row.ToEmail, row.Subject
	}
	publish(bus, topic, evt)
}

// publishContact publishes topic for the contact with this id.
func (s *Store) publishContact(ctx context.Context, topic, contactID, listID string) {
	if s.events == nil {
		return
	}
	c, err := s.Queries.GetContact(ctx, contactID)
	if err != nil {
		publishFailed(topic, contactID, err)
		return
	}
	publish(s.events, topic, contactEvent(c, listID))
}

func contactEvent(c Contact, listID string) events.ContactEvent {
	return events.ContactEvent{
		OrgID:     c.OrgID.String,
		ContactID: c.ID,
		Email:     c.Email,
		ListID:    listID,
		Source:    c.Source.String,
		Timestamp: time.Now().UTC(),
	}
}

func publish(bus *events.Subject, topic string, evt any) {
	if err := events.Emit(bus, topic, evt); err != nil {
		slog.Error("publish event", "topic", topic, "error", err)
	}
}

func publishFailed(topic, id string, err error) {
	slog.Error("publish event: look up the record", "topic", topic, "id", id, "error", err)
}

// CreateContact creates a contact and publishes contact.created.
func (s *Store) CreateContact(ctx context.Context, arg CreateContactParams) (Contact, error) {
	c, err := s.Queries.CreateContact(ctx, arg)
	if err == nil && s.events != nil {
		publish(s.events, events.TopicContactCreated, contactEvent(c, ""))
	}
	return c, err
}

// UnsubscribeContact unsubscribes a contact from everything and publishes
// contact.unsubscribed.
func (s *Store) UnsubscribeContact(ctx context.Context, id string) error {
	if err := s.Queries.UnsubscribeContact(ctx, id); err != nil {
		return err
	}
	s.publishContact(ctx, events.TopicContactUnsubscribed, id, "")
	return nil
}

// GlobalUnsubscribeByOrgAndEmail unsubscribes a brand's contact by address
// and publishes contact.unsubscribed.
func (s *Store) GlobalUnsubscribeByOrgAndEmail(ctx context.Context, arg GlobalUnsubscribeByOrgAndEmailParams) error {
	if err := s.Queries.GlobalUnsubscribeByOrgAndEmail(ctx, arg); err != nil {
		return err
	}
	if s.events == nil {
		return nil
	}
	c, err := s.Queries.GetContactByOrgAndEmail(ctx, GetContactByOrgAndEmailParams{OrgID: arg.OrgID, Email: arg.Email})
	if err == sql.ErrNoRows {
		return nil // nobody to unsubscribe
	}
	if err != nil {
		publishFailed(events.TopicContactUnsubscribed, arg.Email, err)
		return nil
	}
	publish(s.events, events.TopicContactUnsubscribed, contactEvent(c, ""))
	return nil
}

// UnsubscribeFromList unsubscribes a contact from one list and publishes
// contact.unsubscribed with the list.
func (s *Store) UnsubscribeFromList(ctx context.Context, arg UnsubscribeFromListParams) error {
	if err := s.Queries.UnsubscribeFromList(ctx, arg); err != nil {
		return err
	}
	s.publishContact(ctx, events.TopicContactUnsubscribed, arg.ContactID, strconv.FormatInt(arg.ListID, 10))
	return nil
}

// MarkEmailSent records a queued email as sent and publishes email.sent.
func (s *Store) MarkEmailSent(ctx context.Context, id string) error {
	if err := s.Queries.MarkEmailSent(ctx, id); err != nil {
		return err
	}
	PublishEmail(ctx, s.Queries, s.events, events.TopicEmailSent, FromQueue, id, "")
	return nil
}

// MarkCampaignSendSent records a campaign send as sent and publishes email.sent.
func (s *Store) MarkCampaignSendSent(ctx context.Context, id string) error {
	if err := s.Queries.MarkCampaignSendSent(ctx, id); err != nil {
		return err
	}
	PublishEmail(ctx, s.Queries, s.events, events.TopicEmailSent, FromCampaign, id, "")
	return nil
}

// UpdateTransactionalSendStatus records a transactional send's status and
// publishes email.sent when it was sent.
func (s *Store) UpdateTransactionalSendStatus(ctx context.Context, arg UpdateTransactionalSendStatusParams) error {
	if err := s.Queries.UpdateTransactionalSendStatus(ctx, arg); err != nil {
		return err
	}
	if arg.Status.String == "sent" {
		PublishEmail(ctx, s.Queries, s.events, events.TopicEmailSent, FromTransactional, arg.ID, "")
	}
	return nil
}
