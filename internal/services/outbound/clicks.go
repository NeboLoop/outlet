package outbound

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
)

// Click tracking is chosen per campaign and off by default. Links go through
// the sending domain's own go. subdomain, never a shared host, and each link's
// address is stored here, so a token can only lead where the email pointed.

// scannerWindow: mail security scanners open links within seconds of
// delivery; a click this soon after sending is not counted.
const scannerWindow = time.Minute

var linkRe = regexp.MustCompile(`https?://[^\s<>"']+`)

func boolInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// trackingReady refuses click tracking unless every mailbox in the pool is on
// a domain whose go. subdomain points at this server.
func (s *Service) trackingReady(ctx context.Context, orgID string, pool []db.OutboundMailbox) error {
	for _, mb := range pool {
		d, err := s.db.GetOutboundDomain(ctx, db.GetOutboundDomainParams{ID: mb.DomainID, OrgID: orgID})
		if err != nil {
			return err
		}
		if d.TrackingHost == "" {
			return refuse("%s has no click-tracking subdomain. Run setup_sending(action: add_domain, domain: %q, click_tracking: true), or draft without track_clicks.", d.Domain, d.Domain)
		}
		var c DNSCheck
		_ = json.Unmarshal([]byte(d.CheckResult), &c)
		if c.Tracking == nil || !*c.Tracking {
			return refuse("%s doesn't point at this server yet (CNAME go to %s). Add it, then setup_sending(action: check_domain).", d.TrackingHost, s.tracking)
		}
	}
	return nil
}

// trackLinks swaps each link in a body for a tracked one on host. It runs
// when the email is sent, so drafts show the real links and an edit is
// tracked too.
func (s *Service) trackLinks(ctx context.Context, m db.OutboundMessage, host string) (string, error) {
	var err error
	body := linkRe.ReplaceAllStringFunc(m.Body, func(raw string) string {
		url := strings.TrimRight(raw, ".,;:!?)]")
		tail := raw[len(url):]
		if err != nil {
			return raw
		}
		b := make([]byte, 16)
		_, _ = rand.Read(b)
		token := hex.EncodeToString(b)
		if err = s.db.CreateOutboundLink(ctx, db.CreateOutboundLinkParams{Token: token, OrgID: m.OrgID, MessageID: m.ID, Url: url}); err != nil {
			return raw
		}
		return "https://" + host + "/o/c/" + token + tail
	})
	return body, err
}

// trackingHostFor is the host for a message's links, or "" when its campaign
// doesn't track clicks.
func (s *Service) trackingHostFor(ctx context.Context, m db.OutboundMessage, mb db.OutboundMailbox) string {
	if !m.EnrollmentID.Valid || !linkRe.MatchString(m.Body) {
		return ""
	}
	e, err := s.db.GetOutboundEnrollment(ctx, m.EnrollmentID.String)
	if err != nil {
		return ""
	}
	c, err := s.db.GetOutboundCampaign(ctx, db.GetOutboundCampaignParams{ID: e.CampaignID, OrgID: e.OrgID})
	if err != nil || c.TrackClicks == 0 {
		return ""
	}
	d, err := s.db.GetOutboundDomain(ctx, db.GetOutboundDomainParams{ID: mb.DomainID, OrgID: mb.OrgID})
	if err != nil {
		return ""
	}
	return d.TrackingHost
}

// Click records a click on a tracked link and returns where it goes. An
// unknown token returns "".
func (s *Service) Click(ctx context.Context, token string) string {
	l, err := s.db.GetOutboundLinkForClick(ctx, token)
	if err != nil {
		return ""
	}
	now := s.now()
	if !l.SentAt.Valid || now.Sub(parseTS(l.SentAt.String)) >= scannerWindow {
		_ = s.db.RecordOutboundClick(ctx, db.RecordOutboundClickParams{FirstClickedAt: sql.NullString{String: ts(now), Valid: true}, Token: token})
	}
	return l.Url
}
