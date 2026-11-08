package email

import (
	"fmt"
	"strings"
)

// UnsubscribeURL returns the public unsubscribe page for a send's tracking
// token. The page shows a confirmation form on GET and also accepts the RFC 8058
// one-click POST that mailbox providers send from their own unsubscribe button.
func (s *Service) UnsubscribeURL(trackingToken string) string {
	return unsubscribeURL(s.baseURL, trackingToken)
}

func unsubscribeURL(baseURL, trackingToken string) string {
	return fmt.Sprintf("%s/u/%s", strings.TrimRight(baseURL, "/"), trackingToken)
}

// CampaignEmail is one campaign send to one recipient.
type CampaignEmail struct {
	OrgID         string
	FromName      string
	FromEmail     string
	ReplyTo       string
	To            string
	Name          string
	Subject       string
	HTML          string
	TrackingToken string
	TrackOpens    bool
	TrackClicks   bool
}

// CampaignMessage builds the Message for a campaign send, the same way for the
// first attempt and every retry. It personalizes the subject and body, fills
// {{unsubscribe_url}}, adds a footer unsubscribe link when the body has none,
// adds open and click tracking when enabled (never on the unsubscribe link),
// and sets the one-click List-Unsubscribe headers.
func (s *Service) CampaignMessage(c CampaignEmail) Message {
	subject := strings.ReplaceAll(c.Subject, "{{.Name}}", c.Name)
	htmlBody := strings.ReplaceAll(c.HTML, "{{.Name}}", c.Name)
	htmlBody = strings.ReplaceAll(htmlBody, "{{.Email}}", c.To)

	m := Message{
		OrgID:     c.OrgID,
		FromName:  c.FromName,
		FromEmail: c.FromEmail,
		ReplyTo:   c.ReplyTo,
		To:        c.To,
		Subject:   subject,
	}
	if c.TrackingToken == "" {
		m.HTML = htmlBody
		return m
	}

	link := s.UnsubscribeURL(c.TrackingToken)
	htmlBody = strings.ReplaceAll(htmlBody, "{{unsubscribe_url}}", link)
	if !strings.Contains(htmlBody, link) {
		footer := fmt.Sprintf(`<p style="text-align:center;font-size:12px;color:#6b7280;margin-top:30px;"><a href="%s" style="color:#6b7280;">Unsubscribe</a></p>`, link)
		htmlBody = insertBeforeBodyEnd(htmlBody, footer)
	}
	if c.TrackOpens {
		pixel := fmt.Sprintf(`<img src="%s" width="1" height="1" alt="" style="display:none" />`, s.GetTrackingPixelURL(c.TrackingToken))
		htmlBody = insertBeforeBodyEnd(htmlBody, pixel)
	}
	if c.TrackClicks {
		htmlBody = s.RewriteLinksForTracking(htmlBody, c.TrackingToken)
	}

	m.HTML = htmlBody
	m.ListUnsubscribe = link
	return m
}

func insertBeforeBodyEnd(htmlBody, fragment string) string {
	if i := strings.LastIndex(strings.ToLower(htmlBody), "</body>"); i >= 0 {
		return htmlBody[:i] + fragment + htmlBody[i:]
	}
	return htmlBody + fragment
}
