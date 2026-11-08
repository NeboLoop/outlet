package settings

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net"
	"path/filepath"
	"strings"
	"testing"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/services/crypto"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/emersion/go-sasl"
	gosmtp "github.com/emersion/go-smtp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func superAdmin(t *testing.T) (context.Context, *svc.ServiceContext) {
	t.Helper()
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	store := db.NewStore(conn)
	ctx := context.Background()
	_, err = store.CreateUser(ctx, db.CreateUserParams{
		ID: "root", Email: "root@example.com", PasswordHash: "x", Name: "root", Role: "super_admin", Status: "active", EmailVerified: int64(1),
	})
	require.NoError(t, err)
	user, err := access.Active(ctx, store, "root")
	require.NoError(t, err)
	cryptoSvc, err := crypto.NewService(strings.Repeat("ab", 32))
	require.NoError(t, err)
	return access.WithUser(ctx, user), &svc.ServiceContext{DB: store, CryptoService: cryptoSvc, EmailService: email.NewService(store, cryptoSvc)}
}

func TestUpdateEmailSettings_PlatformSender(t *testing.T) {
	ctx, s := superAdmin(t)
	update := func(req types.UpdateEmailSettingsRequest) error {
		_, err := NewUpdateEmailSettingsLogic(ctx, s).UpdateEmailSettings(&req)
		return err
	}
	sender := func() map[string]string {
		rows, err := s.DB.GetPlatformSettingsByCategory(ctx, "email")
		require.NoError(t, err)
		got := map[string]string{}
		for _, r := range rows {
			got[r.Key] = r.ValueText.String
		}
		return got
	}

	// SMTP saved, no sender yet: setup status says SMTP works and From is missing.
	port := smtpServer(t)
	require.NoError(t, update(types.UpdateEmailSettingsRequest{SmtpHost: "127.0.0.1", SmtpPort: port, SmtpUser: "u", SmtpPassword: "p"}))
	status, err := NewGetSetupStatusLogic(ctx, s).GetSetupStatus()
	require.NoError(t, err)
	assert.True(t, status.HasSmtp, "SMTP is configured without a From address")
	assert.False(t, status.PlatformConfigured)
	assert.Equal(t, []string{"from_email"}, status.MissingSettings)

	require.NoError(t, update(types.UpdateEmailSettingsRequest{FromEmail: "outlet@example.com", FromName: "Outlet", ReplyTo: "help@example.com"}))
	got := sender()
	assert.Equal(t, "outlet@example.com", got["from_email"])
	assert.Equal(t, "Outlet", got["from_name"])
	assert.Equal(t, "help@example.com", got["reply_to"])
	assert.Equal(t, "127.0.0.1", got["smtp_host"], "saving the sender leaves SMTP alone")
	status, err = NewGetSetupStatusLogic(ctx, s).GetSetupStatus()
	require.NoError(t, err)
	assert.True(t, status.PlatformConfigured)

	// The sender is saved as one: a blank name or Reply-To clears it.
	require.NoError(t, update(types.UpdateEmailSettingsRequest{FromEmail: "outlet@example.com"}))
	got = sender()
	assert.Equal(t, "", got["from_name"])
	assert.Equal(t, "", got["reply_to"])

	assert.Error(t, update(types.UpdateEmailSettingsRequest{FromEmail: "not an address"}))
	assert.Error(t, update(types.UpdateEmailSettingsRequest{FromEmail: "outlet@example.com", ReplyTo: "nope"}))
	assert.Error(t, update(types.UpdateEmailSettingsRequest{FromEmail: "outlet@example.com", FromName: "a\r\nBcc: x@y.z"}))
}

// smtpServer starts a local SMTP server that accepts the login u / p and
// returns its port.
func smtpServer(t *testing.T) int {
	t.Helper()
	srv := gosmtp.NewServer(loginBackend{})
	srv.AllowInsecureAuth = true
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	return ln.Addr().(*net.TCPAddr).Port
}

type loginBackend struct{}

func (loginBackend) NewSession(*gosmtp.Conn) (gosmtp.Session, error) { return loginSession{}, nil }

type loginSession struct{}

func (loginSession) AuthMechanisms() []string { return []string{sasl.Plain} }
func (loginSession) Auth(string) (sasl.Server, error) {
	return sasl.NewPlainServer(func(_, user, pass string) error {
		if user != "u" || pass != "p" {
			return errors.New("bad login")
		}
		return nil
	}), nil
}
func (loginSession) Mail(string, *gosmtp.MailOptions) error { return nil }
func (loginSession) Rcpt(string, *gosmtp.RcptOptions) error { return nil }
func (loginSession) Data(io.Reader) error                   { return nil }
func (loginSession) Reset()                                 {}
func (loginSession) Logout() error                          { return nil }

// Platform SMTP settings are saved only once they connect and log in: a
// server with no username or password, a login the server turns down and a
// server that isn't there are refused, and nothing is saved. A blank
// password keeps the stored one.
func TestUpdateEmailSettings_SMTPMustLogIn(t *testing.T) {
	ctx, s := superAdmin(t)
	update := func(req types.UpdateEmailSettingsRequest) error {
		_, err := NewUpdateEmailSettingsLogic(ctx, s).UpdateEmailSettings(&req)
		return err
	}
	stored := func() map[string]bool {
		rows, err := s.DB.GetPlatformSettingsByCategory(ctx, "email")
		require.NoError(t, err)
		got := map[string]bool{}
		for _, r := range rows {
			got[r.Key] = true
		}
		return got
	}
	port := smtpServer(t)

	err := update(types.UpdateEmailSettingsRequest{SmtpHost: "127.0.0.1", SmtpPort: port})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "username and password")
	err = update(types.UpdateEmailSettingsRequest{SmtpHost: "127.0.0.1", SmtpPort: port, SmtpUser: "u", SmtpPassword: "wrong"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Couldn't connect and log in")
	closed, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	closedPort := closed.Addr().(*net.TCPAddr).Port
	closed.Close()
	assert.Error(t, update(types.UpdateEmailSettingsRequest{SmtpHost: "127.0.0.1", SmtpPort: closedPort, SmtpUser: "u", SmtpPassword: "p"}))
	assert.False(t, stored()["smtp_host"], "nothing is saved when the test fails")

	require.NoError(t, update(types.UpdateEmailSettingsRequest{SmtpHost: "127.0.0.1", SmtpPort: port, SmtpUser: "u", SmtpPassword: "p"}))
	require.NoError(t, update(types.UpdateEmailSettingsRequest{SmtpHost: "127.0.0.1", SmtpPort: port, SmtpUser: "u"}), "a blank password keeps the stored one")
	assert.Error(t, update(types.UpdateEmailSettingsRequest{SmtpHost: "127.0.0.1", SmtpPort: port, SmtpUser: "someone-else"}))
}
