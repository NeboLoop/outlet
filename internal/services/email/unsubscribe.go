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

// PrepareCampaignEmail personalizes a campaign's subject and body for one
// recipient and makes sure the body carries a working unsubscribe link: the
// {{unsubscribe_url}} merge tag is filled in, and when the body has no link to
// the unsubscribe page a footer link is added. It returns the subject, the body,
// and the send options that add the one-click List-Unsubscribe headers.
func (s *Service) PrepareCampaignEmail(subject, htmlBody, name, toEmail, trackingToken string) (string, string, []SendOption) {
	subject = strings.ReplaceAll(subject, "{{.Name}}", name)
	htmlBody = strings.ReplaceAll(htmlBody, "{{.Name}}", name)
	htmlBody = strings.ReplaceAll(htmlBody, "{{.Email}}", toEmail)

	if trackingToken == "" {
		return subject, htmlBody, nil
	}

	unsubscribeURL := s.UnsubscribeURL(trackingToken)
	htmlBody = strings.ReplaceAll(htmlBody, "{{unsubscribe_url}}", unsubscribeURL)
	if !strings.Contains(htmlBody, unsubscribeURL) {
		footer := fmt.Sprintf(`<p style="text-align:center;font-size:12px;color:#6b7280;margin-top:30px;"><a href="%s" style="color:#6b7280;">Unsubscribe</a></p>`, unsubscribeURL)
		if i := strings.LastIndex(strings.ToLower(htmlBody), "</body>"); i >= 0 {
			htmlBody = htmlBody[:i] + footer + htmlBody[i:]
		} else {
			htmlBody += footer
		}
	}

	return subject, htmlBody, []SendOption{WithListUnsubscribe(unsubscribeURL)}
}
