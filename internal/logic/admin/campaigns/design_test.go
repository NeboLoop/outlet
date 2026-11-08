package campaigns

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"mime/quotedprintable"
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

func (k *mailSink) NewSession(_ *gosmtp.Conn) (gosmtp.Session, error) {
	return &mailSinkSession{k}, nil
}

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

// body returns the decoded text of the last message received.
func (k *mailSink) body(t *testing.T) (subject, body string) {
	t.Helper()
	k.mu.Lock()
	defer k.mu.Unlock()
	msg, err := mail.ReadMessage(bytes.NewReader(k.data))
	require.NoError(t, err)
	raw, err := io.ReadAll(msg.Body)
	require.NoError(t, err)
	decoded, err := io.ReadAll(quotedprintable.NewReader(bytes.NewReader(raw)))
	require.NoError(t, err)
	return msg.Header.Get("Subject"), string(decoded)
}

type designFixture struct {
	svc       *svc.ServiceContext
	sink      *mailSink
	asMember  context.Context
	designID  string
	foreignID string
}

// newDesignFixture: brand alpha with a member and a "Branded" design, brand
// bravo with a design of its own, and an SMTP relay that records mail.
func newDesignFixture(t *testing.T) designFixture {
	t.Helper()
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	store := db.NewStore(conn)
	ctx := context.Background()

	k := &mailSink{}
	srv := gosmtp.NewServer(k)
	srv.AllowInsecureAuth = true
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	for key, value := range map[string]string{
		"smtp_host": "127.0.0.1", "smtp_port": strconv.Itoa(ln.Addr().(*net.TCPAddr).Port), "smtp_user": "outlet", "smtp_password": "secret",
		"from_email": "team@outlet.example",
	} {
		_, err := conn.Exec(`INSERT INTO platform_settings (key, value_text, category) VALUES (?, ?, 'email')`, key, value)
		require.NoError(t, err)
	}

	_, err = store.CreateUser(ctx, db.CreateUserParams{ID: "mike", Email: "mike@example.com", PasswordHash: "x", Name: "M", Role: "agent", Status: "active", EmailVerified: int64(1)})
	require.NoError(t, err)
	var designs []string
	for _, b := range []string{"alpha", "bravo"} {
		_, err = store.CreateOrganization(ctx, db.CreateOrganizationParams{ID: b + "-id", Name: b, Slug: b, ApiKey: "k-" + b})
		require.NoError(t, err)
		d, err := store.CreateEmailDesign(ctx, db.CreateEmailDesignParams{
			OrgID: b + "-id", Name: "Branded", Slug: "branded",
			HtmlBody: `<html><body><h1>` + b + ` header</h1>{{content}}<a href="{{unsubscribe_url}}">Unsubscribe</a></body></html>`,
		})
		require.NoError(t, err)
		designs = append(designs, strconv.FormatInt(d.ID, 10))
	}
	require.NoError(t, store.AddUserToOrganization(ctx, db.AddUserToOrganizationParams{OrgID: "alpha-id", UserID: "mike", Role: sql.NullString{String: "member", Valid: true}}))
	user, err := access.Active(ctx, store, "mike")
	require.NoError(t, err)

	emailService := email.NewService(store, nil)
	emailService.SetBaseURL("https://outlet.test")
	return designFixture{
		svc:       &svc.ServiceContext{DB: store, EmailService: emailService},
		sink:      k,
		asMember:  access.WithBrand(access.WithUser(ctx, user), "alpha-id"),
		designID:  designs[0],
		foreignID: designs[1],
	}
}

// A brand member's test send goes out in the campaign's design, built like a
// real send.
func TestSendCampaignTest_UsesTheDesign(t *testing.T) {
	f := newDesignFixture(t)

	resp, err := NewSendCampaignTestLogic(f.asMember, f.svc).SendCampaignTest(&types.SendCampaignTestRequest{
		To: "mike@example.com", Subject: "Hello {{first_name}}", HtmlBody: "<p>Hi {{name}}, big news</p>", DesignId: f.designID,
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)

	subject, body := f.sink.body(t)
	assert.Equal(t, "[Test] Hello Test", subject)
	assert.Contains(t, body, "<h1>alpha header</h1><p>Hi Test Subscriber, big news</p>")
	assert.Contains(t, body, `href="https://outlet.test/u/test"`, "the design's unsubscribe link is filled")

	// Another brand's design is refused.
	_, err = NewSendCampaignTestLogic(f.asMember, f.svc).SendCampaignTest(&types.SendCampaignTestRequest{
		To: "mike@example.com", Subject: "Hi", HtmlBody: "<p>Hi</p>", DesignId: f.foreignID,
	})
	assert.Error(t, err)
}

// The editor can set, keep and remove a campaign's design.
func TestUpdateCampaign_Design(t *testing.T) {
	f := newDesignFixture(t)
	created, err := NewCreateCampaignLogic(f.asMember, f.svc).CreateCampaign(&types.CreateCampaignRequest{
		Name: "News", Subject: "Hi", HtmlBody: "<p>Hi</p>", DesignId: &f.designID,
	})
	require.NoError(t, err)
	require.NotNil(t, created.DesignId)
	assert.Equal(t, f.designID, *created.DesignId)

	updated, err := NewUpdateCampaignLogic(f.asMember, f.svc).UpdateCampaign(&types.UpdateCampaignRequest{Id: created.Id, Subject: "Hello"})
	require.NoError(t, err)
	require.NotNil(t, updated.DesignId, "no design_id keeps the design")

	none := ""
	updated, err = NewUpdateCampaignLogic(f.asMember, f.svc).UpdateCampaign(&types.UpdateCampaignRequest{Id: created.Id, DesignId: &none})
	require.NoError(t, err)
	assert.Nil(t, updated.DesignId, `"" removes the design`)

	_, err = NewUpdateCampaignLogic(f.asMember, f.svc).UpdateCampaign(&types.UpdateCampaignRequest{Id: created.Id, DesignId: &f.foreignID})
	assert.Error(t, err, "another brand's design is refused")
}
