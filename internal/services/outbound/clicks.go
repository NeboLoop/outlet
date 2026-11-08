package outbound

import (
	"context"
	"html"
	"regexp"
	"strings"

	"github.com/outlet-sh/outlet/internal/db"
)

// Open and click tracking are chosen per campaign and off by default. Links
// and the open pixel go through the sending domain's own go. subdomain, never
// a shared host, as Outlet's signed tracking URLs (/api/e/c, /api/e/o), and
// the tracking service records them on the email.

var linkRe = regexp.MustCompile(`https?://[^\s<>"']+`)

func htmlEscape(s string) string { return html.EscapeString(s) }

// trackLinks swaps each link in a text body for a signed tracked one on host.
// It runs when the email is sent, so drafts show the real links and an edit
// is tracked too.
func (s *Service) trackLinks(body, host, token string) string {
	return linkRe.ReplaceAllStringFunc(body, func(raw string) string {
		target := strings.TrimRight(raw, ".,;:!?)]")
		return s.mailer.ClickLink("https://"+host, token, target) + raw[len(target):]
	})
}

// trackingHost is the go. host of the mailbox's domain, or "".
func (s *Service) trackingHost(ctx context.Context, mb db.Mailbox) string {
	d, err := s.db.GetOutreachDomain(ctx, db.GetOutreachDomainParams{ID: mb.DomainID, OrgID: mb.OrgID})
	if err != nil {
		return ""
	}
	return d.TrackingHost
}
