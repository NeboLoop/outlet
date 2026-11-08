package outbound

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/services/crypto"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// ---- a local mail server: SMTP that records, IMAP that serves inboxes -----

type sentMail struct {
	From, To string
	Data     []byte
}

type mailServer struct {
	mu        sync.Mutex
	sent      []sentMail
	passwords map[string]string
	imapUsers map[string]*imapmemserver.User
	smtpPort  int
	imapPort  int
}

type smtpSession struct {
	srv        *mailServer
	from       string
	to         []string
	authorized bool
}

func (s *smtpSession) AuthMechanisms() []string { return []string{sasl.Plain} }
func (s *smtpSession) Auth(string) (sasl.Server, error) {
	return sasl.NewPlainServer(func(_, user, pass string) error {
		if s.srv.passwords[user] == "" || s.srv.passwords[user] != pass {
			return errors.New("bad login")
		}
		s.authorized = true
		return nil
	}), nil
}
func (s *smtpSession) Mail(from string, _ *smtp.MailOptions) error {
	if !s.authorized {
		return smtp.ErrAuthRequired
	}
	s.from = from
	return nil
}
func (s *smtpSession) Rcpt(to string, _ *smtp.RcptOptions) error { s.to = append(s.to, to); return nil }
func (s *smtpSession) Data(r io.Reader) error {
	b, _ := io.ReadAll(r)
	s.srv.mu.Lock()
	defer s.srv.mu.Unlock()
	for _, to := range s.to {
		s.srv.sent = append(s.srv.sent, sentMail{From: s.from, To: to, Data: b})
	}
	return nil
}
func (s *smtpSession) Reset()        { s.from, s.to = "", nil }
func (s *smtpSession) Logout() error { return nil }

func startMailServer(t *testing.T, users map[string]string) *mailServer {
	t.Helper()
	ms := &mailServer{passwords: users, imapUsers: map[string]*imapmemserver.User{}}

	be := smtp.BackendFunc(func(*smtp.Conn) (smtp.Session, error) { return &smtpSession{srv: ms}, nil })
	ss := smtp.NewServer(be)
	ss.AllowInsecureAuth = true
	ss.Domain = "localhost"
	sl, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go ss.Serve(sl)
	t.Cleanup(func() { ss.Close() })
	ms.smtpPort = sl.Addr().(*net.TCPAddr).Port

	mem := imapmemserver.New()
	for u, p := range users {
		user := imapmemserver.NewUser(u, p)
		require.NoError(t, user.Create("INBOX", nil))
		mem.AddUser(user)
		ms.imapUsers[u] = user
	}
	is := imapserver.New(&imapserver.Options{
		NewSession: func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return mem.NewSession(), nil, nil
		},
		Caps:         imap.CapSet{imap.CapIMAP4rev1: {}},
		InsecureAuth: true,
	})
	il, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go is.Serve(il)
	t.Cleanup(func() { is.Close() })
	ms.imapPort = il.Addr().(*net.TCPAddr).Port
	return ms
}

type literal struct{ *bytes.Reader }

func (l literal) Size() int64 { return int64(l.Reader.Len()) }

// deliver puts a message in a mailbox's INBOX, as if someone wrote to it.
func (ms *mailServer) deliver(t *testing.T, user, raw string) {
	t.Helper()
	raw = strings.ReplaceAll(raw, "\n", "\r\n")
	_, err := ms.imapUsers[user].Append("INBOX", literal{bytes.NewReader([]byte(raw))}, &imap.AppendOptions{})
	require.NoError(t, err)
}

func (ms *mailServer) sentTo(to string) []*mail.Message {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	var out []*mail.Message
	for _, s := range ms.sent {
		if s.To == to {
			m, _ := mail.ReadMessage(bytes.NewReader(s.Data))
			out = append(out, m)
		}
	}
	return out
}

func (ms *mailServer) count() int {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return len(ms.sent)
}

func bodyOf(t *testing.T, m *mail.Message) string {
	b, err := io.ReadAll(m.Body)
	require.NoError(t, err)
	return strings.ReplaceAll(strings.ReplaceAll(string(b), "=\r\n", ""), "\r\n", "\n")
}

// ---- fake DNS -------------------------------------------------------------

type fakeDNS struct{ ok map[string]bool }

func (f fakeDNS) LookupMX(_ context.Context, name string) ([]*net.MX, error) {
	if f.ok[name] {
		return []*net.MX{{Host: "smtp.google.com.", Pref: 1}}, nil
	}
	return nil, errors.New("no MX")
}

func (f fakeDNS) LookupTXT(_ context.Context, name string) ([]string, error) {
	switch {
	case strings.HasPrefix(name, "_dmarc.") && f.ok[strings.TrimPrefix(name, "_dmarc.")]:
		return []string{"v=DMARC1; p=none"}, nil
	case strings.Contains(name, "._domainkey.") && f.ok[name[strings.Index(name, "._domainkey.")+len("._domainkey."):]]:
		return []string{"v=DKIM1; k=rsa; p=abc"}, nil
	case f.ok[name]:
		return []string{"v=spf1 include:_spf.google.com ~all"}, nil
	}
	return nil, errors.New("no TXT")
}

// ---- the harness ----------------------------------------------------------

type harness struct {
	t     *testing.T
	ctx   context.Context
	conn  *sql.DB
	store *db.Store
	svc   *Service
	org   db.Organization
	mail  *mailServer
	clock time.Time
	dns   fakeDNS
}

const testKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func newHarness(t *testing.T) *harness {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn))
	store := db.NewStore(conn)
	ctx := context.Background()
	org, err := store.CreateOrganization(ctx, db.CreateOrganizationParams{ID: "org-1", Name: "Nebo Outreach", Slug: "nebo-outreach", ApiKey: "key-1"})
	require.NoError(t, err)
	_, err = store.CreateOrganization(ctx, db.CreateOrganizationParams{ID: "org-2", Name: "Other", Slug: "other", ApiKey: "key-2"})
	require.NoError(t, err)
	cs, err := crypto.NewService(testKey)
	require.NoError(t, err)

	h := &harness{t: t, ctx: ctx, conn: conn, store: store, org: org}
	// Tuesday, 16:00 UTC: inside the 8 to 17 window.
	h.clock = time.Date(2026, 11, 10, 16, 0, 0, 0, time.UTC)
	h.dns = fakeDNS{ok: map[string]bool{"getexample.com": true}}
	h.svc = New(store, cs, Config{Brands: []string{"nebo-outreach"}, ProtectedDomains: []string{"neboai.com", "neboloop.com"}})
	h.svc.now = func() time.Time { return h.clock }
	h.svc.dns = h.dns
	h.svc.verify = func(_ context.Context, email string) (bool, string) {
		if strings.Contains(email, "risky") {
			return false, "no MX records"
		}
		return true, "ok"
	}
	h.svc.rnd = rand.New(rand.NewPCG(1, 2))
	h.mail = startMailServer(t, map[string]string{"sam@getexample.com": "pw-sam", "pat@getexample.com": "pw-pat"})
	return h
}

func (h *harness) advance(d time.Duration) { h.clock = h.clock.Add(d) }

func (h *harness) mailbox(email, pw string) MailboxInput {
	return MailboxInput{Email: email, FromName: strings.Split(email, "@")[0] + " Rivera", Password: pw,
		SMTPHost: "127.0.0.1", SMTPPort: h.mail.smtpPort, IMAPHost: "127.0.0.1", IMAPPort: h.mail.imapPort}
}

// setup gives the brand settings, a domain and two mailboxes.
func (h *harness) setup() (sam, pat db.OutboundMailbox) {
	t := h.t
	_, err := h.svc.UpdateSettings(h.ctx, h.org, SettingsInput{MailingAddress: "NeboAI, 123 Main St, Provo, UT 84601", Timezone: "UTC"})
	require.NoError(t, err)
	d, pending, check, err := h.svc.AddDomain(h.ctx, h.org, DomainInput{Domain: "getexample.com"})
	require.NoError(t, err)
	assert.True(t, check.OK())
	assert.Equal(t, "ok", d.Status)
	assert.NotEmpty(t, pending, "a manual domain returns the records to paste")
	res, err := h.svc.AddMailboxes(h.ctx, h.org, []MailboxInput{h.mailbox("sam@getexample.com", "pw-sam"), h.mailbox("pat@getexample.com", "pw-pat")})
	require.NoError(t, err)
	for _, r := range res {
		require.Empty(t, r.Error)
	}
	sam, err = h.store.GetOutboundMailboxByEmail(h.ctx, "sam@getexample.com")
	require.NoError(t, err)
	pat, err = h.store.GetOutboundMailboxByEmail(h.ctx, "pat@getexample.com")
	require.NoError(t, err)
	// The reply reader's first pass sets where reading starts.
	h.svc.PollReplies(h.ctx)
	return sam, pat
}

func (h *harness) prospects(in ...ProspectInput) map[string]ProspectResult {
	res, err := h.svc.AddProspects(h.ctx, h.org, in)
	require.NoError(h.t, err)
	out := map[string]ProspectResult{}
	for _, r := range res {
		if _, seen := out[r.Email]; !seen { // a duplicate later in the batch doesn't hide the first
			out[r.Email] = r
		}
	}
	return out
}

func creator(email, first, video string) ProspectInput {
	return ProspectInput{Email: email, FirstName: first, Platform: "YouTube", ChannelURL: "https://youtube.com/@" + first,
		RecentVideo: video, EmailSource: "YouTube About page", Timezone: "UTC"}
}

var threeSteps = []StepInput{
	{Subject: "{Quick question|A question}, {{first_name}}", Body: "Hi {{first_name}},\n\nLoved {{recent_video}}. Want to try Nebo?\n\n{{sender_first}}"},
	{DelayDays: 2, Body: "Following up on this, {{first_name}}."},
	{DelayDays: 3, Body: "Last note from me, {{first_name}}."},
}

// ---- tests ----------------------------------------------------------------

// The whole loop against a real SMTP and IMAP server: domains, mailboxes,
// prospects, drafts, approval, paced sending, reply, bounce and suppression.
func TestOutreachEndToEnd(t *testing.T) {
	h := newHarness(t)
	ctx := h.ctx

	// Protected and the brand's own domains are refused.
	_, _, _, err := h.svc.AddDomain(ctx, h.org, DomainInput{Domain: "neboai.com"})
	assert.ErrorContains(t, err, "protected")
	_, _, _, err = h.svc.AddDomain(ctx, h.org, DomainInput{Domain: "mail.neboloop.com"})
	assert.ErrorContains(t, err, "protected")

	// Campaigns need a mailing address.
	_, err = h.svc.DraftOutreach(ctx, h.org, CampaignInput{Name: "x", Steps: threeSteps, ProspectIDs: []string{"p"}})
	assert.ErrorContains(t, err, "mailing address")

	// A mailbox that can't log in is reported and not saved.
	_, _, _, err = h.svc.AddDomain(ctx, h.org, DomainInput{Domain: "getexample.com"})
	require.NoError(t, err)
	res, err := h.svc.AddMailboxes(ctx, h.org, []MailboxInput{h.mailbox("sam@getexample.com", "wrong")})
	require.NoError(t, err)
	assert.Contains(t, res[0].Error, "SMTP login failed")
	_, err = h.store.GetOutboundMailboxByEmail(ctx, "sam@getexample.com")
	assert.ErrorIs(t, err, sql.ErrNoRows)

	sam, pat := h.setup()

	// Suppressed, unsourced, risky and duplicate prospects are sorted out.
	h.svc.suppress(ctx, h.org.ID, "dana@example.org", "earlier opt-out")
	got := h.prospects(
		creator("ava@example.org", "Ava", "your desk tour"),
		creator("ben@example.org", "Ben", "the camera review"),
		creator("cal@example.org", "Cal", ""),
		creator("dana@example.org", "Dana", "x"),
		ProspectInput{Email: "eve@example.org", FirstName: "Eve"},
		creator("risky@example.org", "Rae", "x"),
		creator("ava@example.org", "Ava", "again"),
	)
	assert.Equal(t, "added", got["ava@example.org"].Status)
	assert.Equal(t, "added", got["ben@example.org"].Status)
	assert.Equal(t, "suppressed", got["dana@example.org"].Status)
	assert.Equal(t, "refused", got["eve@example.org"].Status)
	assert.Equal(t, "held", got["risky@example.org"].Status)
	ids := []string{got["ava@example.org"].ID, got["ben@example.org"].ID, got["cal@example.org"].ID, got["risky@example.org"].ID}

	// Dry run renders without saving.
	dry, err := h.svc.DraftOutreach(ctx, h.org, CampaignInput{Name: "Creators", Steps: threeSteps, ProspectIDs: ids, DryRun: true})
	require.NoError(t, err)
	assert.Equal(t, 2, dry.Enrolled)
	assert.Contains(t, dry.Skipped[got["cal@example.org"].ID], "recent_video", "a missing tag skips, never sends a blank")
	assert.Contains(t, dry.Skipped[got["risky@example.org"].ID], "held")
	camps, _ := h.store.ListOutboundCampaigns(ctx, h.org.ID)
	assert.Empty(t, camps)

	camp, err := h.svc.DraftOutreach(ctx, h.org, CampaignInput{Name: "Creators", Steps: threeSteps, ProspectIDs: ids, IdempotencyKey: "k1"})
	require.NoError(t, err)
	assert.Equal(t, 2, camp.Enrolled)
	again, err := h.svc.DraftOutreach(ctx, h.org, CampaignInput{Name: "Creators", Steps: threeSteps, ProspectIDs: ids, IdempotencyKey: "k1"})
	require.NoError(t, err)
	assert.Equal(t, camp.CampaignID, again.CampaignID, "a repeated call with the same key creates nothing")

	// Drafts wait for approval.
	h.svc.Tick(ctx)
	assert.Equal(t, 0, h.mail.count())
	var draftIDs []string
	for _, d := range camp.Drafts {
		draftIDs = append(draftIDs, d.MessageID)
		assert.NotContains(t, d.Body, "{{")
		assert.Contains(t, d.Body, "123 Main St")
	}
	rv, err := h.svc.Review(ctx, h.org, "approve", draftIDs)
	require.NoError(t, err)
	assert.Equal(t, 2, rv.Done)

	// One send per mailbox per pass: the two prospects went to different mailboxes.
	h.svc.Tick(ctx)
	require.Equal(t, 2, h.mail.count())
	ava := h.mail.sentTo("ava@example.org")
	require.Len(t, ava, 1)
	assert.Contains(t, ava[0].Header.Get("Subject"), "Ava")
	assert.Contains(t, ava[0].Header.Get("List-Unsubscribe"), "mailto:")
	assert.Empty(t, ava[0].Header.Get("In-Reply-To"))
	body := bodyOf(t, ava[0])
	assert.Contains(t, body, "Loved your desk tour")
	assert.Contains(t, body, "reply \"no thanks\"")
	avaMsgID := strings.Trim(ava[0].Header.Get("Message-Id"), "<>")
	avaFrom := ava[0].Header.Get("From")
	ben := h.mail.sentTo("ben@example.org")
	require.Len(t, ben, 1)
	assert.NotEqual(t, avaFrom, ben[0].Header.Get("From"), "prospects rotate between mailboxes")

	// The gap: nothing more right away.
	h.svc.Tick(ctx)
	assert.Equal(t, 2, h.mail.count())

	// Ava replies in the thread: her sequence stops.
	avaBox := "sam@getexample.com"
	if strings.Contains(avaFrom, "pat@") {
		avaBox = "pat@getexample.com"
	}
	benBox := map[string]string{"sam@getexample.com": "pat@getexample.com", "pat@getexample.com": "sam@getexample.com"}[avaBox]
	h.mail.deliver(t, avaBox, "From: Ava <ava@example.org>\nTo: "+avaBox+"\nSubject: Re: hi\nMessage-ID: <reply-1@example.org>\nIn-Reply-To: <"+avaMsgID+">\n\nSounds fun, tell me more!\n\nOn Tue, Sam wrote:\n> Hi Ava")
	// Ben's address bounces.
	h.mail.deliver(t, benBox, "From: Mail Delivery Subsystem <mailer-daemon@googlemail.com>\nTo: "+benBox+"\nSubject: Delivery Status Notification (Failure)\nMessage-ID: <dsn-1@google.com>\nMIME-Version: 1.0\nContent-Type: multipart/report; report-type=delivery-status; boundary=\"b1\"\n\n--b1\nContent-Type: text/plain\n\nAddress not found.\n--b1\nContent-Type: message/delivery-status\n\nReporting-MTA: dns; googlemail.com\n\nFinal-Recipient: rfc822; ben@example.org\nAction: failed\nStatus: 5.1.1\n--b1--\n")
	h.svc.PollReplies(ctx)

	replies, err := h.svc.ListReplies(ctx, h.org, false, 10)
	require.NoError(t, err)
	require.Len(t, replies, 1)
	assert.Equal(t, "needs_review", replies[0].Classification)
	assert.Equal(t, "Sounds fun, tell me more!", replies[0].Body)
	avaP, _ := h.store.GetOutboundProspectByEmail(ctx, db.GetOutboundProspectByEmailParams{OrgID: h.org.ID, Email: "ava@example.org"})
	assert.Equal(t, "replied", avaP.Status)
	benP, _ := h.store.GetOutboundProspectByEmail(ctx, db.GetOutboundProspectByEmailParams{OrgID: h.org.ID, Email: "ben@example.org"})
	assert.Equal(t, "bounced", benP.Status)
	assert.True(t, h.svc.suppressed(ctx, h.org.ID, "ben@example.org"), "a bounce goes on the suppression list")

	// Reading again changes nothing.
	h.svc.PollReplies(ctx)
	replies, _ = h.svc.ListReplies(ctx, h.org, false, 10)
	assert.Len(t, replies, 1)

	// Two business days later neither gets a follow-up: one replied, one bounced.
	h.advance(48 * time.Hour)
	h.svc.Tick(ctx)
	assert.Equal(t, 2, h.mail.count())

	// Answering Ava goes from the same mailbox, in the thread.
	_, err = h.svc.Answer(ctx, h.org, replies[0].ID, "Here's a link to try it.")
	require.NoError(t, err)
	h.advance(10 * time.Minute)
	h.svc.Tick(ctx)
	ava = h.mail.sentTo("ava@example.org")
	require.Len(t, ava, 2)
	assert.Equal(t, avaFrom, ava[1].Header.Get("From"))
	assert.Equal(t, "<reply-1@example.org>", ava[1].Header.Get("In-Reply-To"))

	_ = sam
	_ = pat
}

// A prospect who doesn't answer gets the follow-ups, in the same thread from
// the same mailbox, on business days; the sequence then ends.
func TestFollowUpsThreadFromTheSameMailbox(t *testing.T) {
	h := newHarness(t)
	ctx := h.ctx
	h.setup()
	got := h.prospects(creator("fay@example.org", "Fay", "the studio vlog"))
	camp, err := h.svc.DraftOutreach(ctx, h.org, CampaignInput{Name: "C", Steps: threeSteps[:2], ProspectIDs: []string{got["fay@example.org"].ID}, ApprovalMode: "first"})
	require.NoError(t, err)
	_, err = h.svc.Review(ctx, h.org, "approve", []string{camp.Drafts[0].MessageID})
	require.NoError(t, err)
	h.svc.Tick(ctx)
	first := h.mail.sentTo("fay@example.org")
	require.Len(t, first, 1)

	// One business day later: not yet.
	h.advance(24 * time.Hour)
	h.svc.Tick(ctx)
	assert.Len(t, h.mail.sentTo("fay@example.org"), 1)

	// Two business days later the follow-up is drafted, approved (first-touch
	// mode) and sent in the same pass.
	h.advance(24*time.Hour + time.Minute)
	h.svc.Tick(ctx)
	all := h.mail.sentTo("fay@example.org")
	require.Len(t, all, 2)
	assert.Equal(t, first[0].Header.Get("From"), all[1].Header.Get("From"))
	assert.Equal(t, first[0].Header.Get("Message-Id"), all[1].Header.Get("In-Reply-To"))
	assert.Equal(t, "Re: "+first[0].Header.Get("Subject"), all[1].Header.Get("Subject"))
	assert.Contains(t, bodyOf(t, all[1]), "Following up on this, Fay.")

	e, err := h.store.ListActiveOutboundEnrollmentsForProspect(ctx, got["fay@example.org"].ID)
	require.NoError(t, err)
	assert.Empty(t, e, "the sequence ended after its last step")
}

// The limits: warm-up allowance, opt-outs from a reply, health pauses,
// DNS pauses, crash recovery, and outbound off for brands not in config.
func TestSafetyLimits(t *testing.T) {
	h := newHarness(t)
	ctx := h.ctx
	sam, _ := h.setup()

	mb := db.OutboundMailbox{DailyCap: 30, WarmupStartedAt: ts(h.clock)}
	assert.EqualValues(t, 5, allowance(mb, h.clock))
	assert.EqualValues(t, 7, allowance(mb, h.clock.Add(24*time.Hour)))
	assert.EqualValues(t, 30, allowance(mb, h.clock.Add(40*24*time.Hour)))
	mb.DailyCap = 500
	assert.EqualValues(t, 30, allowance(mb, h.clock.Add(40*24*time.Hour)), "no stored cap gets past 30")

	// Not enabled in config: refused.
	other, _ := h.store.GetOrganizationByID(ctx, "org-2")
	_, err := h.svc.DraftOutreach(ctx, other, CampaignInput{Name: "x", Steps: threeSteps, ProspectIDs: []string{"p"}})
	assert.ErrorContains(t, err, "OUTBOUND_BRANDS")

	// Automatic approval needs 30 days of mailbox history.
	got := h.prospects(creator("gus@example.org", "Gus", "v"))
	_, err = h.svc.DraftOutreach(ctx, h.org, CampaignInput{Name: "x", Steps: threeSteps, ProspectIDs: []string{got["gus@example.org"].ID}, ApprovalMode: "automatic"})
	assert.ErrorContains(t, err, "30 days")

	// An opt-out reply suppresses the address.
	camp, err := h.svc.DraftOutreach(ctx, h.org, CampaignInput{Name: "x", Steps: threeSteps, ProspectIDs: []string{got["gus@example.org"].ID}, MailboxIDs: []string{sam.ID}})
	require.NoError(t, err)
	_, _ = h.svc.Review(ctx, h.org, "approve", []string{camp.Drafts[0].MessageID})
	h.svc.Tick(ctx)
	sent := h.mail.sentTo("gus@example.org")
	require.Len(t, sent, 1)
	h.mail.deliver(t, "sam@getexample.com", "From: gus@example.org\nTo: sam@getexample.com\nSubject: Re: x\nMessage-ID: <r-gus@example.org>\nIn-Reply-To: "+sent[0].Header.Get("Message-Id")+"\n\nNo thanks.\n")
	h.svc.PollReplies(ctx)
	assert.True(t, h.svc.suppressed(ctx, h.org.ID, "gus@example.org"))

	// Health: bounces over 3% pause the mailbox.
	for i, id := range []string{"b1@x", "b2@x"} {
		_, err := h.store.CreateOutboundReply(ctx, db.CreateOutboundReplyParams{ID: id, OrgID: h.org.ID, MailboxID: sam.ID, FromEmail: "mailer-daemon@x", MessageID: id, Classification: "bounce", Handled: 1})
		require.NoError(t, err, i)
	}
	h.svc.Tick(ctx)
	sam, _ = h.store.GetOutboundMailboxByEmail(ctx, "sam@getexample.com")
	assert.Equal(t, "paused", sam.Status)
	assert.Contains(t, sam.PausedReason, "bounce rate")
	err = h.svc.Pause(ctx, h.org, "mailbox", sam.ID, true)
	assert.ErrorContains(t, err, "still over a health limit")
	why, err := h.svc.Explain(ctx, h.org, "", sam.ID, "")
	require.NoError(t, err)
	assert.Contains(t, why, "reputation")

	// Failing DNS pauses every mailbox on the domain; fixing it resumes them.
	d, _ := h.store.GetOutboundDomainByName(ctx, "getexample.com")
	delete(h.dns.ok, "getexample.com")
	c, err := h.svc.CheckDomain(ctx, d)
	require.NoError(t, err)
	assert.False(t, c.OK())
	pat, _ := h.store.GetOutboundMailboxByEmail(ctx, "pat@getexample.com")
	assert.Equal(t, "paused", pat.Status)
	assert.Contains(t, pat.PausedReason, "DNS:")
	h.dns.ok["getexample.com"] = true
	_, err = h.svc.CheckDomain(ctx, d)
	require.NoError(t, err)
	pat, _ = h.store.GetOutboundMailboxByEmail(ctx, "pat@getexample.com")
	assert.Equal(t, "active", pat.Status)

	// A send interrupted mid-way is never sent again.
	stuck, err := h.store.CreateOutboundMessage(ctx, db.CreateOutboundMessageParams{ID: "stuck-1", OrgID: h.org.ID, MailboxID: pat.ID, ProspectID: got["gus@example.org"].ID, ToEmail: "gus@example.org", Subject: "s", Body: "b", Status: "approved", MessageID: "m@x", NotBefore: ts(h.clock)})
	require.NoError(t, err)
	n, err := h.store.ClaimOutboundMessage(ctx, db.ClaimOutboundMessageParams{ClaimedAt: sql.NullString{String: ts(h.clock), Valid: true}, ID: stuck.ID})
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
	h.advance(20 * time.Minute)
	before := h.mail.count()
	h.svc.Tick(ctx)
	m, _ := h.store.GetOutboundMessage(ctx, db.GetOutboundMessageParams{ID: stuck.ID, OrgID: h.org.ID})
	assert.Equal(t, "failed", m.Status)
	assert.Equal(t, before, h.mail.count())
}

func TestRenderAndWindows(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 1))
	out, err := render("{Hi|Hey} {{first_name}}, {{ niche }}", map[string]string{"first_name": "Ava", "niche": "tech"}, r)
	require.NoError(t, err)
	assert.Regexp(t, `^(Hi|Hey) Ava, tech$`, out)
	_, err = render("Hi {{first_name}} {{recent_video}}", map[string]string{"first_name": "Ava"}, r)
	assert.EqualError(t, err, "no value for recent_video")

	// Saturday evening → Monday 8:00.
	sat := time.Date(2026, 11, 14, 19, 0, 0, 0, time.UTC)
	assert.Equal(t, time.Date(2026, 11, 16, 8, 0, 0, 0, time.UTC), nextWindow(sat, time.UTC, 8, 17))
	// Tuesday 6:00 → Tuesday 8:00; Tuesday 10:00 stays.
	tue := time.Date(2026, 11, 10, 6, 0, 0, 0, time.UTC)
	assert.Equal(t, time.Date(2026, 11, 10, 8, 0, 0, 0, time.UTC), nextWindow(tue, time.UTC, 8, 17))
	assert.Equal(t, tue.Add(4*time.Hour), nextWindow(tue.Add(4*time.Hour), time.UTC, 8, 17))
	// Friday + 1 business day = Monday.
	fri := time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)
	assert.Equal(t, time.Monday, addBusinessDays(fri, 1).Weekday())
}

func TestClassify(t *testing.T) {
	mb := db.OutboundMailbox{Email: "sam@getexample.com"}
	parse := func(raw string) incoming {
		in, err := parseIncoming([]byte(strings.ReplaceAll(raw, "\n", "\r\n")))
		require.NoError(t, err)
		return in
	}
	assert.Equal(t, "out_of_office", classify(mb, parse("From: a@x.org\nSubject: Automatic reply: hi\nAuto-Submitted: auto-replied\n\nAway until Monday.")))
	assert.Equal(t, "unsubscribe", classify(mb, parse("From: a@x.org\nSubject: unsubscribe\n\n")))
	assert.Equal(t, "unsubscribe", classify(mb, parse("From: a@x.org\nSubject: Re: hi\n\nPlease remove me from your list.\n\nOn Tue, Sam wrote:\n> no thanks? reply")))
	assert.Equal(t, "not_interested", classify(mb, parse("From: a@x.org\nSubject: Re: hi\n\nNot interested, sorry.")))
	assert.Equal(t, "complaint", classify(mb, parse("From: a@x.org\nSubject: Re: hi\n\nThis is spam. How did you get my email?")))
	assert.Equal(t, "needs_review", classify(mb, parse("From: a@x.org\nSubject: Re: hi\n\nWhat's the commission?\n\nOn Tue, Sam wrote:\n> reply \"no thanks\" and I won't email again")))
	assert.Equal(t, "bounce", classify(mb, parse("From: MAILER-DAEMON@x.org\nSubject: Undelivered\n\nfailed")))
}

// The operations behind the tools: overview, edit, classify, pause, explain.
func TestOperations(t *testing.T) {
	h := newHarness(t)
	ctx := h.ctx
	sam, _ := h.setup()
	got := h.prospects(creator("ivy@example.org", "Ivy", "the build vlog"))
	camp, err := h.svc.DraftOutreach(ctx, h.org, CampaignInput{Name: "C", Steps: threeSteps, ProspectIDs: []string{got["ivy@example.org"].ID}, MailboxIDs: []string{sam.ID}})
	require.NoError(t, err)
	draft := camp.Drafts[0].MessageID

	o, err := h.svc.Overview(ctx, h.org)
	require.NoError(t, err)
	assert.True(t, o.Enabled)
	assert.EqualValues(t, 1, o.Messages["draft"])
	assert.Len(t, o.Mailboxes, 2)
	assert.Contains(t, strings.Join(o.Next, " "), "drafts wait")

	why, err := h.svc.Explain(ctx, h.org, draft, "", "")
	require.NoError(t, err)
	assert.Contains(t, why, "waiting for approval")

	require.NoError(t, h.svc.EditDraft(ctx, h.org, draft, "New subject", ""))
	m, _ := h.store.GetOutboundMessage(ctx, db.GetOutboundMessageParams{ID: draft, OrgID: h.org.ID})
	assert.Equal(t, "New subject", m.Subject)

	// Paused campaign: approved email waits.
	require.NoError(t, h.svc.Pause(ctx, h.org, "campaign", camp.CampaignID, false))
	_, err = h.svc.Review(ctx, h.org, "approve", []string{draft})
	require.NoError(t, err)
	h.svc.Tick(ctx)
	assert.Equal(t, 0, h.mail.count())
	require.NoError(t, h.svc.Pause(ctx, h.org, "campaign", camp.CampaignID, true))

	// Everything paused: still nothing.
	require.NoError(t, h.svc.Pause(ctx, h.org, "all", "", false))
	h.svc.Tick(ctx)
	assert.Equal(t, 0, h.mail.count())
	require.NoError(t, h.svc.Pause(ctx, h.org, "all", "", true))
	h.svc.Tick(ctx)
	require.Equal(t, 1, h.mail.count())
	sent := h.mail.sentTo("ivy@example.org")
	assert.Equal(t, "New subject", sent[0].Header.Get("Subject"))

	// A reply the rules can't settle is classified by the caller.
	h.mail.deliver(t, "sam@getexample.com", "From: ivy@example.org\nTo: sam@getexample.com\nSubject: Re: New subject\nMessage-ID: <r-ivy@example.org>\nIn-Reply-To: "+sent[0].Header.Get("Message-Id")+"\n\nWhat does it pay?\n")
	h.svc.PollReplies(ctx)
	replies, _ := h.svc.ListReplies(ctx, h.org, false, 10)
	require.Len(t, replies, 1)
	require.NoError(t, h.svc.Classify(ctx, h.org, replies[0].ID, "interested"))
	p, _ := h.store.GetOutboundProspect(ctx, db.GetOutboundProspectParams{ID: got["ivy@example.org"].ID, OrgID: h.org.ID})
	assert.Equal(t, "interested", p.Status)
	require.NoError(t, h.svc.MarkPartner(ctx, h.org, p.ID))
	assert.ErrorContains(t, h.svc.Classify(ctx, h.org, replies[0].ID, "maybe"), "must be one of")
}

// DigitalOcean: missing records are added, an existing SPF is never doubled.
func TestDigitalOceanRecords(t *testing.T) {
	h := newHarness(t)
	var mu sync.Mutex
	var posted []doRecord
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer do-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"domain_records":[{"type":"TXT","name":"@","data":"v=spf1 include:_spf.google.com ~all"}]}`))
			return
		}
		var rec doRecord
		_ = json.NewDecoder(r.Body).Decode(&rec)
		mu.Lock()
		posted = append(posted, rec)
		mu.Unlock()
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer api.Close()
	h.svc.doAPI = api.URL

	_, _, _, err := h.svc.AddDomain(h.ctx, h.org, DomainInput{Domain: "getexample.com", DNSHost: "digitalocean"})
	assert.ErrorContains(t, err, "No DigitalOcean token")
	require.NoError(t, h.svc.SetDNSToken(h.ctx, h.org.ID, "do-token"))
	_, pending, _, err := h.svc.AddDomain(h.ctx, h.org, DomainInput{Domain: "getexample.com", DNSHost: "digitalocean", DKIMValue: "v=DKIM1; k=rsa; p=abc"})
	require.NoError(t, err)
	assert.Empty(t, pending)
	var kinds []string
	for _, r := range posted {
		kinds = append(kinds, r.Type+" "+r.Name)
	}
	assert.ElementsMatch(t, []string{"MX @", "TXT _dmarc", "TXT google._domainkey"}, kinds, "SPF already existed and is not doubled")
}
