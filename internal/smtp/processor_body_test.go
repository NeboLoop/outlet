package smtp

import (
	"net/mail"
	"strings"
	"testing"
)

// Drupal mimemail shape: multipart/mixed → multipart/alternative with
// quoted-printable text and base64 HTML.
func TestExtractBodyNestedEncoded(t *testing.T) {
	raw := "Content-Type: multipart/mixed; boundary=\"outer\"\r\n\r\n" +
		"--outer\r\nContent-Type: multipart/alternative; boundary=\"inner\"\r\n\r\n" +
		"--inner\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\nHello =3D world\r\n" +
		"--inner\r\nContent-Type: text/html; charset=utf-8\r\nContent-Transfer-Encoding: base64\r\n\r\nPGgxPkhlbGxvPC9oMT4=\r\n" +
		"--inner--\r\n--outer--\r\n"
	msg, err := mail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	p := &EmailProcessor{}
	html, text, err := p.extractBody(msg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(html) != "<h1>Hello</h1>" {
		t.Fatalf("html = %q", html)
	}
	if strings.TrimSpace(text) != "Hello = world" {
		t.Fatalf("text = %q", text)
	}
}

func TestExtractBodyFlatHTML(t *testing.T) {
	msg, _ := mail.ReadMessage(strings.NewReader("Content-Type: text/html\r\n\r\n<b>hi</b>"))
	p := &EmailProcessor{}
	html, _, _ := p.extractBody(msg)
	if html != "<b>hi</b>" {
		t.Fatalf("html = %q", html)
	}
}
