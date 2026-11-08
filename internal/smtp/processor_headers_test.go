package smtp

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"net"
	"net/mail"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/services/tracking"
	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/emersion/go-sasl"
	gosmtp "github.com/emersion/go-smtp"
	_ "modernc.org/sqlite"
)

// relay is the platform's SMTP server in these tests: it keeps each message
// Outlet delivers to it.
type relay struct {
	mu   sync.Mutex
	msgs [][]byte
}

func (r *relay) NewSession(*gosmtp.Conn) (gosmtp.Session, error) { return &relaySession{r: r}, nil }

type relaySession struct{ r *relay }

func (s *relaySession) AuthMechanisms() []string { return []string{sasl.Plain} }
func (s *relaySession) Auth(string) (sasl.Server, error) {
	return sasl.NewPlainServer(func(string, string, string) error { return nil }), nil
}
func (s *relaySession) Mail(string, *gosmtp.MailOptions) error { return nil }
func (s *relaySession) Rcpt(string, *gosmtp.RcptOptions) error { return nil }
func (s *relaySession) Data(r io.Reader) error {
	b, err := io.ReadAll(r)
	s.r.mu.Lock()
	s.r.msgs = append(s.r.msgs, b)
	s.r.mu.Unlock()
	return err
}
func (s *relaySession) Reset()        {}
func (s *relaySession) Logout() error { return nil }

func (r *relay) take(t *testing.T) []*mail.Message {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*mail.Message
	for _, raw := range r.msgs {
		m, err := mail.ReadMessage(bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, m)
	}
	r.msgs = nil
	return out
}

func body(t *testing.T, m *mail.Message) string {
	t.Helper()
	h, text := walkPart(m.Header.Get("Content-Type"), m.Header.Get("Content-Transfer-Encoding"), m.Body, 0)
	return h + text
}

// newIngress makes a brand that sends through the platform SMTP relay, with a
// transactional template and a list with one subscriber, and returns a
// processor factory for mail that brand receives over SMTP.
func newIngress(t *testing.T) (func(to ...string) *EmailProcessor, *relay, *sql.DB) {
	t.Helper()
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := migrations.Run(conn, nil); err != nil {
		t.Fatal(err)
	}

	r := &relay{}
	srv := gosmtp.NewServer(r)
	srv.AllowInsecureAuth = true
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })

	for _, q := range []string{
		`INSERT INTO organizations (id, name, slug, api_key, from_email) VALUES ('org-a', 'A', 'a', 'key-a', 'hello@a.example')`,
		`INSERT INTO domain_identities (id, org_id, domain, pathway_id) VALUES ('d-a', 'org-a', 'a.example', (SELECT id FROM sending_pathways WHERE org_id = 'org-a'))`,
		`INSERT INTO transactional_emails (id, org_id, name, slug, subject, html_body, is_active) VALUES ('tpl', 'org-a', 'Welcome', 'welcome', 'Welcome aboard', '<p>From the template: <a href="https://a.example/start">start</a></p>', 1)`,
		`INSERT INTO email_lists (id, public_id, org_id, name, slug) VALUES (1, 'news', 'org-a', 'News', 'news')`,
		`INSERT INTO contacts (id, org_id, name, email) VALUES ('ann', 'org-a', 'Ann', 'ann@x.example'), ('bob', 'org-a', 'Bob', 'bob@x.example')`,
		`INSERT INTO list_subscribers (id, list_id, contact_id, status) VALUES ('s1', 1, 'ann', 'active'), ('s2', 1, 'bob', 'unsubscribed')`,
	} {
		if _, err := conn.Exec(q); err != nil {
			t.Fatal(q, err)
		}
	}
	for key, value := range map[string]string{
		"smtp_host": "127.0.0.1", "smtp_port": strconv.Itoa(ln.Addr().(*net.TCPAddr).Port),
		"smtp_user": "outlet", "smtp_password": "secret",
	} {
		if _, err := conn.Exec(`INSERT INTO platform_settings (key, value_text, category) VALUES (?, ?, 'email')`, key, value); err != nil {
			t.Fatal(err)
		}
	}

	store := db.NewStore(conn)
	es := email.NewService(store, nil)
	es.SetBaseURL("https://outlet.test")
	es.SetLinkSecret("link-secret")
	s := &svc.ServiceContext{DB: store, EmailService: es}
	org, err := store.GetOrganizationByID(context.Background(), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	return func(to ...string) *EmailProcessor { return NewEmailProcessor(s, org, "app@a.example", to) }, r, conn
}

const plainMessage = "From: app@a.example\r\nTo: x\r\nSubject: Hello\r\nContent-Type: text/html\r\n"

// Every header the SMTP settings page documents changes the send.
func TestProcess_OutletHeaders(t *testing.T) {
	newProcessor, r, conn := newIngress(t)

	// X-Outlet-Template: the template's subject and body replace the message's.
	// X-Outlet-Track defaults to opens and clicks.
	if _, err := newProcessor("carl@x.example").Process(strings.NewReader(plainMessage +
		"X-Outlet-Template: welcome\r\n\r\n<p>ignored</p>")); err != nil {
		t.Fatal(err)
	}
	got := r.take(t)
	if len(got) != 1 {
		t.Fatalf("%d messages delivered", len(got))
	}
	b := body(t, got[0])
	if got[0].Header.Get("Subject") != "Welcome aboard" || !strings.Contains(b, "From the template") {
		t.Errorf("template not used: subject %q, body %q", got[0].Header.Get("Subject"), b)
	}
	if !strings.Contains(b, "/api/e/o/") || !strings.Contains(b, "/api/e/c/") {
		t.Errorf("default tracking missing: %q", b)
	}
	if got[0].Header.Get("List-Unsubscribe") != "" {
		t.Error("transactional mail got an unsubscribe header")
	}

	// The pixel and the links record the open and the click on the send.
	var token string
	if err := conn.QueryRow(`SELECT tracking_token FROM transactional_sends WHERE to_email = 'carl@x.example'`).Scan(&token); err != nil {
		t.Fatal(err)
	}
	tr := tracking.New(db.NewStore(conn), nil)
	if err := tr.RecordOpen(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	if err := tr.RecordClick(context.Background(), token, "https://a.example/start"); err != nil {
		t.Fatal(err)
	}
	var opened, clicked sql.NullString
	if err := conn.QueryRow(`SELECT opened_at, clicked_at FROM transactional_sends WHERE tracking_token = ?`, token).Scan(&opened, &clicked); err != nil || !opened.Valid || !clicked.Valid {
		t.Errorf("open and click not recorded: opened %v, clicked %v, %v", opened, clicked, err)
	}

	// X-Outlet-Track: none sends the links as they are, without a pixel.
	if _, err := newProcessor("carl@x.example").Process(strings.NewReader(plainMessage +
		"X-Outlet-Track: none\r\n\r\n<p><a href=\"https://a.example/x\">x</a></p>")); err != nil {
		t.Fatal(err)
	}
	b = body(t, r.take(t)[0])
	if strings.Contains(b, "/api/e/o/") || strings.Contains(b, "/api/e/c/") {
		t.Errorf("X-Outlet-Track: none still tracked: %q", b)
	}

	// X-Outlet-Track: clicks tracks clicks only.
	if _, err := newProcessor("carl@x.example").Process(strings.NewReader(plainMessage +
		"X-Outlet-Track: clicks\r\n\r\n<p><a href=\"https://a.example/x\">x</a></p>")); err != nil {
		t.Fatal(err)
	}
	b = body(t, r.take(t)[0])
	if strings.Contains(b, "/api/e/o/") || !strings.Contains(b, "/api/e/c/") {
		t.Errorf("X-Outlet-Track: clicks: %q", b)
	}

	// X-Outlet-Type: marketing with X-Outlet-List goes to the list's active
	// subscriber with an unsubscribe link, and is refused for one who left.
	marketing := plainMessage + "X-Outlet-Type: marketing\r\nX-Outlet-List: news\r\n\r\n<p>News</p>"
	if _, err := newProcessor("ann@x.example").Process(strings.NewReader(marketing)); err != nil {
		t.Fatal(err)
	}
	got = r.take(t)
	if len(got) != 1 || !strings.HasPrefix(got[0].Header.Get("List-Unsubscribe"), "<https://outlet.test/u/") {
		t.Fatalf("marketing to a subscriber: %d messages, List-Unsubscribe %q", len(got), got[0].Header.Get("List-Unsubscribe"))
	}
	if b := body(t, got[0]); !strings.Contains(b, "https://outlet.test/u/") {
		t.Errorf("no unsubscribe link in %q", b)
	}
	var contactID sql.NullString
	if err := conn.QueryRow(`SELECT contact_id FROM transactional_sends WHERE to_email = 'ann@x.example'`).Scan(&contactID); err != nil || contactID.String != "ann" {
		t.Errorf("the marketing send isn't linked to its contact: %v %v", contactID, err)
	}

	for _, refused := range []struct{ to, msg string }{
		{"bob@x.example", marketing}, // unsubscribed from the list
		{"dan@x.example", marketing}, // never subscribed
		{"ann@x.example", plainMessage + "X-Outlet-Type: marketing\r\n\r\n<p>News</p>"}, // no list
		{"ann@x.example", plainMessage + "X-Outlet-List: news\r\n\r\n<p>News</p>"},      // a list on transactional mail
	} {
		_, err := newProcessor(refused.to).Process(strings.NewReader(refused.msg))
		var smtpErr *gosmtp.SMTPError
		if !errors.As(err, &smtpErr) || smtpErr.Code != 554 {
			t.Errorf("%s: want a permanent refusal, got %v", refused.to, err)
		}
	}
	if got := r.take(t); len(got) != 0 {
		t.Errorf("%d refused messages were delivered", len(got))
	}

	// X-Outlet-Tags and X-Outlet-Meta-* are recorded with the send.
	if _, err := newProcessor("erin@x.example").Process(strings.NewReader(plainMessage +
		"X-Outlet-Tags: receipt, order\r\nX-Outlet-Meta-Order-Id: 42\r\n\r\n<p>x</p>")); err != nil {
		t.Fatal(err)
	}
	var contextData string
	if err := conn.QueryRow(`SELECT context_data FROM transactional_sends WHERE to_email = 'erin@x.example'`).Scan(&contextData); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(contextData, `"Order-Id":"42"`) || !strings.Contains(contextData, `"receipt","order"`) {
		t.Errorf("context_data = %s", contextData)
	}
}
