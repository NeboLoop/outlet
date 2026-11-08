package email

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/mail"
	"strconv"
	"sync"
	"testing"

	"github.com/emersion/go-sasl"
	gosmtp "github.com/emersion/go-smtp"
)

// sink is a minimal SMTP server that records what it receives.
type sink struct {
	mu       sync.Mutex
	from     string
	rcpts    []string
	data     []byte
	username string
}

func (k *sink) NewSession(_ *gosmtp.Conn) (gosmtp.Session, error) { return &sinkSession{k: k}, nil }

type sinkSession struct{ k *sink }

func (s *sinkSession) AuthMechanisms() []string { return []string{sasl.Plain} }
func (s *sinkSession) Auth(string) (sasl.Server, error) {
	return sasl.NewPlainServer(func(_, username, password string) error {
		if password != "smtp-secret" {
			return errors.New("bad password")
		}
		s.k.mu.Lock()
		s.k.username = username
		s.k.mu.Unlock()
		return nil
	}), nil
}
func (s *sinkSession) Mail(from string, _ *gosmtp.MailOptions) error {
	s.k.mu.Lock()
	defer s.k.mu.Unlock()
	s.k.from = from
	return nil
}
func (s *sinkSession) Rcpt(to string, _ *gosmtp.RcptOptions) error {
	s.k.mu.Lock()
	defer s.k.mu.Unlock()
	s.k.rcpts = append(s.k.rcpts, to)
	return nil
}
func (s *sinkSession) Data(r io.Reader) error {
	b, err := io.ReadAll(r)
	s.k.mu.Lock()
	defer s.k.mu.Unlock()
	s.k.data = b
	return err
}
func (s *sinkSession) Reset()        {}
func (s *sinkSession) Logout() error { return nil }

func TestSendDeliversOverSMTP(t *testing.T) {
	s, store := newTestService(t)

	k := &sink{}
	srv := gosmtp.NewServer(k)
	srv.AllowInsecureAuth = true
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	port := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)

	for key, value := range map[string]string{
		"smtp_host": "127.0.0.1", "smtp_port": port, "smtp_user": "outlet", "smtp_password": "smtp-secret",
		"from_email": "platform@outlet.example", "reply_to": "platform-replies@outlet.example",
	} {
		if _, err := store.GetDB().Exec(`INSERT INTO platform_settings (key, value_text, category) VALUES (?, ?, 'email')`, key, value); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.GetDB().Exec(`UPDATE organizations SET from_name = 'Brand A', reply_to = 'support@a.example' WHERE id = 'org-a'`); err != nil {
		t.Fatal(err)
	}

	err = s.Send(context.Background(), Message{
		OrgID: "org-a", To: "Customer <customer@x.example>", Subject: "Your receipt",
		HTML: "<p>Thanks</p>", Text: "Thanks",
	})
	if err != nil {
		t.Fatal(err)
	}

	k.mu.Lock()
	defer k.mu.Unlock()
	if k.username != "outlet" || k.from != "hello@a.example" || len(k.rcpts) != 1 || k.rcpts[0] != "customer@x.example" {
		t.Errorf("envelope: user=%q from=%q rcpts=%v", k.username, k.from, k.rcpts)
	}
	msg, err := mail.ReadMessage(bytes.NewReader(k.data))
	if err != nil {
		t.Fatal(err)
	}
	// The brand's From and Reply-To, never the platform's
	if got := msg.Header.Get("From"); got != `"Brand A" <hello@a.example>` {
		t.Errorf("From = %q", got)
	}
	if got := msg.Header.Get("Reply-To"); got != "<support@a.example>" {
		t.Errorf("Reply-To = %q", got)
	}
	if !bytes.Contains(k.data, []byte("multipart/alternative")) {
		t.Error("expected a text and an HTML part")
	}
}
