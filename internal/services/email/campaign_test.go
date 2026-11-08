package email

import (
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func TestCampaignMessage(t *testing.T) {
	s := &Service{baseURL: "https://mail.example.com/"}
	s.SetLinkSecret("test-secret")
	link := "https://mail.example.com/u/tok"

	t.Run("personalizes and adds footer link and headers", func(t *testing.T) {
		m := s.CampaignMessage(CampaignEmail{OrgID: "o1", To: "ana@x.com", Name: "Ana <b>Lee</b>", Subject: "Hi {{first_name,fallback=there}}",
			HTML: "<html><body><p>Hello {{name}} ({{email}})</p></body></html>", TrackingToken: "tok"})
		if m.Subject != "Hi Ana" || !strings.Contains(m.HTML, "Hello Ana &lt;b&gt;Lee&lt;/b&gt; (ana@x.com)") {
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

func TestClickLinksAreSigned(t *testing.T) {
	s := &Service{baseURL: "https://mail.example.com"}
	s.SetLinkSecret("test-secret")
	out := s.RewriteLinksForTracking(`<a href="https://shop.example/a?x=1&amp;y=2">Buy</a>`, "tok")

	m := regexp.MustCompile(`url=([^&]+)&amp;sig=([0-9a-f]+)`).FindStringSubmatch(out)
	if m == nil {
		t.Fatalf("no signed link in %q", out)
	}
	target, _ := url.QueryUnescape(m[1])
	if target != "https://shop.example/a?x=1&y=2" {
		t.Errorf("target = %q", target)
	}
	if !s.VerifyClickLink("tok", target, m[2]) {
		t.Error("own link should verify")
	}
	for name, ok := range map[string]bool{
		"other target": s.VerifyClickLink("tok", "https://evil.example", m[2]),
		"other token":  s.VerifyClickLink("tok2", target, m[2]),
		"no signature": s.VerifyClickLink("tok", target, ""),
	} {
		if ok {
			t.Errorf("%s verified", name)
		}
	}
}

func TestMergeTagFallbacksAndUnknownTags(t *testing.T) {
	c := mergeContext{BaseURL: "https://m.example", Email: "a@x.com", TrackingToken: "tok", CustomFields: map[string]string{"plan": "Pro"}}
	got := fillMergeTags("Hi {{first_name,fallback=there}} on {{plan}}, {{coupon,fallback=none}} {{mystery}} {{web_version_url}}", c, true)
	want := "Hi there on Pro, none {{mystery}} https://m.example/w/tok"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}
