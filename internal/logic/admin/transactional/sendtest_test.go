package transactional

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
	"sync"
	"testing"

	"github.com/emersion/go-sasl"
	gosmtp "github.com/emersion/go-smtp"
	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

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

// A brand member (who never sees the API key) sends a test of a template
// through the SDK route's send, in the template's design.
func TestSendTransactionalTest_AsMember(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))

	k := &mailSink{}
	srv := gosmtp.NewServer(k)
	srv.AllowInsecureAuth = true
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })

	for _, q := range []string{
		`INSERT INTO users (id, email, password_hash, name, role, status, email_verified) VALUES ('mike', 'mike@example.com', 'x', 'M', 'agent', 'active', 1)`,
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('alpha-id', 'alpha', 'alpha', 'k-a'), ('bravo-id', 'bravo', 'bravo', 'k-b')`,
		`INSERT INTO org_users (org_id, user_id, role) VALUES ('alpha-id', 'mike', 'member')`,
		`INSERT INTO email_designs (id, org_id, name, slug, html_body) VALUES (7, 'alpha-id', 'D', 'd', '<html><body><div>ACME brand {{order}}</div>{{content}}</body></html>')`,
		`INSERT INTO transactional_emails (id, org_id, design_id, name, slug, subject, html_body, is_active) VALUES ('te-1', 'alpha-id', 7, 'Receipt', 'receipt', 'Receipt {{order}}', '<p>Order {{order}}</p>', 1)`,
		`INSERT INTO platform_settings (key, value_text, category) VALUES ('smtp_host', '127.0.0.1', 'email'), ('smtp_port', '` + strconv.Itoa(ln.Addr().(*net.TCPAddr).Port) + `', 'email'), ('smtp_user', 'u', 'email'), ('smtp_password', 'p', 'email'), ('from_email', 'team@outlet.example', 'email')`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err, q)
	}
	store := db.NewStore(conn)
	s := &svc.ServiceContext{DB: store, EmailService: email.NewService(store, nil)}
	user, err := access.Active(context.Background(), store, "mike")
	require.NoError(t, err)
	in := func(brand string) context.Context {
		return access.WithBrand(access.WithUser(context.Background(), user), brand)
	}

	resp, err := NewSendTransactionalTestLogic(in("alpha-id"), s).SendTransactionalTest(&types.SendEmailRequest{
		To: "mike@example.com", TemplateSlug: "receipt", Variables: map[string]string{"order": "42"},
	})
	require.NoError(t, err)
	assert.Equal(t, "sent", resp.Status)
	k.mu.Lock()
	msg, err := mail.ReadMessage(bytes.NewReader(k.data))
	k.mu.Unlock()
	require.NoError(t, err)
	assert.Equal(t, "Receipt 42", msg.Header.Get("Subject"))
	body, err := io.ReadAll(msg.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "ACME brand 42", "the design, its variables filled")
	assert.Contains(t, string(body), "Order 42")

	// A failed send is an error the page shows.
	_, err = NewSendTransactionalTestLogic(in("alpha-id"), s).SendTransactionalTest(&types.SendEmailRequest{To: "mike@example.com", TemplateSlug: "missing"})
	var ce *errorx.CodeError
	require.True(t, errors.As(err, &ce))
	assert.Equal(t, "Template 'missing' not found", ce.Message)

	// Not a member of bravo.
	_, err = NewSendTransactionalTestLogic(in("bravo-id"), s).SendTransactionalTest(&types.SendEmailRequest{To: "mike@example.com", TemplateSlug: "receipt"})
	require.True(t, errors.As(err, &ce))
	assert.Equal(t, 403, ce.Code)
}
