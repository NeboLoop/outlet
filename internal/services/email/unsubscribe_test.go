package email

import (
	"strings"
	"testing"
)

func TestListUnsubscribeHeaders(t *testing.T) {
	url := "https://mail.example.com/u/tok123"
	raw, err := rawMessage{
		From:    "Shop <news@example.com>",
		To:      "reader@example.org",
		ReplyTo: "help@example.com",
		Subject: "Hello",
		HTML:    "<p>hi</p>",
		opts:    applyOpts([]SendOption{WithListUnsubscribe(url)}),
	}.build()
	if err != nil {
		t.Fatal(err)
	}
	msg := string(raw)
	headers, body, ok := strings.Cut(msg, "\r\n\r\n")
	if !ok {
		t.Fatal("message has no header/body separator")
	}
	for _, want := range []string{
		"List-Unsubscribe: <" + url + ">\r\n",
		"List-Unsubscribe-Post: List-Unsubscribe=One-Click\r\n",
		"Reply-To: help@example.com\r\n",
		"Content-Type: text/html; charset=UTF-8\r\n",
	} {
		if !strings.Contains(headers+"\r\n", want) {
			t.Errorf("headers missing %q", want)
		}
	}
	if body != "<p>hi</p>" {
		t.Errorf("body = %q", body)
	}
}

func TestNoListUnsubscribeByDefault(t *testing.T) {
	raw, err := rawMessage{From: "a <a@x.com>", To: "b@y.com", Subject: "s", HTML: "h"}.build()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "List-Unsubscribe") {
		t.Error("transactional message must not carry List-Unsubscribe")
	}
}

func TestNeedsRawMessage(t *testing.T) {
	if applyOpts(nil).needsRawMessage() {
		t.Error("plain send should use body-only SES call")
	}
	if !applyOpts([]SendOption{WithListUnsubscribe("https://x/u/t")}).needsRawMessage() {
		t.Error("List-Unsubscribe needs a raw message")
	}
}

func TestPrepareCampaignEmail(t *testing.T) {
	s := &Service{baseURL: "https://mail.example.com/"}
	want := "https://mail.example.com/u/tok"

	t.Run("adds footer link and headers", func(t *testing.T) {
		subject, body, opts := s.PrepareCampaignEmail("Hi {{.Name}}", "<html><body><p>Hello {{.Name}} ({{.Email}})</p></body></html>", "Ana", "ana@x.com", "tok")
		if subject != "Hi Ana" {
			t.Errorf("subject = %q", subject)
		}
		if !strings.Contains(body, "Hello Ana (ana@x.com)") {
			t.Errorf("body not personalized: %q", body)
		}
		if !strings.Contains(body, `href="`+want+`"`) || !strings.HasSuffix(body, "</body></html>") {
			t.Errorf("footer link missing or misplaced: %q", body)
		}
		if applyOpts(opts).listUnsubscribe != want {
			t.Errorf("List-Unsubscribe = %q", applyOpts(opts).listUnsubscribe)
		}
	})

	t.Run("fills merge tag without a second link", func(t *testing.T) {
		_, body, _ := s.PrepareCampaignEmail("s", `<p><a href="{{unsubscribe_url}}">Leave</a></p>`, "", "", "tok")
		if strings.Count(body, want) != 1 {
			t.Errorf("want exactly one unsubscribe link: %q", body)
		}
	})

	t.Run("no token means no link or headers", func(t *testing.T) {
		_, body, opts := s.PrepareCampaignEmail("s", "<p>x</p>", "", "", "")
		if body != "<p>x</p>" || opts != nil {
			t.Errorf("body = %q, opts = %v", body, opts)
		}
	})
}

func TestTrackingKeepsUnsubscribeLinkDirect(t *testing.T) {
	s := &Service{baseURL: "https://mail.example.com"}
	_, body, _ := s.PrepareCampaignEmail("s", `<p><a href="https://shop.example.com/sale">Sale</a></p>`, "", "", "tok")
	out := s.RewriteLinksForTracking(body, "tok")
	if !strings.Contains(out, `href="https://mail.example.com/u/tok"`) {
		t.Errorf("unsubscribe link was rewritten: %q", out)
	}
	if !strings.Contains(out, "/api/e/c/tok?url=") {
		t.Errorf("other links should still be tracked: %q", out)
	}
}
