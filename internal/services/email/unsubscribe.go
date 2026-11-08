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
// campaign's design, fills merge tags, and adds the unsubscribe link, the
// List-Unsubscribe headers and open and click tracking through Track.
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
	m.HTML = fillMergeTags(ApplyDesign(c.DesignHTML, c.HTML), tags, true)
	s.Track(&m, Tracking{Token: c.TrackingToken, Unsubscribe: true, Opens: c.TrackOpens, Clicks: c.TrackClicks})
	return m
}

// Tracking is what Track adds to one send's email.
type Tracking struct {
	Token string // the send's tracking token; without one nothing is added
	// Unsubscribe adds a footer unsubscribe link when the body has none, and
	// the one-click List-Unsubscribe headers: marketing mail.
	Unsubscribe bool
	Opens       bool // an open pixel
	Clicks      bool // signed click-tracking links (never on the unsubscribe link)
}

// Track adds a send's unsubscribe link, open pixel and click tracking to its
// built message. It is the one way: campaign sends through CampaignMessage,
// and transactional and marketing sends from the API and SMTP.
func (s *Service) Track(m *Message, t Tracking) {
	if t.Token == "" {
		return
	}
	htmlBody := m.HTML
	if t.Unsubscribe {
		link := s.UnsubscribeURL(t.Token)
		if !strings.Contains(htmlBody, link) {
			footer := fmt.Sprintf(`<p style="text-align:center;font-size:12px;color:#6b7280;margin-top:30px;"><a href="%s" style="color:#6b7280;">Unsubscribe</a></p>`, link)
			htmlBody = insertBeforeBodyEnd(htmlBody, footer)
		}
		if m.Text != "" && !strings.Contains(m.Text, link) {
			m.Text += "\n\nUnsubscribe: " + link
		}
		m.ListUnsubscribe = link
	}
	if t.Opens {
		pixel := fmt.Sprintf(`<img src="%s" width="1" height="1" alt="" style="display:none" />`, s.GetTrackingPixelURL(t.Token))
		htmlBody = insertBeforeBodyEnd(htmlBody, pixel)
	}
	if t.Clicks {
		htmlBody = s.RewriteLinksForTracking(htmlBody, t.Token)
	}
	m.HTML = htmlBody
}

func insertBeforeBodyEnd(htmlBody, fragment string) string {
	if i := strings.LastIndex(strings.ToLower(htmlBody), "</body>"); i >= 0 {
		return htmlBody[:i] + fragment + htmlBody[i:]
	}
	return htmlBody + fragment
}
