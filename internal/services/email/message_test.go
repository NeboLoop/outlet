package email

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"testing"
)

func envelopeFor(t *testing.T, m Message) envelope {
	t.Helper()
	from, err := parseSingleAddress("from", "news@shop.example")
	if err != nil {
		t.Fatal(err)
	}
	from.Name = "Acme, Inc"
	to, err := parseSingleAddress("to", m.To)
	if err != nil {
		t.Fatal(err)
	}
	return envelope{from: from, to: to, msg: m}
}

func parse(t *testing.T, raw []byte) *mail.Message {
	t.Helper()
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("message does not parse: %v\n%s", err, raw)
	}
	return msg
}

func TestBuildHeaders(t *testing.T) {
	raw, err := envelopeFor(t, Message{To: "ana@x.example", Subject: "Votre reçu – #42", HTML: "<p>hi</p>"}).build()
	if err != nil {
		t.Fatal(err)
	}
	msg := parse(t, raw)

	from, err := mail.ParseAddress(msg.Header.Get("From"))
	if err != nil || from.Name != "Acme, Inc" || from.Address != "news@shop.example" {
		t.Errorf("From = %q (%v)", msg.Header.Get("From"), err)
	}
	subject, _ := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject"))
	if subject != "Votre reçu – #42" {
		t.Errorf("Subject = %q", subject)
	}
	if msg.Header.Get("Date") == "" || !strings.HasSuffix(msg.Header.Get("Message-ID"), "@shop.example>") {
		t.Errorf("missing Date or Message-ID: %v", msg.Header)
	}
	if msg.Header.Get("List-Unsubscribe") != "" {
		t.Error("transactional mail must not carry List-Unsubscribe")
	}
}

func TestBuildRejectsHeaderInjection(t *testing.T) {
	for name, m := range map[string]Message{
		"subject":     {To: "a@x.example", Subject: "Hi\r\nBcc: victim@y.example", HTML: "x"},
		"unsubscribe": {To: "a@x.example", Subject: "s", HTML: "x", ListUnsubscribe: "https://x/u/t>\r\nBcc: v@y"},
	} {
		if _, err := envelopeFor(t, m).build(); !errors.Is(err, ErrInvalidMessage) {
			t.Errorf("%s: want ErrInvalidMessage, got %v", name, err)
		}
	}
	for _, to := range []string{"a@x.example, b@y.example", "a@x.example\r\nBcc: b@y.example", "not-an-address"} {
		if _, err := parseSingleAddress("to", to); !errors.Is(err, ErrInvalidMessage) {
			t.Errorf("to %q: want ErrInvalidMessage, got %v", to, err)
		}
	}
}

func TestBuildTextAndHTML(t *testing.T) {
	longLine := strings.Repeat("word ", 400) // well past the 998-character line limit
	raw, err := envelopeFor(t, Message{To: "a@x.example", Subject: "s", HTML: "<p>" + longLine + "</p>", Text: "plain " + longLine}).build()
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(raw), "\r\n") {
		if len(line) > 998 {
			t.Fatalf("line of %d characters", len(line))
		}
	}
	msg := parse(t, raw)
	mediaType, params, _ := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if mediaType != "multipart/alternative" {
		t.Fatalf("Content-Type = %s", mediaType)
	}
	parts := readParts(t, msg.Body, params["boundary"])
	if len(parts) != 2 || !strings.HasPrefix(parts[0], "plain word") || !strings.HasPrefix(parts[1], "<p>word") {
		t.Errorf("parts = %q", parts)
	}
}

func TestBuildAttachments(t *testing.T) {
	raw, err := envelopeFor(t, Message{
		To: "a@x.example", Subject: "Report", HTML: "<p>hi</p>", Text: "hi",
		Attachments: []Attachment{{Filename: "report.xlsx", ContentType: "application/vnd.ms-excel", Content: []byte("bytes")}},
	}).build()
	if err != nil {
		t.Fatal(err)
	}
	msg := parse(t, raw)
	mediaType, params, _ := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if mediaType != "multipart/mixed" {
		t.Fatalf("Content-Type = %s", mediaType)
	}
	mr := multipart.NewReader(msg.Body, params["boundary"])
	first, _ := mr.NextPart()
	if inner, _, _ := mime.ParseMediaType(first.Header.Get("Content-Type")); inner != "multipart/alternative" {
		t.Errorf("first part = %s", inner)
	}
	second, _ := mr.NextPart()
	if second.FileName() != "report.xlsx" {
		t.Errorf("attachment name = %q", second.FileName())
	}
	if content, _ := io.ReadAll(base64.NewDecoder(base64.StdEncoding, second)); string(content) != "bytes" {
		t.Errorf("attachment content = %q", content)
	}
}

func TestBuildListUnsubscribe(t *testing.T) {
	raw, err := envelopeFor(t, Message{To: "a@x.example", Subject: "s", HTML: "x", ListUnsubscribe: "https://mail.example/u/tok"}).build()
	if err != nil {
		t.Fatal(err)
	}
	msg := parse(t, raw)
	if msg.Header.Get("List-Unsubscribe") != "<https://mail.example/u/tok>" ||
		msg.Header.Get("List-Unsubscribe-Post") != "List-Unsubscribe=One-Click" {
		t.Errorf("headers = %v", msg.Header)
	}
}

func readParts(t *testing.T, body io.Reader, boundary string) []string {
	t.Helper()
	var out []string
	mr := multipart.NewReader(body, boundary)
	for {
		p, err := mr.NextPart()
		if err != nil {
			return out
		}
		b, _ := io.ReadAll(p) // multipart.Reader decodes quoted-printable
		out = append(out, string(b))
	}
}
