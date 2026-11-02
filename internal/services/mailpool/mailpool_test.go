package mailpool

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"github.com/outlet-sh/outlet/internal/config"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/events"
	"github.com/outlet-sh/outlet/internal/services/crypto"
	"github.com/outlet-sh/outlet/internal/services/webhook"
)

const (
	org    = "org-hub"
	bounce = "bounce.pool.test"
)

func testConfig() config.PoolConfig {
	return config.PoolConfig{
		Enabled:               "true",
		BounceDomain:          bounce,
		RampStart:             5,
		RampStep:              2,
		Ceiling:               30,
		MinGapSeconds:         240,
		RestDays:              30,
		BouncePausePct:        3.0,
		BounceWindow:          100,
		BounceMinSample:       20,
		ComplaintPausePct:     0.1,
		ComplaintWindow:       1000,
		ComplaintMinCount:     1,
		PolicyBlocksPerDay:    3,
		ProviderFaultsToPause: 3,
		DomainPauseShare:      0.5,
		RepeatPauses:          2,
		RepeatPauseDays:       30,
	}
}

type harness struct {
	t     *testing.T
	s     *Service
	bus   *events.Subject
	store *db.Store
	smtp  *fakeSMTP
	clock time.Time
	ctx   context.Context

	mu     sync.Mutex
	events []recorded
}

type recorded struct {
	topic string
	ev    events.PoolEvent
}

func newHarness(t *testing.T, cfg config.PoolConfig) *harness {
	t.Helper()
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn))
	store := db.NewStore(conn)

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	cs, err := crypto.NewService(hex.EncodeToString(key))
	require.NoError(t, err)

	bus := events.NewSubject(events.WithBufferSize(64))
	h := &harness{t: t, bus: bus, store: store, smtp: newFakeSMTP(t), ctx: context.Background(),
		clock: time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)}
	for _, topic := range []string{events.TopicPoolMessageBounced, events.TopicPoolMessageComplained, events.TopicPoolMailboxPaused, events.TopicPoolMailboxResumed} {
		topic := topic
		events.Subscribe[events.PoolEvent](bus, topic, func(_ context.Context, ev events.PoolEvent) error {
			h.mu.Lock()
			h.events = append(h.events, recorded{topic, ev})
			h.mu.Unlock()
			return nil
		})
	}
	h.s = New(cfg, store, cs, bus)
	h.s.now = func() time.Time { return h.clock }
	h.s.lookupMX = func(_ context.Context, domain string) ([]*net.MX, error) {
		if domain == "nomx.test" {
			return nil, nil
		}
		return []*net.MX{{Host: "mx." + domain, Pref: 10}}, nil
	}
	t.Cleanup(func() {
		h.s.mu.Lock()
		for _, p := range h.s.smtp {
			p.Close()
		}
		h.s.mu.Unlock()
	})
	return h
}

func (h *harness) advance(d time.Duration) { h.clock = h.clock.Add(d) }
func (h *harness) gap()                    { h.advance(time.Duration(h.s.cfg.MinGapSeconds) * time.Second) }

// addMailboxes makes a domain with n mailboxes on the fake SMTP server,
// warm-started warmDays ago.
func (h *harness) addMailboxes(domain string, n, warmDays int) (db.PoolDomain, []db.PoolMailbox) {
	h.t.Helper()
	d, err := h.s.CreateDomain(h.ctx, domain)
	require.NoError(h.t, err)
	var out []db.PoolMailbox
	for i := 0; i < n; i++ {
		mb, err := h.s.CreateMailbox(h.ctx, MailboxInput{
			DomainID: d.ID, Address: fmt.Sprintf("rep%d@%s", i, domain),
			SMTPHost: "127.0.0.1", SMTPPort: h.smtp.port, SMTPUsername: "user", SMTPPassword: "secret",
			WarmStartedAt: h.clock.Add(-time.Duration(warmDays) * 24 * time.Hour).Format(time.RFC3339),
		})
		require.NoError(h.t, err)
		out = append(out, mb)
	}
	return d, out
}

func (h *harness) send(tenant, to, key string) (*SendResult, error) {
	return h.s.Send(h.ctx, SendInput{
		OrgID: org, Tenant: tenant, To: to, FromName: "Dana at Acme", Subject: "Quick question",
		Text: "Hi there,\nA short note.\n", ListUnsubscribeURL: "https://optout.pool.test/u/tok",
		ListUnsubscribeMailto: "unsubscribe@optout.pool.test", IdempotencyKey: key,
	})
}

func (h *harness) recorded(topic string) []events.PoolEvent {
	var out []events.PoolEvent
	require.Eventually(h.t, func() bool {
		h.mu.Lock()
		defer h.mu.Unlock()
		out = nil
		for _, r := range h.events {
			if r.topic == topic {
				out = append(out, r.ev)
			}
		}
		return len(out) > 0
	}, 2*time.Second, 10*time.Millisecond, "no %s event", topic)
	return out
}

func httpCode(err error) int {
	var ce *errorx.CodeError
	if errors.As(err, &ce) {
		return ce.Code
	}
	return 0
}

func TestRampAllowancePerDay(t *testing.T) {
	s := New(testConfig(), nil, nil, nil)
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	mb := db.PoolMailbox{WarmStartedAt: fmtTime(start)}
	for days, want := range map[int]int{0: 5, 1: 7, 2: 9, 12: 29, 13: 30, 40: 30} {
		assert.Equal(t, want, s.allowance(mb, start.Add(time.Duration(days)*24*time.Hour+time.Hour)), "day %d", days)
	}
	// Before the warm start it is still day 0.
	assert.Equal(t, 5, s.allowance(mb, start.Add(-time.Hour)))
}

func TestPoolOffAnswers503(t *testing.T) {
	cfg := testConfig()
	cfg.Enabled = ""
	h := newHarness(t, cfg)
	_, err := h.send("acct-1", "lead@example.com", "k1")
	assert.Equal(t, http.StatusServiceUnavailable, httpCode(err))
	assert.Contains(t, err.Error(), "pool is off")
	_, err = h.s.Capacity(h.ctx, org, "acct-1")
	assert.Equal(t, http.StatusServiceUnavailable, httpCode(err))
	_, err = h.s.CreateDomain(h.ctx, "outreach-a.test")
	assert.Equal(t, http.StatusServiceUnavailable, httpCode(err))
}

func TestDuplicateDomainAndMailboxAnswer409(t *testing.T) {
	h := newHarness(t, testConfig())
	d, mbs := h.addMailboxes("outreach-a.test", 1, 0)
	_, err := h.s.CreateDomain(h.ctx, "Outreach-A.test")
	assert.Equal(t, http.StatusConflict, httpCode(err))
	_, err = h.s.CreateMailbox(h.ctx, MailboxInput{DomainID: d.ID, Address: mbs[0].Address, SMTPHost: "127.0.0.1",
		SMTPPort: 587, SMTPUsername: "u", SMTPPassword: "p"})
	assert.Equal(t, http.StatusConflict, httpCode(err))
	// The password is stored encrypted, never as given.
	assert.NotContains(t, string(mbs[0].SmtpPasswordEnc), "secret")
}

func TestSendWithNoMailboxesAnswers503(t *testing.T) {
	h := newHarness(t, testConfig())
	_, err := h.send("acct-1", "lead@example.com", "k1")
	assert.Equal(t, http.StatusServiceUnavailable, httpCode(err))
	assert.Contains(t, err.Error(), "no mailboxes")
}

func TestSendSetsEnvelopeAndHeaders(t *testing.T) {
	h := newHarness(t, testConfig())
	h.addMailboxes("outreach-a.test", 1, 0)
	_, assigned, err := h.s.Assign(h.ctx, org, "acct-1", 1)
	require.NoError(t, err)
	require.Equal(t, 1, assigned)

	res, err := h.send("acct-1", "Lead@Example.com", "touch-1")
	require.NoError(t, err)
	assert.Equal(t, "sent", res.Status)
	assert.Equal(t, "rep0@outreach-a.test", res.MailboxAddress)
	id := strings.SplitN(res.MessageID, "@", 2)[0]
	assert.Equal(t, "outreach-a.test", strings.SplitN(res.MessageID, "@", 2)[1])

	msgs := h.smtp.sent()
	require.Len(t, msgs, 1)
	m := msgs[0]
	assert.Equal(t, "b+"+id+"@"+bounce, m.MailFrom, "VERP Return-Path")
	assert.Equal(t, []string{"lead@example.com"}, m.Rcpt)
	assert.Contains(t, m.Data, "Message-ID: <"+res.MessageID+">\r\n")
	assert.Contains(t, m.Data, "List-Unsubscribe: <https://optout.pool.test/u/tok>, <mailto:unsubscribe@optout.pool.test>\r\n")
	assert.Contains(t, m.Data, "List-Unsubscribe-Post: List-Unsubscribe=One-Click\r\n")
	assert.Contains(t, m.Data, `From: "Dana at Acme" <rep0@outreach-a.test>`)
	assert.Contains(t, m.Data, "Content-Type: text/plain; charset=UTF-8")
	assert.NotContains(t, m.Data, "text/html")
	assert.Contains(t, m.Data, "Date: ")
}

func TestGapAndDailyAllowance(t *testing.T) {
	h := newHarness(t, testConfig())
	h.addMailboxes("outreach-a.test", 1, 0) // day 0: 5 a day
	_, _, err := h.s.Assign(h.ctx, org, "acct-1", 1)
	require.NoError(t, err)

	_, err = h.send("acct-1", "a0@example.com", "k0")
	require.NoError(t, err)
	_, err = h.send("acct-1", "a1@example.com", "k1")
	assert.Equal(t, http.StatusTooManyRequests, httpCode(err))
	assert.Contains(t, err.Error(), "inside the gap")

	for i := 1; i < 5; i++ {
		h.gap()
		_, err = h.send("acct-1", fmt.Sprintf("a%d@example.com", i), fmt.Sprintf("k%d", i))
		require.NoError(t, err)
	}
	h.gap()
	_, err = h.send("acct-1", "a5@example.com", "k5")
	assert.Equal(t, http.StatusTooManyRequests, httpCode(err))
	assert.Contains(t, err.Error(), "daily allowance used")

	c, err := h.s.Capacity(h.ctx, org, "acct-1")
	require.NoError(t, err)
	require.Len(t, c.Mailboxes, 1)
	assert.Equal(t, 0, c.Mailboxes[0].RemainingToday)
	assert.Equal(t, time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC), c.Mailboxes[0].NextAllowedAt)

	// The next UTC day the ramp allows 7.
	h.clock = time.Date(2026, 11, 3, 9, 0, 0, 0, time.UTC)
	c, err = h.s.Capacity(h.ctx, org, "acct-1")
	require.NoError(t, err)
	assert.Equal(t, 7, c.Mailboxes[0].RemainingToday)
	_, err = h.send("acct-1", "a5@example.com", "k5")
	require.NoError(t, err)
}

func TestIdempotentReplaySendsNothing(t *testing.T) {
	h := newHarness(t, testConfig())
	h.addMailboxes("outreach-a.test", 1, 10)
	_, _, err := h.s.Assign(h.ctx, org, "acct-1", 1)
	require.NoError(t, err)

	first, err := h.send("acct-1", "lead@example.com", "touch-42")
	require.NoError(t, err)
	h.gap()
	again, err := h.send("acct-1", "lead@example.com", "touch-42")
	require.NoError(t, err)
	assert.Equal(t, first, again)
	assert.Len(t, h.smtp.sent(), 1)
}

func TestStickyMailboxAndRestDays(t *testing.T) {
	h := newHarness(t, testConfig())
	_, mbs := h.addMailboxes("outreach-a.test", 2, 10)
	_, assigned, err := h.s.Assign(h.ctx, org, "acct-1", 2)
	require.NoError(t, err)
	require.Equal(t, 2, assigned)

	first, err := h.send("acct-1", "lead@example.com", "t1")
	require.NoError(t, err)
	// Another lead goes to the readier mailbox (the other one).
	other, err := h.send("acct-1", "second@example.com", "t2")
	require.NoError(t, err)
	assert.NotEqual(t, first.MailboxID, other.MailboxID)
	// The follow-up to the first lead stays on its mailbox, even when the
	// other one has more left today.
	h.gap()
	follow, err := h.send("acct-1", "lead@example.com", "t3")
	require.NoError(t, err)
	assert.Equal(t, first.MailboxID, follow.MailboxID)

	// Another tenant's mailbox can't be named.
	_, _, err = h.s.Assign(h.ctx, org, "acct-2", 1)
	assert.Equal(t, http.StatusServiceUnavailable, httpCode(err), "every mailbox is taken")
	_, err = h.s.Send(h.ctx, SendInput{OrgID: org, Tenant: "acct-2", MailboxID: mbs[0].ID, To: "x@example.com",
		Subject: "s", Text: "t", ListUnsubscribeURL: "https://o.test/u", IdempotencyKey: "z"})
	assert.Equal(t, http.StatusServiceUnavailable, httpCode(err))

	// Released mailboxes rest RestDays before another tenant gets them.
	n, err := h.s.Release(h.ctx, org, "acct-1")
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	_, _, err = h.s.Assign(h.ctx, org, "acct-2", 2)
	assert.Equal(t, http.StatusServiceUnavailable, httpCode(err), "still resting")
	// The same tenant may take them back at once.
	_, assigned, err = h.s.Assign(h.ctx, org, "acct-1", 1)
	require.NoError(t, err)
	assert.Equal(t, 1, assigned)
	_, err = h.s.Release(h.ctx, org, "acct-1")
	require.NoError(t, err)

	h.advance(31 * 24 * time.Hour)
	c, assigned, err := h.s.Assign(h.ctx, org, "acct-2", 2)
	require.NoError(t, err)
	assert.Equal(t, 2, assigned)
	assert.Len(t, c.Mailboxes, 2)
	for _, m := range c.Mailboxes {
		assert.Equal(t, "active", m.Status)
	}
	// Old sends still name the first tenant.
	send, err := h.store.GetPoolSendByMessageID(h.ctx, first.MessageID)
	require.NoError(t, err)
	assert.Equal(t, "acct-1", send.Tenant)
}

const dsnTemplate = "From: MAILER-DAEMON@mx.example.com\r\n" +
	"To: %s\r\n" +
	"Subject: Undelivered Mail Returned to Sender\r\n" +
	"MIME-Version: 1.0\r\n" +
	"Content-Type: multipart/report; report-type=delivery-status; boundary=\"BOUND\"\r\n" +
	"\r\n" +
	"--BOUND\r\n" +
	"Content-Type: text/plain\r\n" +
	"\r\n" +
	"Your message could not be delivered.\r\n" +
	"--BOUND\r\n" +
	"Content-Type: message/delivery-status\r\n" +
	"\r\n" +
	"Reporting-MTA: dns; mx.example.com\r\n" +
	"\r\n" +
	"Final-Recipient: rfc822; %s\r\n" +
	"Action: %s\r\n" +
	"Status: %s\r\n" +
	"Diagnostic-Code: smtp; %s\r\n" +
	"\r\n" +
	"--BOUND\r\n" +
	"Content-Type: text/rfc822-headers\r\n" +
	"\r\n" +
	"Message-ID: <%s>\r\n" +
	"Subject: Quick question\r\n" +
	"--BOUND--\r\n"

const arfTemplate = "From: feedback@isp.example\r\n" +
	"To: abuse@" + bounce + "\r\n" +
	"Subject: FW: complaint\r\n" +
	"MIME-Version: 1.0\r\n" +
	"Content-Type: multipart/report; report-type=feedback-report; boundary=\"ARF\"\r\n" +
	"\r\n" +
	"--ARF\r\n" +
	"Content-Type: text/plain\r\n" +
	"\r\n" +
	"This is an abuse report.\r\n" +
	"--ARF\r\n" +
	"Content-Type: message/feedback-report\r\n" +
	"\r\n" +
	"Feedback-Type: abuse\r\n" +
	"User-Agent: isp-fbl/1.0\r\n" +
	"Version: 1\r\n" +
	"Original-Mail-From: <%s>\r\n" +
	"Original-Rcpt-To: <%s>\r\n" +
	"\r\n" +
	"--ARF\r\n" +
	"Content-Type: message/rfc822\r\n" +
	"\r\n" +
	"Message-ID: <%s>\r\n" +
	"Subject: Quick question\r\n" +
	"\r\n" +
	"Hi there\r\n" +
	"--ARF--\r\n"

func verp(res *SendResult) string {
	return "b+" + strings.SplitN(res.MessageID, "@", 2)[0] + "@" + bounce
}

func TestReportParsingDSNAndARF(t *testing.T) {
	h := newHarness(t, testConfig())
	h.addMailboxes("outreach-a.test", 1, 20)
	_, _, err := h.s.Assign(h.ctx, org, "acct-1", 1)
	require.NoError(t, err)

	hard, err := h.send("acct-1", "gone@example.com", "t1")
	require.NoError(t, err)
	h.gap()
	soft, err := h.send("acct-1", "full@example.com", "t2")
	require.NoError(t, err)
	h.gap()
	angry, err := h.send("acct-1", "angry@example.com", "t3")
	require.NoError(t, err)

	// Hard bounce matched by VERP.
	r, err := h.s.IngestReport(h.ctx, org, []byte(fmt.Sprintf(dsnTemplate, verp(hard), "gone@example.com", "failed", "5.1.1", "550 5.1.1 no such user", "unrelated@elsewhere")))
	require.NoError(t, err)
	assert.Equal(t, &ReportResult{MessageID: hard.MessageID, MailboxID: hard.MailboxID, Tenant: "acct-1", Email: "gone@example.com", Kind: "bounce", Hard: true}, r)
	var poolSendID sql.NullString
	var bounceType string
	require.NoError(t, h.store.GetDB().QueryRow(`SELECT bounce_type, pool_send_id FROM email_bounce WHERE email_lower = 'gone@example.com'`).Scan(&bounceType, &poolSendID))
	assert.Equal(t, "Permanent", bounceType)
	assert.Equal(t, strings.SplitN(hard.MessageID, "@", 2)[0], poolSendID.String)
	ev := h.recorded(events.TopicPoolMessageBounced)
	assert.Equal(t, events.PoolEvent{OrgID: org, Tenant: "acct-1", MailboxID: hard.MailboxID, MessageID: hard.MessageID,
		Email: "gone@example.com", Reason: "550 5.1.1 no such user", Hard: true, Timestamp: ev[0].Timestamp}, ev[0])

	// The address is now on the pool blocklist.
	h.gap()
	_, err = h.send("acct-1", "gone@example.com", "t4")
	assert.Equal(t, http.StatusUnprocessableEntity, httpCode(err))

	// Soft bounce matched by the original Message-ID: recorded, not blocked.
	r, err = h.s.IngestReport(h.ctx, org, []byte(fmt.Sprintf(dsnTemplate, "postmaster@"+bounce, "full@example.com", "delayed", "4.2.2", "452 4.2.2 mailbox full", soft.MessageID)))
	require.NoError(t, err)
	assert.Equal(t, "bounce", r.Kind)
	assert.False(t, r.Hard)
	blocked, err := h.store.IsPoolAddressBlocked(h.ctx, "full@example.com")
	require.NoError(t, err)
	assert.Equal(t, int64(0), blocked)

	// Complaint (ARF).
	r, err = h.s.IngestReport(h.ctx, org, []byte(fmt.Sprintf(arfTemplate, verp(angry), "angry@example.com", angry.MessageID)))
	require.NoError(t, err)
	assert.Equal(t, "complaint", r.Kind)
	require.NoError(t, h.store.GetDB().QueryRow(`SELECT pool_send_id FROM email_complaint WHERE email_lower = 'angry@example.com'`).Scan(&poolSendID))
	assert.Equal(t, strings.SplitN(angry.MessageID, "@", 2)[0], poolSendID.String)
	assert.NotEmpty(t, h.recorded(events.TopicPoolMessageComplained))

	// Another brand's key can't see the send; nonsense isn't a report.
	_, err = h.s.IngestReport(h.ctx, "org-other", []byte(fmt.Sprintf(arfTemplate, verp(angry), "angry@example.com", angry.MessageID)))
	assert.Equal(t, http.StatusNotFound, httpCode(err))
	_, err = h.s.IngestReport(h.ctx, org, []byte("Subject: hi\r\n\r\nhello"))
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestBounceThresholdPausesMailboxThenDomain(t *testing.T) {
	cfg := testConfig()
	cfg.RampStart, cfg.Ceiling = 100, 100
	h := newHarness(t, cfg)
	d, mbs := h.addMailboxes("outreach-a.test", 2, 0)
	// Give the tenant only the first mailbox; the second stays free.
	_, _, err := h.s.Assign(h.ctx, org, "acct-1", 1)
	require.NoError(t, err)
	c, err := h.s.Capacity(h.ctx, org, "acct-1")
	require.NoError(t, err)
	mine := c.Mailboxes[0].ID

	var sends []*SendResult
	for i := 0; i < 20; i++ {
		res, err := h.send("acct-1", fmt.Sprintf("lead%d@example.com", i), fmt.Sprintf("k%d", i))
		require.NoError(t, err)
		sends = append(sends, res)
		h.gap()
	}
	// 1 hard bounce in 20 sends is 5%, past 3%.
	_, err = h.s.IngestReport(h.ctx, org, []byte(fmt.Sprintf(dsnTemplate, verp(sends[0]), "lead0@example.com", "failed", "5.1.1", "550 no such user", "x@y")))
	require.NoError(t, err)

	mb, err := h.store.GetPoolMailbox(h.ctx, mine)
	require.NoError(t, err)
	assert.Equal(t, "paused", mb.Status)
	assert.Equal(t, "list", mb.PausedKind.String)
	assert.Contains(t, mb.PausedReason.String, "hard bounces were 5.0% of its last 20 emails")
	paused := h.recorded(events.TopicPoolMailboxPaused)
	assert.Equal(t, "acct-1", paused[0].Tenant)
	assert.False(t, paused[0].Movable)

	// Its queue stays put: a send to it is refused with 409.
	_, err = h.send("acct-1", "lead1@example.com", "again")
	assert.Equal(t, http.StatusConflict, httpCode(err))

	// One of two mailboxes paused is half: the domain pauses too.
	dom, err := h.store.GetPoolDomain(h.ctx, d.ID)
	require.NoError(t, err)
	assert.Equal(t, "paused", dom.Status)
	assert.Equal(t, "1 of its 2 mailboxes are paused", dom.PausedReason.String)
	other := mbs[0].ID
	if other == mine {
		other = mbs[1].ID
	}
	om, err := h.store.GetPoolMailbox(h.ctx, other)
	require.NoError(t, err)
	assert.Equal(t, "active", om.Status, "the mailbox itself is fine; its domain stops it")

	// An admin resume starts the health window fresh.
	_, err = h.s.SetDomainStatus(h.ctx, d.ID, "active", "")
	require.NoError(t, err)
	_, err = h.s.SetMailboxStatus(h.ctx, mine, "active", "")
	require.NoError(t, err)
	assert.NotEmpty(t, h.recorded(events.TopicPoolMailboxResumed))
	h.gap()
	_, err = h.send("acct-1", "fresh@example.com", "fresh")
	require.NoError(t, err)
	mb, err = h.store.GetPoolMailbox(h.ctx, mine)
	require.NoError(t, err)
	assert.Equal(t, "active", mb.Status)
}

func TestAuthFaultPauseIsMovable(t *testing.T) {
	h := newHarness(t, testConfig())
	h.addMailboxes("outreach-a.test", 2, 10)
	_, _, err := h.s.Assign(h.ctx, org, "acct-1", 2)
	require.NoError(t, err)
	c, err := h.s.Capacity(h.ctx, org, "acct-1")
	require.NoError(t, err)
	target := c.Mailboxes[0].ID
	h.smtp.setAuthFail(true)

	for i := 0; i < 3; i++ {
		res, err := h.s.Send(h.ctx, SendInput{OrgID: org, Tenant: "acct-1", MailboxID: target, To: fmt.Sprintf("l%d@example.com", i),
			Subject: "s", Text: "t", ListUnsubscribeURL: "https://o.test/u", IdempotencyKey: fmt.Sprintf("a%d", i)})
		require.NoError(t, err)
		assert.Equal(t, "failed", res.Status)
		assert.Equal(t, "provider", res.FailureKind)
		assert.Equal(t, 535, res.SMTPCode)
		h.gap()
	}
	c, err = h.s.Capacity(h.ctx, org, "acct-1")
	require.NoError(t, err)
	var got MailboxCapacity
	for _, m := range c.Mailboxes {
		if m.ID == target {
			got = m
		}
	}
	assert.Equal(t, "paused", got.Status)
	assert.Equal(t, "provider", got.PauseKind)
	assert.True(t, got.Movable)
	ev := h.recorded(events.TopicPoolMailboxPaused)
	assert.True(t, ev[0].Movable)
	assert.False(t, ev[0].TenantHeld, "a provider fault never holds the tenant")

	_, err = h.s.Send(h.ctx, SendInput{OrgID: org, Tenant: "acct-1", MailboxID: target, To: "l9@example.com",
		Subject: "s", Text: "t", ListUnsubscribeURL: "https://o.test/u", IdempotencyKey: "a9"})
	assert.Equal(t, http.StatusConflict, httpCode(err))
	assert.Contains(t, err.Error(), "the queue may move")
	// No address was blocked by the provider's refusals.
	blocked, err := h.store.IsPoolAddressBlocked(h.ctx, "l0@example.com")
	require.NoError(t, err)
	assert.Equal(t, int64(0), blocked)
}

func TestRecipientAndPolicyRefusals(t *testing.T) {
	h := newHarness(t, testConfig())
	h.addMailboxes("outreach-a.test", 1, 10)
	_, _, err := h.s.Assign(h.ctx, org, "acct-1", 1)
	require.NoError(t, err)

	h.smtp.reject("nobody@example.com", "550 5.1.1 no such user")
	res, err := h.send("acct-1", "nobody@example.com", "r1")
	require.NoError(t, err)
	assert.Equal(t, "failed", res.Status)
	assert.Equal(t, "recipient", res.FailureKind)
	blocked, err := h.store.IsPoolAddressBlocked(h.ctx, "nobody@example.com")
	require.NoError(t, err)
	assert.Equal(t, int64(1), blocked)

	for i := 0; i < 3; i++ {
		h.gap()
		to := fmt.Sprintf("p%d@example.com", i)
		h.smtp.reject(to, "550 5.7.1 message blocked for sender reputation")
		res, err := h.send("acct-1", to, fmt.Sprintf("p%d", i))
		require.NoError(t, err)
		assert.Equal(t, "policy", res.FailureKind)
	}
	c, err := h.s.Capacity(h.ctx, org, "acct-1")
	require.NoError(t, err)
	assert.Equal(t, "paused", c.Mailboxes[0].Status)
	assert.Equal(t, "policy", c.Mailboxes[0].PauseKind)
	blocked, err = h.store.IsPoolAddressBlocked(h.ctx, "p0@example.com")
	require.NoError(t, err)
	assert.Equal(t, int64(0), blocked, "a policy block is not the address's fault")
}

func TestRepeatPausesHoldTheTenant(t *testing.T) {
	h := newHarness(t, testConfig())
	// Five mailboxes, so two pauses stay under the domain rule.
	_, mbs := h.addMailboxes("outreach-a.test", 5, 10)
	_, _, err := h.s.Assign(h.ctx, org, "acct-1", 5)
	require.NoError(t, err)
	for i := 0; i < 2; i++ {
		mb, err := h.store.GetPoolMailbox(h.ctx, mbs[i].ID)
		require.NoError(t, err)
		require.NoError(t, h.s.pause(h.ctx, mb, "list", "hard bounces", h.clock))
	}
	c, err := h.s.Capacity(h.ctx, org, "acct-1")
	require.NoError(t, err)
	assert.True(t, c.TenantHeld)
	_, err = h.send("acct-1", "lead@example.com", "h1")
	assert.Equal(t, http.StatusConflict, httpCode(err))
	assert.Contains(t, err.Error(), "tenant held")

	n, err := h.s.ClearTenantHold(h.ctx, org, "acct-1")
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	_, err = h.send("acct-1", "lead@example.com", "h1")
	require.NoError(t, err)
}

func TestVerifyWithoutSMTPProbing(t *testing.T) {
	h := newHarness(t, testConfig())
	_, err := h.store.CreateEmailComplaint(h.ctx, db.CreateEmailComplaintParams{ID: "c1", Email: "angry@example.com", EmailForLower: "angry@example.com"})
	require.NoError(t, err)

	got, err := h.s.Verify(h.ctx, []string{"Info@Example.com", "lead@nomx.test", "x@mailinator.com", "not-an-address", "angry@example.com"})
	require.NoError(t, err)
	require.Len(t, got, 5)
	assert.Equal(t, VerifyResult{Email: "info@example.com", Syntax: true, MX: true, Role: true, OK: true}, got[0])
	assert.False(t, got[1].OK)
	assert.True(t, got[1].Syntax)
	assert.False(t, got[1].MX)
	assert.True(t, got[2].Disposable)
	assert.False(t, got[2].OK)
	assert.False(t, got[3].Syntax)
	assert.False(t, got[3].OK)
	assert.True(t, got[4].Blocked)
	assert.False(t, got[4].OK)
	assert.Empty(t, h.smtp.sent())
}

// Pool events go out through Outlet's existing webhook system to the brand
// that holds the tenant.
func TestPausedMailboxReachesTheBrandWebhook(t *testing.T) {
	h := newHarness(t, testConfig())
	got := make(chan webhook.WebhookPayload, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var p webhook.WebhookPayload
		if json.Unmarshal(b, &p) == nil && r.Header.Get("X-Webhook-Event") == p.Event {
			got <- p
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	_, err := h.store.GetDB().Exec(`INSERT INTO organizations (id, name, slug, api_key) VALUES (?, 'NeboAI hub', 'hub', 'k')`, org)
	require.NoError(t, err)
	_, err = h.store.CreateWebhook(h.ctx, db.CreateWebhookParams{ID: "wh1", OrgID: org, Url: srv.URL, Secret: "s",
		Events: events.TopicPoolMailboxPaused + "," + events.TopicPoolMailboxResumed, Active: sql.NullInt64{Int64: 1, Valid: true}})
	require.NoError(t, err)
	d := webhook.NewDispatcher(h.store.Queries, h.bus)
	require.NoError(t, d.Start(h.ctx))
	t.Cleanup(d.Stop)

	h.addMailboxes("outreach-a.test", 3, 10)
	_, _, err = h.s.Assign(h.ctx, org, "acct-1", 1)
	require.NoError(t, err)
	c, err := h.s.Capacity(h.ctx, org, "acct-1")
	require.NoError(t, err)
	_, err = h.s.SetMailboxStatus(h.ctx, c.Mailboxes[0].ID, "paused", "provider account review")
	require.NoError(t, err)

	select {
	case p := <-got:
		assert.Equal(t, events.TopicPoolMailboxPaused, p.Event)
		assert.Equal(t, "acct-1", p.Data["tenant"])
		assert.Equal(t, c.Mailboxes[0].ID, p.Data["mailbox_id"])
		assert.Equal(t, "provider account review", p.Data["reason"])
		assert.Equal(t, "admin", p.Data["pause_kind"])
	case <-time.After(5 * time.Second):
		t.Fatal("no webhook delivered")
	}
}
