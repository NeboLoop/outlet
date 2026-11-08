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
	raw, err := rawMessage{
		From:    "Cascade Ops <ops@chc.com>",
		To:      "mgr@chc.com",
		Subject: "Report",
		HTML:    "<p>hi</p>",
		opts: applyOpts([]SendOption{WithAttachments([]Attachment{
			{Filename: "report.xlsx", ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", Content: content},
		})}),
	}.build()
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
