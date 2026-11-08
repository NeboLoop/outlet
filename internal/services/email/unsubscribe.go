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
	OrgID     string
	FromName  string
	FromEmail string
	ReplyTo   string
	To        string
	Name      string
	Subject   string
	HTML      string
	Text      string
	// The campaign's design (email_designs), if it has one: the HTML and
	// plain text its content is placed into at {{content}}.
	DesignHTML    string
	DesignText    string
	TrackingToken string
	TrackOpens    bool
	TrackClicks   bool
}

// CampaignMessage builds the Message for a campaign send, the same way for the
// first attempt, every retry and a test send. It places the content in the
// campaign's design, fills merge tags, adds a footer
// unsubscribe link when the body has none, adds open and click tracking when
// enabled (never on the unsubscribe link), and sets the one-click
// List-Unsubscribe headers.
func (s *Service) CampaignMessage(c CampaignEmail) Message {
	tags := mergeContext{BaseURL: s.baseURL, Name: c.Name, Email: c.To, TrackingToken: c.TrackingToken}
	m := Message{
		OrgID:     c.OrgID,
		Kind:      KindMarketing,
		FromName:  c.FromName,
		FromEmail: c.FromEmail,
		ReplyTo:   c.ReplyTo,
		To:        c.To,
		Subject:   fillMergeTags(c.Subject, tags, false),
	}
	if c.Text != "" {
		m.Text = fillMergeTags(ApplyDesign(c.DesignText, c.Text), tags, false)
	}
	htmlBody := fillMergeTags(ApplyDesign(c.DesignHTML, c.HTML), tags, true)
	if c.TrackingToken == "" {
		m.HTML = htmlBody
		return m
	}

	link := s.UnsubscribeURL(c.TrackingToken)
	if !strings.Contains(htmlBody, link) {
		footer := fmt.Sprintf(`<p style="text-align:center;font-size:12px;color:#6b7280;margin-top:30px;"><a href="%s" style="color:#6b7280;">Unsubscribe</a></p>`, link)
		htmlBody = insertBeforeBodyEnd(htmlBody, footer)
	}
	if m.Text != "" && !strings.Contains(m.Text, link) {
		m.Text += "\n\nUnsubscribe: " + link
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
