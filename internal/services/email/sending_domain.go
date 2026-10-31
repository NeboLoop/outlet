package email

import (
	"context"
	"fmt"
	"strings"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
)

// Brands verify their sending domains on a shared SES account, and SES
// accepts any From address on a verified domain. So a domain belongs to one
// brand (its domain identity), and a brand sends only from its own domains.

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

// CheckSender refuses a From address that isn't on one of the sending
// brand's domains. An empty From is the platform's own default sender.
func CheckSender(ctx context.Context, q db.Querier, orgID, from string) error {
	from = strings.TrimSpace(from)
	if from == "" {
		return nil
	}
	domain := ExtractDomainFromEmail(from)
	if domain == "" {
		return errorx.NewBadRequestError(fmt.Sprintf("%s is not an email address.", from))
	}
	n, err := q.CountBrandDomain(ctx, db.CountBrandDomainParams{OrgID: orgID, Domain: domain})
	if err != nil {
		return err
	}
	if n == 0 {
		return errorx.NewBadRequestError(fmt.Sprintf("This brand can't send from %s. Add %s as one of its sending domains first.", from, domain))
	}
	return nil
}
