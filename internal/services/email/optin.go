package email

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html"
	"strings"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
)

// An outreach list (email_lists.kind 'outreach') holds prospects: people
// found for outreach, who never signed up. Only outreach sequences, sent from
// Outbound through the brand's mailboxes, write to them. Opt-in mail (a
// newsletter campaign, any sequence other than an outreach sequence, a
// marketing email sent through the API) never goes to one.

// CheckOptInLists refuses an outreach list among the lists opt-in mail goes
// to. A list that no longer exists sends nothing and passes.
func CheckOptInLists(ctx context.Context, q db.Querier, listIDs []int64) error {
	for _, id := range listIDs {
		list, err := q.GetEmailList(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		if list.Kind == "outreach" {
			return errorx.NewBadRequestError(fmt.Sprintf("%q is an outreach list: its people are prospects, who never signed up. Newsletter campaigns and autoresponders go only to newsletter lists; prospects are emailed from Outbound.", list.Name))
		}
	}
	return nil
}

// Every commercial email a brand sends carries its postal address
// (organizations.mailing_address), as CAN-SPAM requires: outreach in its
// plain-text sign-off, opt-in mail (newsletter campaigns and every
// non-transactional sequence step) in its footer. Opt-in mail without one is
// refused where it is turned on: activating a sequence, scheduling or
// sending a campaign.

// MailingAddress is a brand's postal address as its email carries it, in
// plain text. It is the one place the address is formatted.
func MailingAddress(address string) string {
	lines := strings.Split(strings.ReplaceAll(strings.TrimSpace(address), "\r\n", "\n"), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	return strings.Join(lines, "\n")
}

// mailingAddressHTML is MailingAddress for an HTML footer, a line break
// between its lines.
func mailingAddressHTML(address string) string {
	return strings.ReplaceAll(html.EscapeString(MailingAddress(address)), "\n", "<br>")
}

// CheckMailingAddress refuses opt-in mail from a brand that has no mailing
// address set.
func CheckMailingAddress(ctx context.Context, q db.Querier, orgID string) error {
	org, err := q.GetOrganizationByID(ctx, orgID)
	if err != nil {
		return err
	}
	if MailingAddress(org.MailingAddress) == "" {
		return errorx.NewBadRequestError(NoMailingAddress)
	}
	return nil
}

// NoMailingAddress is why opt-in mail from a brand without a mailing address
// is refused.
const NoMailingAddress = "Set the brand's mailing address first: in Settings, Email, or with setup_sending (action: settings, mailing_address). Every newsletter and autoresponder email carries it in its footer, as the law (CAN-SPAM) requires."
