package workers

import (
	"context"
	"database/sql"
	"io"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/services/crypto"
	"github.com/outlet-sh/outlet/internal/services/email"

	"github.com/emersion/go-sasl"
	gosmtp "github.com/emersion/go-smtp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// relay is an SMTP server that answers by recipient: temp@ gets a 451 on
// its first attempt and is accepted after, stuck@ always gets a 451, perm@
// always gets a 550, anyone
// else is accepted. It counts every attempt and every accepted message.
type relay struct {
	mu       sync.Mutex
	attempts map[string]int
	accepted map[string]int
}

func (k *relay) NewSession(_ *gosmtp.Conn) (gosmtp.Session, error) { return &relaySession{k: k}, nil }

func (k *relay) counts(rcpt string) (attempts, accepted int) {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.attempts[rcpt], k.accepted[rcpt]
}

type relaySession struct {
	k    *relay
	rcpt string
}

func (s *relaySession) AuthMechanisms() []string { return []string{sasl.Plain} }
func (s *relaySession) Auth(string) (sasl.Server, error) {
	return sasl.NewPlainServer(func(_, _, _ string) error { return nil }), nil
}
func (s *relaySession) Mail(string, *gosmtp.MailOptions) error { return nil }
func (s *relaySession) Rcpt(to string, _ *gosmtp.RcptOptions) error {
	s.rcpt = to
	return nil
}
func (s *relaySession) Data(r io.Reader) error {
	_, _ = io.ReadAll(r)
	s.k.mu.Lock()
	defer s.k.mu.Unlock()
	s.k.attempts[s.rcpt]++
	switch {
	case s.rcpt == "perm@x.example":
		return &gosmtp.SMTPError{Code: 550, EnhancedCode: gosmtp.EnhancedCode{5, 1, 1}, Message: "No such user"}
	case s.rcpt == "stuck@x.example", s.rcpt == "temp@x.example" && s.k.attempts[s.rcpt] == 1:
		return &gosmtp.SMTPError{Code: 451, EnhancedCode: gosmtp.EnhancedCode{4, 3, 0}, Message: "Try again later"}
	}
	s.k.accepted[s.rcpt]++
	return nil
}
func (s *relaySession) Reset()        {}
func (s *relaySession) Logout() error { return nil }

// newRetryHarness is a brand whose marketing mail goes through an smtp
// pathway on a local relay.
func newRetryHarness(t *testing.T) (*db.Store, *email.Service, *relay) {
	t.Helper()
	// Opened as the server opens it: the workers write at once
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	store := db.NewStore(conn)

	k := &relay{attempts: map[string]int{}, accepted: map[string]int{}}
	srv := gosmtp.NewServer(k)
	srv.AllowInsecureAuth = true
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })

	cs, err := crypto.NewService(strings.Repeat("ab", 32))
	require.NoError(t, err)
	svc := email.NewService(store, cs)
	ctx := context.Background()
	for _, q := range []string{
		`INSERT INTO organizations (id, name, slug, api_key, from_email) VALUES ('org-a', 'A', 'a', 'key-a', 'hello@a.example')`,
		`INSERT INTO domain_identities (id, org_id, domain) VALUES ('d-a', 'org-a', 'a.example')`,
		`INSERT INTO contacts (id, org_id, name, email) VALUES
			('c-temp', 'org-a', 'T', 'temp@x.example'), ('c-perm', 'org-a', 'P', 'perm@x.example'),
			('c-sent', 'org-a', 'S', 'sent@x.example'), ('c-failed', 'org-a', 'F', 'failed@x.example'),
			('c-stuck', 'org-a', 'K', 'stuck@x.example')`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err)
	}
	p, err := svc.CreatePathway(ctx, "org-a", email.PathwayInput{
		Provider: email.ProviderSMTP, Kinds: []string{"marketing"}, SMTPHost: "127.0.0.1",
		SMTPPort: int64(ln.Addr().(*net.TCPAddr).Port), SMTPUsername: "outlet", SMTPPassword: "smtp-secret",
	})
	require.NoError(t, err)
	_, err = conn.Exec(`UPDATE domain_identities SET pathway_id = ? WHERE id = 'd-a'`, p.ID)
	require.NoError(t, err)
	return store, svc, k
}

func status(t *testing.T, store *db.Store, table, id string) string {
	t.Helper()
	var s string
	require.NoError(t, store.GetDB().QueryRow(`SELECT status FROM `+table+` WHERE id = ?`, id).Scan(&s))
	return s
}

// A campaign send that fails for now (a 451) waits, pending, and is sent
// once when it comes due; one that fails for good (a 550) is failed and
// never tried again; a send already sent or failed is never sent.
func TestCampaignSends_RetryTemporaryFailuresOnce(t *testing.T) {
	store, svc, k := newRetryHarness(t)
	for _, q := range []string{
		`INSERT INTO email_campaigns (id, org_id, name, subject, html_body, status) VALUES ('camp', 'org-a', 'n', 's', '<p>b</p>', 'sending')`,
		`INSERT INTO campaign_sends (id, campaign_id, contact_id, status) VALUES
			('s-temp', 'camp', 'c-temp', 'pending'), ('s-perm', 'camp', 'c-perm', 'pending'),
			('s-sent', 'camp', 'c-sent', 'sent'), ('s-failed', 'camp', 'c-failed', 'failed')`,
		// Refused for six days: the next refusal gives up.
		`INSERT INTO campaign_sends (id, campaign_id, contact_id, status, first_refused_at) VALUES
			('s-stuck', 'camp', 'c-stuck', 'pending', '` + time.Now().Add(-6*24*time.Hour).UTC().Format(time.RFC3339) + `')`,
	} {
		_, err := store.GetDB().Exec(q)
		require.NoError(t, err)
	}

	config := DefaultCampaignSchedulerConfig()
	config.SendPollInterval = 20 * time.Millisecond
	config.ScheduleInterval = time.Hour
	sched := NewCampaignScheduler(store, svc, config)
	sched.Start()
	t.Cleanup(sched.Stop)

	require.Eventually(t, func() bool {
		a, _ := k.counts("temp@x.example")
		return a == 1 && status(t, store, "campaign_sends", "s-perm") == "failed"
	}, 5*time.Second, 10*time.Millisecond)
	time.Sleep(100 * time.Millisecond) // several polls
	var sendAfter string
	var claim sql.NullString
	require.NoError(t, store.GetDB().QueryRow(`SELECT send_after, claim_token FROM campaign_sends WHERE id = 's-temp'`).Scan(&sendAfter, &claim))
	assert.Equal(t, "pending", status(t, store, "campaign_sends", "s-temp"))
	assert.False(t, claim.Valid, "unclaimed, so a worker takes it again")
	assert.Greater(t, sendAfter, time.Now().UTC().Format(time.DateTime), "it waits before trying again")
	var refused sql.NullString
	require.NoError(t, store.GetDB().QueryRow(`SELECT first_refused_at FROM campaign_sends WHERE id = 's-temp'`).Scan(&refused))
	assert.True(t, refused.Valid, "its first refusal is recorded")
	require.Eventually(t, func() bool { return status(t, store, "campaign_sends", "s-stuck") == "failed" }, 5*time.Second, 10*time.Millisecond)
	a, _ := k.counts("stuck@x.example")
	assert.Equal(t, 1, a, "refused for over five days: given up after one more try")

	a, ok := k.counts("temp@x.example")
	assert.Equal(t, 1, a, "not tried again before it is due")
	assert.Zero(t, ok)

	// It comes due.
	_, err := store.GetDB().Exec(`UPDATE campaign_sends SET send_after = datetime('now', '-1 minute') WHERE id = 's-temp'`)
	require.NoError(t, err)
	require.Eventually(t, func() bool { return status(t, store, "campaign_sends", "s-temp") == "sent" }, 5*time.Second, 10*time.Millisecond)
	time.Sleep(100 * time.Millisecond)

	a, ok = k.counts("temp@x.example")
	assert.Equal(t, 2, a)
	assert.Equal(t, 1, ok, "sent once")
	a, _ = k.counts("perm@x.example")
	assert.Equal(t, 1, a, "a permanent failure is never retried")
	a, _ = k.counts("sent@x.example")
	assert.Zero(t, a, "a sent row is never sent again")
	a, _ = k.counts("failed@x.example")
	assert.Zero(t, a, "a failed row is never sent again")
	assert.Equal(t, "failed", status(t, store, "campaign_sends", "s-failed"))
}

// The same for queued sequence emails, through the email dispatcher.
func TestSequenceEmails_RetryTemporaryFailuresOnce(t *testing.T) {
	store, svc, k := newRetryHarness(t)
	for _, q := range []string{
		`INSERT INTO email_lists (id, public_id, org_id, name, slug) VALUES (1, 'p1', 'org-a', 'L', 'l')`,
		`INSERT INTO email_sequences (id, org_id, list_id, slug, name, trigger_event) VALUES ('seq', 'org-a', 1, 's', 'S', 'manual')`,
		`INSERT INTO email_templates (id, org_id, sequence_id, position, subject, html_body) VALUES ('t1', 'org-a', 'seq', 1, 's', 'b')`,
		`INSERT INTO email_queue (id, contact_id, template_id, scheduled_for, tracking_token, status) VALUES
			('q-temp', 'c-temp', 't1', '2000-01-01T00:00:00Z', 'tok1', 'pending'),
			('q-perm', 'c-perm', 't1', '2000-01-01T00:00:00Z', 'tok2', 'pending'),
			('q-sent', 'c-sent', 't1', '2000-01-01T00:00:00Z', 'tok3', 'sent'),
			('q-failed', 'c-failed', 't1', '2000-01-01T00:00:00Z', 'tok4', 'failed')`,
		// Refused for six days: the next refusal gives up.
		`INSERT INTO email_queue (id, contact_id, template_id, scheduled_for, tracking_token, status, first_refused_at) VALUES
			('q-stuck', 'c-stuck', 't1', '2000-01-01T00:00:00Z', 'tok5', 'pending', '` + time.Now().Add(-6*24*time.Hour).UTC().Format(time.RFC3339) + `')`,
	} {
		_, err := store.GetDB().Exec(q)
		require.NoError(t, err)
	}

	config := email.DefaultDispatcherConfig()
	config.PollInterval = 20 * time.Millisecond
	d := email.NewDispatcher(email.NewSequenceServiceWithBaseURL(store, svc, "https://outlet.example.com"), store, config)
	d.Start()
	t.Cleanup(d.Stop)

	require.Eventually(t, func() bool {
		a, _ := k.counts("temp@x.example")
		return a == 1 && status(t, store, "email_queue", "q-perm") == "failed"
	}, 5*time.Second, 10*time.Millisecond)
	time.Sleep(100 * time.Millisecond)
	var scheduled string
	var claim sql.NullString
	require.NoError(t, store.GetDB().QueryRow(`SELECT scheduled_for, claim_token FROM email_queue WHERE id = 'q-temp'`).Scan(&scheduled, &claim))
	assert.Equal(t, "pending", status(t, store, "email_queue", "q-temp"))
	assert.False(t, claim.Valid, "unclaimed, so a worker takes it again")
	assert.Greater(t, scheduled, time.Now().UTC().Format(time.RFC3339), "it waits before trying again")
	var refused sql.NullString
	require.NoError(t, store.GetDB().QueryRow(`SELECT first_refused_at FROM email_queue WHERE id = 'q-temp'`).Scan(&refused))
	assert.True(t, refused.Valid, "its first refusal is recorded")
	require.Eventually(t, func() bool { return status(t, store, "email_queue", "q-stuck") == "failed" }, 5*time.Second, 10*time.Millisecond)
	a, _ := k.counts("stuck@x.example")
	assert.Equal(t, 1, a, "refused for over five days: given up after one more try")

	a, ok := k.counts("temp@x.example")
	assert.Equal(t, 1, a, "not tried again before it is due")
	assert.Zero(t, ok)

	_, err := store.GetDB().Exec(`UPDATE email_queue SET scheduled_for = '2000-01-01T00:00:00Z' WHERE id = 'q-temp'`)
	require.NoError(t, err)
	require.Eventually(t, func() bool { return status(t, store, "email_queue", "q-temp") == "sent" }, 5*time.Second, 10*time.Millisecond)
	time.Sleep(100 * time.Millisecond)

	a, ok = k.counts("temp@x.example")
	assert.Equal(t, 2, a)
	assert.Equal(t, 1, ok, "sent once")
	a, _ = k.counts("perm@x.example")
	assert.Equal(t, 1, a, "a permanent failure is never retried")
	a, _ = k.counts("sent@x.example")
	assert.Zero(t, a, "a sent row is never sent again")
	a, _ = k.counts("failed@x.example")
	assert.Zero(t, a, "a failed row is never sent again")
}
