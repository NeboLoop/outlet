// Package gdpr carries out a person's GDPR requests about their data.
package gdpr

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/email"

	"github.com/google/uuid"
)

// ErrNotFound means the brand has no such contact.
var ErrNotFound = errors.New("contact not found")

// DeleteRequest is a request to delete one contact's data.
type DeleteRequest struct {
	OrgID     string // the brand the contact belongs to
	ContactID string
	// ProviderLogs also asks the brand's email providers to delete what they
	// keep about the address, as far as each allows (email.ForgetRecipient).
	ProviderLogs bool
	// Who asked: a user ID ("api-key" for a brand API key), their email, and
	// the way in (admin or mcp).
	RequestedBy, RequestedByEmail, Via string
}

// Deleted is what a deletion did.
type Deleted struct {
	AuditID   string // the gdpr_deletions row recording it
	Email     string // the address deleted
	Providers []email.ProviderErasure
}

// Delete deletes a brand's contact and what hangs off it, asks the email
// providers to forget the address when the request says so, and records the
// deletion in the GDPR audit log (gdpr_deletions). It is the one deletion,
// for the admin screen and MCP.
func Delete(ctx context.Context, store *db.Store, mail *email.Service, req DeleteRequest) (Deleted, error) {
	contact, err := store.GetContactByOrgID(ctx, db.GetContactByOrgIDParams{
		ID:    req.ContactID,
		OrgID: sql.NullString{String: req.OrgID, Valid: true},
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Deleted{}, ErrNotFound
	}
	if err != nil {
		return Deleted{}, err
	}
	if err := store.DeleteContact(ctx, contact.ID); err != nil {
		return Deleted{}, fmt.Errorf("delete contact: %w", err)
	}

	out := Deleted{AuditID: uuid.NewString(), Email: contact.Email, Providers: []email.ProviderErasure{}}
	if req.ProviderLogs {
		providers, err := mail.ForgetRecipient(ctx, req.OrgID, contact.Email)
		if err != nil {
			providers = []email.ProviderErasure{{Provider: "Email providers", Result: "Couldn't ask them: " + err.Error(), Failed: true}}
		}
		out.Providers = providers
	}

	results, err := json.Marshal(out.Providers)
	if err != nil {
		return Deleted{}, err
	}
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(contact.Email))))
	providersRequested := int64(0)
	if req.ProviderLogs {
		providersRequested = 1
	}
	if err := store.CreateGDPRDeletion(ctx, db.CreateGDPRDeletionParams{
		ID:                 out.AuditID,
		OrgID:              req.OrgID,
		ContactID:          contact.ID,
		EmailSha256:        hex.EncodeToString(sum[:]),
		RequestedBy:        req.RequestedBy,
		RequestedByEmail:   req.RequestedByEmail,
		Via:                req.Via,
		ProvidersRequested: providersRequested,
		ProviderResults:    string(results),
	}); err != nil {
		return Deleted{}, fmt.Errorf("the contact was deleted, but the audit record wasn't saved: %w", err)
	}
	return out, nil
}
