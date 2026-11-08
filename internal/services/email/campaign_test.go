package email

import (
	"strings"
	"testing"
)

func TestCampaignMessage(t *testing.T) {
	s := &Service{baseURL: "https://mail.example.com/"}
	link := "https://mail.example.com/u/tok"

	t.Run("personalizes and adds footer link and headers", func(t *testing.T) {
		m := s.CampaignMessage(CampaignEmail{OrgID: "o1", To: "ana@x.com", Name: "Ana", Subject: "Hi {{.Name}}",
			HTML: "<html><body><p>Hello {{.Name}} ({{.Email}})</p></body></html>", TrackingToken: "tok"})
		if m.Subject != "Hi Ana" || !strings.Contains(m.HTML, "Hello Ana (ana@x.com)") {
			t.Errorf("not personalized: %q %q", m.Subject, m.HTML)
		}
		if !strings.Contains(m.HTML, `href="`+link+`"`) || !strings.HasSuffix(m.HTML, "</body></html>") {
			t.Errorf("footer link missing or misplaced: %q", m.HTML)
		}
		if m.ListUnsubscribe != link || m.OrgID != "o1" {
			t.Errorf("message = %+v", m)
		}
	})

	t.Run("fills merge tag without a second link", func(t *testing.T) {
		m := s.CampaignMessage(CampaignEmail{To: "a@x.com", HTML: `<a href="{{unsubscribe_url}}">Leave</a>`, TrackingToken: "tok"})
		if strings.Count(m.HTML, link) != 1 {
			t.Errorf("want exactly one unsubscribe link: %q", m.HTML)
		}
	})

	t.Run("tracking keeps the unsubscribe link direct", func(t *testing.T) {
		m := s.CampaignMessage(CampaignEmail{To: "a@x.com", HTML: `<body><a href="https://shop.example/sale">Sale</a></body>`,
			TrackingToken: "tok", TrackOpens: true, TrackClicks: true})
		if !strings.Contains(m.HTML, `href="`+link+`"`) {
			t.Errorf("unsubscribe link was rewritten: %q", m.HTML)
		}
		if !strings.Contains(m.HTML, "/api/e/c/tok?url=") || !strings.Contains(m.HTML, "/api/e/o/tok") {
			t.Errorf("click or open tracking missing: %q", m.HTML)
		}
	})

	t.Run("no token means no link or headers", func(t *testing.T) {
		m := s.CampaignMessage(CampaignEmail{To: "a@x.com", HTML: "<p>x</p>"})
		if m.HTML != "<p>x</p>" || m.ListUnsubscribe != "" {
			t.Errorf("message = %+v", m)
		}
	})
}
