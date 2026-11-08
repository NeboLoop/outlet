package organizations

import (
	"bytes"
	"io"
	"net"
	"net/mail"
	"strconv"
	"sync"
	"testing"

	"github.com/emersion/go-sasl"
	gosmtp "github.com/emersion/go-smtp"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mailSink is an SMTP server that keeps the last message it receives.
type mailSink struct {
	mu   sync.Mutex
	data []byte
}

func (k *mailSink) NewSession(_ *gosmtp.Conn) (gosmtp.Session, error) { return &mailSinkSession{k}, nil }

type mailSinkSession struct{ k *mailSink }

func (s *mailSinkSession) AuthMechanisms() []string { return []string{sasl.Plain} }
func (s *mailSinkSession) Auth(string) (sasl.Server, error) {
	return sasl.NewPlainServer(func(_, _, _ string) error { return nil }), nil
}
func (s *mailSinkSession) Mail(string, *gosmtp.MailOptions) error { return nil }
func (s *mailSinkSession) Rcpt(string, *gosmtp.RcptOptions) error { return nil }
func (s *mailSinkSession) Data(r io.Reader) error {
	b, err := io.ReadAll(r)
	s.k.mu.Lock()
	defer s.k.mu.Unlock()
	s.k.data = b
	return err
}
func (s *mailSinkSession) Reset()        {}
func (s *mailSinkSession) Logout() error { return nil }

// An invitation is platform mail: it goes out from the platform's sender set
// on the Email settings screen.
func TestCreateInvitation_SendsFromThePlatformSender(t *testing.T) {
	f := newRESTFixture(t)

	k := &mailSink{}
	srv := gosmtp.NewServer(k)
	srv.AllowInsecureAuth = true
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })

	for key, value := range map[string]string{
		"smtp_host": "127.0.0.1", "smtp_port": strconv.Itoa(ln.Addr().(*net.TCPAddr).Port), "smtp_user": "outlet", "smtp_password": "secret",
		"from_email": "team@outlet.example", "from_name": "Outlet",
	} {
		_, err := f.svc.DB.GetDB().Exec(`INSERT INTO platform_settings (key, value_text, category) VALUES (?, ?, 'email')`, key, value)
		require.NoError(t, err)
	}

	resp, err := NewCreateInvitationLogic(f.as(t, "anne"), f.svc).CreateInvitation(&types.CreateInvitationRequest{Id: "alpha-id", Email: "new@example.com"})
	require.NoError(t, err)
	assert.True(t, resp.EmailSent)

	k.mu.Lock()
	defer k.mu.Unlock()
	msg, err := mail.ReadMessage(bytes.NewReader(k.data))
	require.NoError(t, err)
	assert.Equal(t, `"Outlet" <team@outlet.example>`, msg.Header.Get("From"))
	assert.Equal(t, "<new@example.com>", msg.Header.Get("To"))
}
