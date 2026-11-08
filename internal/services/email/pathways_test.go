package email

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/crypto"

	gosmtp "github.com/emersion/go-smtp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pathwayHarness is a Service with an encryption key, a moving clock and a
// local SMTP server that accepts the password "smtp-secret".
type pathwayHarness struct {
	s     *Service
	store *db.Store
	clock time.Time
	port  int64
}

func newPathwayHarness(t *testing.T) *pathwayHarness {
	t.Helper()
	_, store := newTestService(t)
	cs, err := crypto.NewService(strings.Repeat("ab", 32))
	require.NoError(t, err)
	h := &pathwayHarness{s: NewService(store, cs), store: store, clock: time.Date(2026, 3, 2, 15, 0, 0, 0, time.UTC)}
	h.s.SetClock(func() time.Time { return h.clock })

	srv := gosmtp.NewServer(&sink{})
	srv.AllowInsecureAuth = true
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	h.port = int64(ln.Addr().(*net.TCPAddr).Port)
	return h
}

// relay adds an smtp pathway on the local server carrying kinds for org-a,
// with a.example as its domain.
func (h *pathwayHarness) relay(t *testing.T, limits Limits, kinds ...string) Pathway {
	t.Helper()
	p, err := h.s.CreatePathway(context.Background(), "org-a", PathwayInput{
		Provider: ProviderSMTP, Kinds: kinds, SMTPHost: "127.0.0.1", SMTPPort: h.port,
		SMTPUsername: "outlet", SMTPPassword: "smtp-secret", Limits: &limits,
	})
	require.NoError(t, err)
	_, err = h.store.GetDB().Exec(`UPDATE domain_identities SET pathway_id = ? WHERE id = 'd-a'`, p.ID)
	require.NoError(t, err)
	return p
}

func (h *pathwayHarness) send(kind Kind) error {
	return h.s.Send(context.Background(), Message{OrgID: "org-a", Kind: kind, To: "c@x.example", Subject: "s", HTML: "b"})
}

func overLimitErr(t *testing.T, err error) time.Time {
	t.Helper()
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrOverLimit), "want ErrOverLimit, got %v", err)
	require.True(t, IsTemporary(err), "a limit is temporary")
	at := RetryAt(err, time.Time{})
	require.False(t, at.IsZero(), "a limit says when it lifts")
	return at
}

// Ticking a kind on one pathway takes it off the others, so each kind has
// exactly one pathway; outreach on a provider other than mailboxes is
// allowed with the warning; a pathway carrying mail can't be deleted.
func TestPathwayKindsAreExclusive(t *testing.T) {
	h := newPathwayHarness(t)
	ctx := context.Background()

	relay := h.relay(t, DefaultLimits(ProviderSMTP), "marketing", "outreach")
	assert.Equal(t, []string{"marketing", "outreach"}, relay.Kinds)
	assert.Equal(t, OutreachWarning, relay.Warning, "outreach on a relay is allowed, with the warning")

	all, err := h.s.ListPathways(ctx, "org-a")
	require.NoError(t, err)
	require.Len(t, all, 2)
	platform := all[0]
	assert.Equal(t, ProviderPlatform, platform.Provider)
	assert.Equal(t, []string{"transactional"}, platform.Kinds, "marketing left the platform pathway")
	assert.Empty(t, platform.Warning)

	// Moving marketing back
	platform, err = h.s.UpdatePathway(ctx, "org-a", platform.ID, PathwayInput{Kinds: []string{"transactional", "marketing"}})
	require.NoError(t, err)
	assert.Equal(t, []string{"transactional", "marketing"}, platform.Kinds)
	got, err := h.s.GetPathway(ctx, "org-a", relay.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"outreach"}, KindsOf(got))

	assert.Error(t, h.s.DeletePathway(ctx, "org-a", relay.ID), "it still carries outreach")
	_, err = h.s.UpdatePathway(ctx, "org-a", relay.ID, PathwayInput{Kinds: []string{}})
	require.NoError(t, err)
	assert.Error(t, h.s.DeletePathway(ctx, "org-a", relay.ID), "it still holds a.example")

	// A mailbox pool gets the pool defaults; credentials never come back.
	pool, err := h.s.CreatePathway(ctx, "org-a", PathwayInput{Provider: ProviderMailboxes})
	require.NoError(t, err)
	assert.Equal(t, DefaultLimits(ProviderMailboxes), pool.Limits)
	assert.Equal(t, Limits{PauseBounceRate: 0.03, PauseMinBounces: 2, PauseSpamBlocks: 2, PauseComplaints: 2}, DefaultLimits(ProviderSES))
	assert.True(t, relay.SMTPPasswordSet)
	require.NoError(t, h.s.DeletePathway(ctx, "org-a", pool.ID))

	_, err = h.s.CreatePathway(ctx, "org-a", PathwayInput{Provider: "postmark"})
	assert.Error(t, err)
	_, err = h.s.CreatePathway(ctx, "org-a", PathwayInput{Provider: ProviderSES, AWSAccessKey: "only-one"})
	assert.Error(t, err, "ses needs both keys")
}

// A sender stops at its daily limit with a temporary error that lifts at
// midnight in the brand's time zone, and a failed delivery gives its place
// back.
func TestDailyLimit(t *testing.T) {
	h := newPathwayHarness(t)
	h.relay(t, Limits{DailyLimit: 2}, "transactional")

	require.NoError(t, h.send(KindTransactional))
	require.NoError(t, h.send(KindTransactional))
	at := overLimitErr(t, h.send(KindTransactional))
	denver := LoadLocation("America/Denver")
	assert.Equal(t, time.Date(2026, 3, 3, 0, 0, 0, 0, denver), at.In(denver), "the next day in the brand's time zone")

	h.clock = at.Add(time.Minute)
	require.NoError(t, h.send(KindTransactional), "a new day")

	// A delivery that fails doesn't use up the allowance.
	_, err := h.store.GetDB().Exec(`UPDATE sending_pathways SET smtp_port = 1 WHERE provider = 'smtp'`)
	require.NoError(t, err)
	require.Error(t, h.send(KindTransactional))
	var n int64
	require.NoError(t, h.store.GetDB().QueryRow(`SELECT n FROM send_counts WHERE day = '2026-03-03'`).Scan(&n))
	assert.EqualValues(t, 1, n)
}

// The warm-up ramp grows from the sender's start date up to the limit.
func TestWarmupRamp(t *testing.T) {
	h := newPathwayHarness(t)
	p := h.relay(t, Limits{DailyLimit: 3, WarmupStart: 1, WarmupStep: 1}, "marketing")
	_, err := h.store.GetDB().Exec(`UPDATE sending_pathways SET created_at = ? WHERE id = ?`, h.clock.Format(TimeLayout), p.ID)
	require.NoError(t, err)

	require.NoError(t, h.send(KindMarketing))
	overLimitErr(t, h.send(KindMarketing))

	h.clock = h.clock.Add(24 * time.Hour)
	require.NoError(t, h.send(KindMarketing))
	require.NoError(t, h.send(KindMarketing))
	overLimitErr(t, h.send(KindMarketing))

	row, err := h.s.GetPathway(context.Background(), "org-a", p.ID)
	require.NoError(t, err)
	start := parseStored(row.CreatedAt)
	for days, want := range []int64{1, 2, 3, 3, 3} {
		n, limited := Allowance(row, start, sql.NullInt64{}, start.Add(time.Duration(days)*24*time.Hour))
		assert.True(t, limited)
		assert.Equal(t, want, n, "day %d", days)
	}
	n, limited := Allowance(row, start, sql.NullInt64{Int64: 50, Valid: true}, start.Add(100*24*time.Hour))
	assert.True(t, limited)
	assert.EqualValues(t, 50, n, "a mailbox's own limit, reached through the ramp")
	_, limited = Allowance(db.SendingPathway{}, start, sql.NullInt64{}, start)
	assert.False(t, limited, "no limit set")

	// Warming up until the ramp reaches the limit in use; the mailbox's own
	// limit is its choice, never capped to the pathway's.
	assert.EqualValues(t, 3, DailyLimitOf(row, sql.NullInt64{}))
	assert.EqualValues(t, 50, DailyLimitOf(row, sql.NullInt64{Int64: 50, Valid: true}))
	assert.True(t, WarmingUp(row, start, sql.NullInt64{}, start.Add(24*time.Hour)))
	assert.False(t, WarmingUp(row, start, sql.NullInt64{}, start.Add(2*24*time.Hour)), "day 3 reaches the limit of 3")
	assert.True(t, WarmingUp(row, start, sql.NullInt64{Int64: 50, Valid: true}, start.Add(10*24*time.Hour)), "an own limit of 50 takes longer to reach")
	assert.False(t, WarmingUp(db.SendingPathway{DailyLimit: 30}, start, sql.NullInt64{}, start), "no warm-up")
}

// Two sends from one sender are at least the minimum gap apart.
func TestSpacing(t *testing.T) {
	h := newPathwayHarness(t)
	h.relay(t, Limits{MinGapSeconds: 60, MaxGapSeconds: 60}, "transactional")

	require.NoError(t, h.send(KindTransactional))
	at := overLimitErr(t, h.send(KindTransactional))
	assert.Equal(t, h.clock.Add(60*time.Second), at)

	h.clock = h.clock.Add(61 * time.Second)
	require.NoError(t, h.send(KindTransactional))
}

// A mailbox pool carrying marketing sends from its least-used mailbox under
// its limit, a mailbox's own limit overrides the pool's, and when every
// mailbox is spent the mail waits.
func TestPoolPicksLeastUsedMailbox(t *testing.T) {
	h := newPathwayHarness(t)
	ctx := context.Background()
	pool, err := h.s.CreatePathway(ctx, "org-a", PathwayInput{Provider: ProviderMailboxes, Kinds: []string{"marketing"}, Limits: &Limits{DailyLimit: 1}})
	require.NoError(t, err)
	_, err = h.store.GetDB().Exec(`INSERT INTO domain_identities (id, org_id, domain, purpose, pathway_id) VALUES ('d-out', 'org-a', 'getexample.com', 'outreach', ?)`, pool.ID)
	require.NoError(t, err)
	pw, err := h.s.cryptoSvc.EncryptString("smtp-secret")
	require.NoError(t, err)
	for _, id := range []string{"mb-1", "mb-2"} {
		_, err := h.store.GetDB().Exec(`INSERT INTO mailboxes (id, org_id, domain_id, email, from_name, smtp_host, smtp_port, smtp_username, smtp_password_enc, imap_host, imap_port, imap_username, imap_password_enc)
			VALUES (?, 'org-a', 'd-out', ? || '@getexample.com', 'Sam', '127.0.0.1', ?, 'sam', ?, '127.0.0.1', 1, 'sam', ?)`, id, id, h.port, pw, pw)
		require.NoError(t, err)
	}
	two := int64(2)
	require.NoError(t, h.s.SetMailboxDailyLimit(ctx, "org-a", "mb-2", &two))
	zero := int64(0)
	assert.Error(t, h.s.SetMailboxDailyLimit(ctx, "org-a", "mb-2", &zero))
	assert.Error(t, h.s.SetMailboxDailyLimit(ctx, "org-b", "mb-2", &two), "another brand's mailbox")

	for range 3 {
		require.NoError(t, h.send(KindMarketing))
	}
	overLimitErr(t, h.send(KindMarketing))

	counts := map[string]int64{}
	rows, err := h.store.ListSendCountsForDay(ctx, db.ListSendCountsForDayParams{PathwayID: pool.ID, Day: Day(h.clock, db.Organization{OutboundTimezone: "America/Denver"})})
	require.NoError(t, err)
	for _, r := range rows {
		counts[r.SenderID] = r.N
	}
	assert.Equal(t, map[string]int64{"mb-1": 1, "mb-2": 2}, counts)

	// Clearing the override puts mb-2 back on the pool's limit.
	require.NoError(t, h.s.SetMailboxDailyLimit(ctx, "org-a", "mb-2", nil))
	mb, err := h.store.GetMailbox(ctx, db.GetMailboxParams{ID: "mb-2", OrgID: "org-a"})
	require.NoError(t, err)
	assert.False(t, mb.DailyLimit.Valid)
}

// A domain moves between the brand's pathways only where its purpose fits;
// an outreach domain takes its mailboxes, and their day's sends, along.
func TestMoveDomain(t *testing.T) {
	h := newPathwayHarness(t)
	ctx := context.Background()
	relay := h.relay(t, Limits{}, "transactional")
	poolA, err := h.s.CreatePathway(ctx, "org-a", PathwayInput{Provider: ProviderMailboxes})
	require.NoError(t, err)
	poolB, err := h.s.CreatePathway(ctx, "org-a", PathwayInput{Name: "Second pool", Provider: ProviderMailboxes, Kinds: []string{"outreach"}})
	require.NoError(t, err)
	_, err = h.store.GetDB().Exec(`INSERT INTO domain_identities (id, org_id, domain, purpose, pathway_id) VALUES ('d-out', 'org-a', 'getexample.com', 'outreach', ?)`, poolA.ID)
	require.NoError(t, err)
	pw, err := h.s.cryptoSvc.EncryptString("pw")
	require.NoError(t, err)
	_, err = h.store.GetDB().Exec(`INSERT INTO mailboxes (id, org_id, domain_id, email, from_name, smtp_host, smtp_port, smtp_username, smtp_password_enc, imap_host, imap_port, imap_username, imap_password_enc)
		VALUES ('mb-1', 'org-a', 'd-out', 'sam@getexample.com', 'Sam', '127.0.0.1', 1, 'sam', ?, '127.0.0.1', 1, 'sam', ?)`, pw, pw)
	require.NoError(t, err)
	_, err = h.store.GetDB().Exec(`INSERT INTO send_counts (pathway_id, sender_id, day, n) VALUES (?, 'mb-1', '2026-03-02', 4), (?, 'mb-1', '2026-03-02', 1)`, poolA.ID, poolB.ID)
	require.NoError(t, err)

	_, err = h.s.MoveDomain(ctx, "org-a", "d-out", relay.ID)
	assert.ErrorContains(t, err, "outreach domains belong to a mailbox pool")
	_, err = h.s.MoveDomain(ctx, "org-a", "d-a", poolB.ID)
	assert.ErrorContains(t, err, "sending domains belong to a platform, ses or smtp pathway")
	_, err = h.s.MoveDomain(ctx, "org-b", "d-out", poolB.ID)
	assert.ErrorContains(t, err, "no such domain", "another brand's domain")
	assert.ErrorContains(t, h.s.DeletePathway(ctx, "org-a", poolA.ID), "move them to another pathway first")

	to, err := h.s.MoveDomain(ctx, "org-a", "d-out", poolB.ID)
	require.NoError(t, err)
	assert.Equal(t, "Second pool", to.Name)
	got, err := h.store.GetMailboxPathway(ctx, "mb-1")
	require.NoError(t, err)
	assert.Equal(t, poolB.ID, got.ID, "the mailbox moved with its domain")
	n, err := h.store.SumSendCountsSince(ctx, db.SumSendCountsSinceParams{PathwayID: poolB.ID, SenderID: "mb-1", SinceDay: "2026-03-02"})
	require.NoError(t, err)
	assert.EqualValues(t, 5, n, "the day's sends count in the new pool")
	n, err = h.store.SumSendCountsSince(ctx, db.SumSendCountsSinceParams{PathwayID: poolA.ID, SenderID: "mb-1", SinceDay: "2026-03-02"})
	require.NoError(t, err)
	assert.Zero(t, n)

	// The emptied pool can go now.
	require.NoError(t, h.s.DeletePathway(ctx, "org-a", poolA.ID))

	// A sending domain moves between sending pathways.
	platform, err := h.s.CreatePathway(ctx, "org-a", PathwayInput{Provider: ProviderPlatform})
	require.NoError(t, err)
	_, err = h.s.MoveDomain(ctx, "org-a", "d-a", platform.ID)
	require.NoError(t, err)
	views, err := h.s.ListPathways(ctx, "org-a")
	require.NoError(t, err)
	for _, v := range views {
		if v.ID == platform.ID {
			assert.Equal(t, []PathwayDomain{{ID: "d-a", Domain: "a.example", Purpose: "sending"}}, v.Domains)
		}
	}
}

// The health thresholds are the pathway's.
func TestHealthPause(t *testing.T) {
	p := db.SendingPathway{PauseBounceRate: 0.03, PauseMinBounces: 2, PauseSpamBlocks: 2, PauseComplaints: 2}
	assert.Empty(t, HealthPause(p, Health{Sent: 10, Bounces: 1}), "one bounce alone never pauses")
	assert.Contains(t, HealthPause(p, Health{Sent: 10, Bounces: 2}), "bounce rate")
	assert.Empty(t, HealthPause(p, Health{Sent: 100, Bounces: 2}), "2% is under 3%")
	assert.Contains(t, HealthPause(p, Health{SpamBlocks: 2}), "spam")
	assert.Contains(t, HealthPause(p, Health{Complaints: 2}), "complaints")
	p.PauseComplaints = 5
	assert.Empty(t, HealthPause(p, Health{Complaints: 2}), "the pathway's own threshold")
	assert.Empty(t, HealthPause(db.SendingPathway{}, Health{Sent: 1, Bounces: 1, SpamBlocks: 9, Complaints: 9}), "0 turns a pause off")
}

// The sequence worker defers an email held back by its pathway's limit: it
// stays pending, unclaimed, until the limit lifts. It is never failed.
func TestSequenceEmailsWaitForTheLimit(t *testing.T) {
	h := newPathwayHarness(t)
	h.relay(t, Limits{DailyLimit: 1}, "marketing")
	ctx := context.Background()
	for _, q := range []string{
		`INSERT INTO contacts (id, org_id, name, email) VALUES ('c1', 'org-a', 'C', 'c1@x.example'), ('c2', 'org-a', 'D', 'c2@x.example')`,
		`INSERT INTO email_lists (id, public_id, org_id, name, slug) VALUES (1, 'p1', 'org-a', 'L', 'l')`,
		`INSERT INTO email_sequences (id, org_id, list_id, slug, name, trigger_event) VALUES ('seq', 'org-a', 1, 's', 'S', 'manual')`,
		`INSERT INTO email_templates (id, org_id, sequence_id, position, subject, html_body) VALUES ('t1', 'org-a', 'seq', 1, 's', 'b')`,
		`INSERT INTO email_queue (id, contact_id, template_id, scheduled_for, tracking_token) VALUES
			('q1', 'c1', 't1', '2000-01-01T00:00:00Z', 'tok1'), ('q2', 'c2', 't1', '2000-01-01T00:00:01Z', 'tok2')`,
	} {
		_, err := h.store.GetDB().Exec(q)
		require.NoError(t, err)
	}
	seq := NewSequenceServiceWithBaseURL(h.store, h.s, "https://outlet.example.com")
	sent, err := seq.ProcessPendingEmails(ctx, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, sent)

	var status, scheduled string
	var claim sql.NullString
	require.NoError(t, h.store.GetDB().QueryRow(`SELECT status, scheduled_for, claim_token FROM email_queue WHERE id = 'q2'`).Scan(&status, &scheduled, &claim))
	assert.Equal(t, "pending", status)
	assert.False(t, claim.Valid, "unclaimed, so a worker takes it again")
	assert.Greater(t, scheduled, h.clock.UTC().Format(time.RFC3339), "it waits until the limit lifts")
}
