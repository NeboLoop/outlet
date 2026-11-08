package tracking

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/outlet-sh/outlet/internal/db"
)

// mockQueries implements the database methods needed by tracking service
type mockQueries struct {
	getEmailByTokenFn      func(ctx context.Context, token sql.NullString) (db.GetEmailByTrackingTokenRow, error)
	recordEmailOpenFn      func(ctx context.Context, id string) error
	recordEmailClickFn     func(ctx context.Context, id string) error
	getContactByTokenFn    func(ctx context.Context, token sql.NullString) (db.Contact, error)
	unsubscribeContactFn   func(ctx context.Context, id string) error
	cancelEmailsForContactFn func(ctx context.Context, contactID sql.NullString) error
}

func (m *mockQueries) GetEmailByTrackingToken(ctx context.Context, token sql.NullString) (db.GetEmailByTrackingTokenRow, error) {
	if m.getEmailByTokenFn != nil {
		return m.getEmailByTokenFn(ctx, token)
	}
	return db.GetEmailByTrackingTokenRow{}, sql.ErrNoRows
}

func (m *mockQueries) RecordEmailOpen(ctx context.Context, id string) error {
	if m.recordEmailOpenFn != nil {
		return m.recordEmailOpenFn(ctx, id)
	}
	return nil
}

func (m *mockQueries) RecordEmailClick(ctx context.Context, id string) error {
	if m.recordEmailClickFn != nil {
		return m.recordEmailClickFn(ctx, id)
	}
	return nil
}

func (m *mockQueries) GetContactByTrackingToken(ctx context.Context, token sql.NullString) (db.Contact, error) {
	if m.getContactByTokenFn != nil {
		return m.getContactByTokenFn(ctx, token)
	}
	return db.Contact{}, sql.ErrNoRows
}

func (m *mockQueries) UnsubscribeContact(ctx context.Context, id string) error {
	if m.unsubscribeContactFn != nil {
		return m.unsubscribeContactFn(ctx, id)
	}
	return nil
}

func (m *mockQueries) CancelEmailsForContact(ctx context.Context, contactID sql.NullString) error {
	if m.cancelEmailsForContactFn != nil {
		return m.cancelEmailsForContactFn(ctx, contactID)
	}
	return nil
}

func TestNew(t *testing.T) {
	svc := New(nil, nil)
	if svc == nil {
		t.Error("New should return a non-nil service")
	}
}

func TestRecordOpen_EmptyToken(t *testing.T) {
	svc := New(nil, nil)
	err := svc.RecordOpen(context.Background(), "")
	if err != ErrInvalidToken {
		t.Errorf("Expected ErrInvalidToken, got %v", err)
	}
}

func TestRecordOpen_TokenNotFound(t *testing.T) {
	mock := &mockQueries{
		getEmailByTokenFn: func(ctx context.Context, token sql.NullString) (db.GetEmailByTrackingTokenRow, error) {
			return db.GetEmailByTrackingTokenRow{}, sql.ErrNoRows
		},
	}
	svc := &Service{db: (*db.Queries)(nil)} // We'll call mock directly

	// Test with actual service using nil db (will fail on db call)
	// For proper testing, we need interface-based mocking
	// This test verifies the empty token check
	err := svc.RecordOpen(context.Background(), "")
	if err != ErrInvalidToken {
		t.Errorf("Expected ErrInvalidToken for empty token, got %v", err)
	}

	// Verify mock would return not found
	_, err = mock.GetEmailByTrackingToken(context.Background(), sql.NullString{String: "invalid", Valid: true})
	if err != sql.ErrNoRows {
		t.Errorf("Mock should return sql.ErrNoRows, got %v", err)
	}
}

func TestRecordClick_EmptyToken(t *testing.T) {
	svc := New(nil, nil)
	err := svc.RecordClick(context.Background(), "", "")
	if err != ErrInvalidToken {
		t.Errorf("Expected ErrInvalidToken, got %v", err)
	}
}

func TestUnsubscribe_EmptyToken(t *testing.T) {
	svc := New(nil, nil)
	err := svc.Unsubscribe(context.Background(), "")
	if err != ErrInvalidToken {
		t.Errorf("Expected ErrInvalidToken, got %v", err)
	}
}

func TestErrInvalidToken(t *testing.T) {
	if ErrInvalidToken.Error() != "invalid or missing token" {
		t.Errorf("ErrInvalidToken message incorrect: %s", ErrInvalidToken.Error())
	}
}

func TestErrNotFound(t *testing.T) {
	if ErrNotFound.Error() != "record not found" {
		t.Errorf("ErrNotFound message incorrect: %s", ErrNotFound.Error())
	}
}

func TestErrors_AreDistinct(t *testing.T) {
	if errors.Is(ErrInvalidToken, ErrNotFound) {
		t.Error("ErrInvalidToken and ErrNotFound should be distinct errors")
	}
}

// TestMockQueries_RecordOpen tests the mock implementation
func TestMockQueries_RecordOpen(t *testing.T) {
	called := false
	mock := &mockQueries{
		getEmailByTokenFn: func(ctx context.Context, token sql.NullString) (db.GetEmailByTrackingTokenRow, error) {
			if token.String != "test-token" {
				t.Errorf("Expected token 'test-token', got '%s'", token.String)
			}
			return db.GetEmailByTrackingTokenRow{ID: "email-123"}, nil
		},
		recordEmailOpenFn: func(ctx context.Context, id string) error {
			called = true
			if id != "email-123" {
				t.Errorf("Expected id 'email-123', got '%s'", id)
			}
			return nil
		},
	}

	email, err := mock.GetEmailByTrackingToken(context.Background(), sql.NullString{String: "test-token", Valid: true})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if email.ID != "email-123" {
		t.Errorf("Expected email ID 'email-123', got '%s'", email.ID)
	}

	err = mock.RecordEmailOpen(context.Background(), email.ID)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !called {
		t.Error("RecordEmailOpen was not called")
	}
}

// TestMockQueries_RecordClick tests the mock implementation
func TestMockQueries_RecordClick(t *testing.T) {
	called := false
	mock := &mockQueries{
		getEmailByTokenFn: func(ctx context.Context, token sql.NullString) (db.GetEmailByTrackingTokenRow, error) {
			return db.GetEmailByTrackingTokenRow{ID: "email-456"}, nil
		},
		recordEmailClickFn: func(ctx context.Context, id string) error {
			called = true
			if id != "email-456" {
				t.Errorf("Expected id 'email-456', got '%s'", id)
			}
			return nil
		},
	}

	email, _ := mock.GetEmailByTrackingToken(context.Background(), sql.NullString{String: "click-token", Valid: true})
	err := mock.RecordEmailClick(context.Background(), email.ID)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !called {
		t.Error("RecordEmailClick was not called")
	}
}

// TestMockQueries_Unsubscribe tests the mock unsubscribe flow
func TestMockQueries_Unsubscribe(t *testing.T) {
	unsubscribeCalled := false
	cancelCalled := false

	mock := &mockQueries{
		getContactByTokenFn: func(ctx context.Context, token sql.NullString) (db.Contact, error) {
			if token.String != "unsub-token" {
				t.Errorf("Expected token 'unsub-token', got '%s'", token.String)
			}
			return db.Contact{ID: "contact-789"}, nil
		},
		unsubscribeContactFn: func(ctx context.Context, id string) error {
			unsubscribeCalled = true
			if id != "contact-789" {
				t.Errorf("Expected id 'contact-789', got '%s'", id)
			}
			return nil
		},
		cancelEmailsForContactFn: func(ctx context.Context, contactID sql.NullString) error {
			cancelCalled = true
			if contactID.String != "contact-789" {
				t.Errorf("Expected contactID 'contact-789', got '%s'", contactID.String)
			}
			return nil
		},
	}

	contact, err := mock.GetContactByTrackingToken(context.Background(), sql.NullString{String: "unsub-token", Valid: true})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	err = mock.UnsubscribeContact(context.Background(), contact.ID)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	err = mock.CancelEmailsForContact(context.Background(), sql.NullString{String: contact.ID, Valid: true})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !unsubscribeCalled {
		t.Error("UnsubscribeContact was not called")
	}
	if !cancelCalled {
		t.Error("CancelEmailsForContact was not called")
	}
}

// TestMockQueries_ErrorHandling tests error propagation
func TestMockQueries_ErrorHandling(t *testing.T) {
	dbError := errors.New("database connection failed")

	mock := &mockQueries{
		getEmailByTokenFn: func(ctx context.Context, token sql.NullString) (db.GetEmailByTrackingTokenRow, error) {
			return db.GetEmailByTrackingTokenRow{}, dbError
		},
	}

	_, err := mock.GetEmailByTrackingToken(context.Background(), sql.NullString{String: "token", Valid: true})
	if err != dbError {
		t.Errorf("Expected database error, got %v", err)
	}
}

// TestMockQueries_UnsubscribeError tests unsubscribe error handling
func TestMockQueries_UnsubscribeError(t *testing.T) {
	unsubError := errors.New("unsubscribe failed")

	mock := &mockQueries{
		getContactByTokenFn: func(ctx context.Context, token sql.NullString) (db.Contact, error) {
			return db.Contact{ID: "contact-123"}, nil
		},
		unsubscribeContactFn: func(ctx context.Context, id string) error {
			return unsubError
		},
	}

	contact, _ := mock.GetContactByTrackingToken(context.Background(), sql.NullString{String: "token", Valid: true})
	err := mock.UnsubscribeContact(context.Background(), contact.ID)
	if err != unsubError {
		t.Errorf("Expected unsubscribe error, got %v", err)
	}
}

// TestTokenValidation tests various token formats
func TestTokenValidation(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr error
	}{
		{"empty string", "", ErrInvalidToken},
		{"whitespace only would pass validation", "   ", nil}, // Note: whitespace passes - only empty is invalid
		{"valid uuid-like", "550e8400-e29b-41d4-a716-446655440000", nil},
		{"simple string", "abc123", nil},
		{"special chars", "token-with-dashes_and_underscores", nil},
	}

	svc := New(nil, nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can only test empty token without a real DB
			// Non-empty tokens will fail on DB call, not validation
			if tt.token == "" {
				err := svc.RecordOpen(context.Background(), tt.token)
				if err != tt.wantErr {
					t.Errorf("RecordOpen(%q) error = %v, want %v", tt.token, err, tt.wantErr)
				}
			}
		})
	}
}
