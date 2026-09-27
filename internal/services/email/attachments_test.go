package email

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestWithAttachments(t *testing.T) {
	o := applyOpts([]SendOption{WithAttachments([]Attachment{{Filename: "a.xlsx"}})})
	if len(o.attachments) != 1 || o.attachments[0].Filename != "a.xlsx" {
		t.Fatalf("WithAttachments did not apply: %+v", o)
	}
	if len(applyOpts(nil).attachments) != 0 {
		t.Fatal("no options should yield no attachments")
	}
}

func TestBuildRawMessage(t *testing.T) {
	content := []byte("hello-spreadsheet-bytes")
	raw, err := buildRawMessage(
		"Cascade Ops <ops@chc.com>", "mgr@chc.com", "Report",
		"<p>hi</p>",
		sendOpts{attachments: []Attachment{{Filename: "report.xlsx", ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", Content: content}}},
	)
	if err != nil {
		t.Fatal(err)
	}
	msg := string(raw)
	for _, want := range []string{
		"From: Cascade Ops <ops@chc.com>",
		"To: mgr@chc.com",
		"Subject: Report",
		"Content-Type: multipart/mixed; boundary=",
		"Content-Type: text/html; charset=UTF-8",
		`filename="report.xlsx"`,
		"Content-Transfer-Encoding: base64",
		base64.StdEncoding.EncodeToString(content),
		"<p>hi</p>",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("raw message missing %q", want)
		}
	}
}

// A reply's sender overrides: Reply-To, a plain-text alternative, and the
// threading headers, with anything else (or a header injection) dropped.
func TestBuildRawMessageReplyHeaders(t *testing.T) {
	o := applyOpts([]SendOption{
		WithReplyTo("desk@nebo.bot"),
		WithTextBody("Yes, we're open."),
		WithHeaders(map[string]string{
			"in-reply-to": "<q1@example.com>",
			"References":  "<root@example.com> <q1@example.com>",
			"Message-ID":  "<ours@nebo.bot>",
			"Bcc":         "someone@example.com",
			"X-Evil":      "a\r\nBcc: x@example.com",
		}),
	})
	if !o.needsRaw() {
		t.Fatal("reply options should need a raw message")
	}
	raw, err := buildRawMessage(formatFrom("Receptionist at Nanna · Alma's Nebo", "desk@nebo.bot"), "c@example.com", "Re: Hours", "<p>Yes, we're open.</p>", o)
	if err != nil {
		t.Fatal(err)
	}
	msg := string(raw)
	for _, want := range []string{
		"From: =?utf-8?",
		"<desk@nebo.bot>",
		"Reply-To: desk@nebo.bot",
		"In-Reply-To: <q1@example.com>",
		"References: <root@example.com> <q1@example.com>",
		"Message-ID: <ours@nebo.bot>",
		"multipart/alternative",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Type: text/html; charset=UTF-8",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("raw message missing %q", want)
		}
	}
	for _, not := range []string{"Bcc", "X-Evil"} {
		if strings.Contains(msg, not) {
			t.Errorf("raw message carries %q", not)
		}
	}
}

// A line break in an address header would inject headers; SES SendRawEmail
// then delivers to whatever Bcc was smuggled in. Such a message is refused.
func TestBuildRawMessageRejectsHeaderInjection(t *testing.T) {
	for _, tc := range []struct{ from, to, replyTo string }{
		{"ops@chc.com", "mgr@chc.com\r\nBcc: x@example.com", ""},
		{"ops@chc.com", "mgr@chc.com", "desk@chc.com\nBcc: x@example.com"},
		{"ops@chc.com\r\nBcc: x@example.com", "mgr@chc.com", ""},
	} {
		o := applyOpts([]SendOption{WithTextBody("hi")})
		o.replyTo = tc.replyTo
		if _, err := buildRawMessage(tc.from, tc.to, "Hi", "<p>hi</p>", o); err == nil {
			t.Errorf("expected an error for from=%q to=%q reply-to=%q", tc.from, tc.to, tc.replyTo)
		}
	}
}
