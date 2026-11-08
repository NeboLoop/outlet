package db

import (
	"context"
	"database/sql"
	"errors"
	"net/mail"
	"strings"

	"github.com/google/uuid"
)

// ErrBlockedDomain means the address is at a domain the brand blocked, so it
// can't become a contact or subscribe.
var ErrBlockedDomain = errors.New("this email address's domain is blocked")

// ErrInvalidEmail means the address isn't one mail can be sent to.
var ErrInvalidEmail = errors.New("not a valid email address")

// RefuseBlockedDomain returns ErrBlockedDomain when email is at a domain the
// brand blocked, and counts the attempt on that domain.
func (s *Store) RefuseBlockedDomain(ctx context.Context, orgID, email string) error {
	at := strings.LastIndex(email, "@")
	if orgID == "" || at < 0 {
		return nil
	}
	domain := strings.ToLower(strings.TrimSpace(email[at+1:]))
	blocked, err := s.Queries.IsDomainBlocked(ctx, IsDomainBlockedParams{OrgID: orgID, Domain: domain})
	if err != nil {
		return err
	}
	if blocked == 0 {
		return nil
	}
	if err := s.Queries.IncrementBlockedDomainAttempts(ctx, IncrementBlockedDomainAttemptsParams{OrgID: orgID, Domain: domain}); err != nil {
		return err
	}
	return ErrBlockedDomain
}

// AddSubscriber puts a brand's contact on a list as an active subscriber, the
// way an admin adds people by hand or by CSV import: no confirmation email.
// It finds the contact by address or creates it, and takes a given name. A
// contact already on the list, unsubscribed included, is left as it is:
// added is false. listID 0 adds the contact to no list.
func (s *Store) AddSubscriber(ctx context.Context, orgID string, listID int64, email, name, source string) (added bool, err error) {
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)
	if a, err := mail.ParseAddress(email); err != nil || a.Address != email {
		return false, ErrInvalidEmail
	}
	if err := s.RefuseBlockedDomain(ctx, orgID, email); err != nil {
		return false, err
	}

	org := sql.NullString{String: orgID, Valid: true}
	contact, err := s.Queries.GetContactByOrgAndEmail(ctx, GetContactByOrgAndEmailParams{OrgID: org, Email: email})
	switch {
	case errors.Is(err, sql.ErrNoRows):
		contact, err = s.CreateContact(ctx, CreateContactParams{
			ID: uuid.NewString(), OrgID: org, Email: email, Name: name,
			Source: sql.NullString{String: source, Valid: source != ""},
		})
		if err != nil {
			return false, err
		}
	case err != nil:
		return false, err
	case name != "" && name != contact.Name:
		if err := s.Queries.UpdateContactName(ctx, UpdateContactNameParams{ID: contact.ID, Name: name}); err != nil {
			return false, err
		}
	}

	if listID == 0 {
		return false, nil
	}
	exists, err := s.Queries.CheckListSubscription(ctx, CheckListSubscriptionParams{ListID: listID, ContactID: contact.ID})
	if err != nil || exists != 0 {
		return false, err
	}
	_, err = s.Queries.CreateListSubscriber(ctx, CreateListSubscriberParams{
		ID: uuid.NewString(), ListID: listID, ContactID: contact.ID,
		Status: sql.NullString{String: "active", Valid: true},
	})
	return err == nil, err
}
