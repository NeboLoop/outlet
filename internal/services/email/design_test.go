package email

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/outlet-sh/outlet/internal/db"
)

const testDesign = `<html><body><div class="brand">ACME</div>{{ content }}<a href="{{unsubscribe_url}}">Leave</a></body></html>`

func TestApplyDesign(t *testing.T) {
	if got := ApplyDesign("", "<p>Hi</p>"); got != "<p>Hi</p>" {
		t.Errorf("no design: %q", got)
	}
	if got := ApplyDesign(testDesign, "<p>$1 off</p>"); !strings.Contains(got, `<div class="brand">ACME</div><p>$1 off</p><a`) {
		t.Errorf("content not placed at {{content}}: %q", got)
	}
	if got := ApplyDesign("<html><body><header/></body></html>", "<p>Hi</p>"); got != "<html><body><header/><p>Hi</p></body></html>" {
		t.Errorf("design without the tag: %q", got)
	}
}

func TestCampaignMessageAppliesTheDesign(t *testing.T) {
	s := &Service{baseURL: "https://mail.example.com"}
	link := "https://mail.example.com/u/tok"
	m := s.CampaignMessage(CampaignEmail{
		To: "ana@x.com", Name: "Ana", Subject: "Hi", TrackingToken: "tok",
		HTML: "<p>Hello {{name}}</p>", Text: "Hello {{name}}",
		DesignHTML: testDesign, DesignText: "ACME\n\n{{content}}\n\nLeave: {{unsubscribe_url}}",
	})
	if !strings.Contains(m.HTML, `<div class="brand">ACME</div><p>Hello Ana</p>`) {
		t.Errorf("design not applied: %q", m.HTML)
	}
	if strings.Count(m.HTML, link) != 1 {
		t.Errorf("the design's unsubscribe link is filled, and no second one added: %q", m.HTML)
	}
	if m.Text != "ACME\n\nHello Ana\n\nLeave: "+link {
		t.Errorf("text design: %q", m.Text)
	}
}

// A sequence email takes its template's design when it has one, and
// otherwise the built-in layout its template type names, through the same
// ApplyDesign as campaigns.
func TestRenderQueuedSequenceEmail(t *testing.T) {
	sender, store := newTestService(t)
	seq := NewSequenceServiceWithBaseURL(store, sender, "https://mail.example.com")
	row := db.GetPendingEmailsRow{
		Name: "Ana", Email: "ana@x.com", Subject: "Hi {{name}}", HtmlBody: "<p>Hello {{name}}</p>",
		TrackingToken: sql.NullString{String: "tok", Valid: true},
	}

	row.DesignHtml = testDesign
	subject, html := seq.renderQueued(context.Background(), row)
	if subject != "Hi Ana" || !strings.Contains(html, `<div class="brand">ACME</div><p>Hello Ana</p>`) ||
		!strings.Contains(html, `href="https://mail.example.com/u/tok"`) {
		t.Errorf("design: %q %q", subject, html)
	}

	row.DesignHtml = ""
	row.TemplateType = sql.NullString{String: "branded", Valid: true}
	_, html = seq.renderQueued(context.Background(), row)
	if !strings.Contains(html, `<div class="content">`) || !strings.Contains(html, "<p>Hello Ana</p>") {
		t.Errorf("branded layout: %q", html)
	}

	row.TemplateType = sql.NullString{String: "none", Valid: true}
	if _, html = seq.renderQueued(context.Background(), row); html != "<p>Hello Ana</p>" {
		t.Errorf("no layout: %q", html)
	}
}
