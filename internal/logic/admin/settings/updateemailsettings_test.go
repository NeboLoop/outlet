package settings

import (
	"context"
	"database/sql"
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func superAdmin(t *testing.T) (context.Context, *svc.ServiceContext) {
	t.Helper()
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn))
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
	require.NoError(t, update(types.UpdateEmailSettingsRequest{SmtpHost: "smtp.example", SmtpPort: 587, SmtpUser: "u", SmtpPassword: "p"}))
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
	assert.Equal(t, "smtp.example", got["smtp_host"], "saving the sender leaves SMTP alone")
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
