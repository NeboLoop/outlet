package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/outlet-sh/outlet/internal/events"
)

// Store provides all database operations with transaction support
type Store struct {
	*Queries
	db       *sql.DB
	events   *events.Subject                                  // optional; see events.go
	tagAdded func(ctx context.Context, contactID, tag string) // optional; see OnTagAdded
	// optional; see OnSubscriptionActivated
	subscriptionActivated func(ctx context.Context, contactID string, listID int64) error
}

// OnTagAdded sets what runs when a contact is given a tag it didn't have
// (the sequences waiting on that tag), whichever handler, worker or tool
// tagged it. Tags added inside ExecTx don't run it.
func (s *Store) OnTagAdded(fn func(ctx context.Context, contactID, tag string)) {
	s.tagAdded = fn
}

// OnSubscriptionActivated sets what runs when a contact's subscription to a
// list becomes active (the list's "On signup" autoresponders and the entry
// rules waiting on the list), whichever handler, worker or tool made it
// active: SubscribeToList, ConfirmListSubscription, and AddSubscriber when
// asked to. Subscriptions made inside ExecTx don't run it.
func (s *Store) OnSubscriptionActivated(fn func(ctx context.Context, contactID string, listID int64) error) {
	s.subscriptionActivated = fn
}

// activated runs OnSubscriptionActivated. The subscription stands either
// way, so a failure is logged.
func (s *Store) activated(ctx context.Context, contactID string, listID int64) {
	if s.subscriptionActivated == nil {
		return
	}
	if err := s.subscriptionActivated(ctx, contactID, listID); err != nil {
		slog.Error("start sign-up emails", "contact", contactID, "list", listID, "error", err)
	}
}

// SubscribeToList makes a contact an active subscriber of a list and runs
// OnSubscriptionActivated.
func (s *Store) SubscribeToList(ctx context.Context, arg SubscribeToListParams) (ListSubscriber, error) {
	sub, err := s.Queries.SubscribeToList(ctx, arg)
	if err == nil {
		s.activated(ctx, sub.ContactID, sub.ListID)
	}
	return sub, err
}

// ConfirmListSubscription confirms a pending double opt-in subscription and
// runs OnSubscriptionActivated.
func (s *Store) ConfirmListSubscription(ctx context.Context, token sql.NullString) (ListSubscriber, error) {
	sub, err := s.Queries.ConfirmListSubscription(ctx, token)
	if err == nil {
		s.activated(ctx, sub.ContactID, sub.ListID)
	}
	return sub, err
}

// AddContactTag tags a contact. A tag the contact already has is left as it is
// (sql.ErrNoRows); a new one runs OnTagAdded.
func (s *Store) AddContactTag(ctx context.Context, arg AddContactTagParams) (ContactTag, error) {
	t, err := s.Queries.AddContactTag(ctx, arg)
	if err == nil && s.tagAdded != nil {
		s.tagAdded(ctx, arg.ContactID.String, arg.Tag)
	}
	return t, err
}

// NewStore creates a new Store instance
func NewStore(db *sql.DB) *Store {
	return &Store{
		Queries: New(db),
		db:      db,
	}
}

// ExecTx executes a function within a database transaction
// If the function returns an error, the transaction is rolled back
// Otherwise, the transaction is committed
func (s *Store) ExecTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	q := New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx error: %v, rb error: %w", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ExecTxWithResult executes a function within a transaction and returns a result
// If the function returns an error, the transaction is rolled back
// Otherwise, the transaction is committed and the result is returned
func ExecTxWithResult[T any](s *Store, ctx context.Context, fn func(*Queries) (T, error)) (T, error) {
	var result T

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("failed to begin transaction: %w", err)
	}

	q := New(tx)
	result, err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return result, fmt.Errorf("tx error: %v, rb error: %w", err, rbErr)
		}
		return result, err
	}

	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result, nil
}

// GetDB returns the underlying database connection
func (s *Store) GetDB() *sql.DB {
	return s.db
}

// Ping verifies the database connection is alive
func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}
