package email

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
)

// Brands verify their sending domains on a shared SES account, and SES
// accepts any From address on a verified domain. So a domain belongs to one
// brand (its domain identity) and one of its sending pathways, and a brand's
// mail sends only from the domains of the pathway carrying it.

// ClaimDomain refuses a domain another brand already holds.
func ClaimDomain(ctx context.Context, q db.Querier, orgID, domain string) error {
	shared, err := DomainShared(ctx, q, orgID, domain)
	if err != nil {
		return err
	}
	if shared {
		return errorx.NewBadRequestError(fmt.Sprintf("Another brand already uses %s.", domain))
	}
	return nil
}

// DomainShared reports whether a brand other than orgID holds the domain
// too. A domain another brand still holds stays in SES when one brand
// removes it.
func DomainShared(ctx context.Context, q db.Querier, orgID, domain string) (bool, error) {
	n, err := q.CountOtherBrandsWithDomain(ctx, db.CountOtherBrandsWithDomainParams{OrgID: orgID, Domain: domain})
	return n > 0, err
}

// CheckSender refuses a From address whose domain doesn't belong to the
// brand's pathway carrying this kind of mail. An empty From is the
// platform's own default sender.
func CheckSender(ctx context.Context, q db.Querier, orgID string, kind Kind, from string) error {
	if strings.TrimSpace(from) == "" {
		return nil
	}
	p, err := q.GetPathwayForKind(ctx, db.GetPathwayForKindParams{OrgID: orgID, Kind: kind.Name()})
	if errors.Is(err, sql.ErrNoRows) {
		return errorx.NewBadRequestError(fmt.Sprintf("This brand has no sending pathway for %s mail. Add one in Settings, Sending.", kind.Name()))
	}
	if err != nil {
		return err
	}
	return checkPathwayDomain(ctx, q, p, kind, from)
}

func checkPathwayDomain(ctx context.Context, q db.Querier, p db.SendingPathway, kind Kind, from string) error {
	from = strings.TrimSpace(from)
	domain := ExtractDomainFromEmail(from)
	if domain == "" {
		return errorx.NewBadRequestError(fmt.Sprintf("%s is not an email address.", from))
	}
	n, err := q.CountPathwayDomain(ctx, db.CountPathwayDomainParams{PathwayID: sql.NullString{String: p.ID, Valid: true}, Domain: domain})
	if err != nil {
		return err
	}
	if n == 0 {
		return errorx.NewBadRequestError(fmt.Sprintf("This brand can't send %s mail from %s. Add %s to its %q sending pathway first.", kind.Name(), from, domain, p.Name))
	}
	return nil
}
