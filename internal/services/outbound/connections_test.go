package outbound

import (
	"database/sql"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/services/outbound/providers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mixedPool gives the brand one mailbox pool with three connections side by
// side, each with one domain and one mailbox that logs in to the harness's
// real SMTP and IMAP server:
//   - Maildoso: getexample.com, sam (bought through a confirmed quote);
//   - CheapInboxes: getother.com, kim (connected, then imported by a sync);
//   - manual: getthird.com, lee (added by hand).
type mixedPool struct {
	maildoso, cheap, manual Connection
	sam, kim, lee           db.Mailbox
}

func (h *harness) mixedPool() mixedPool {
	t, ctx := h.t, h.ctx
	h.dns.ok["getother.com"], h.dns.ok["getthird.com"] = true, true
	h.mail.addUser(t, "kim@getother.com", "pw-kim")
	h.mail.addUser(t, "lee@getthird.com", "pw-lee")
	smtpHost, imapHost := harnessHost(h)

	var mu sync.Mutex
	mdAccounts := []map[string]any{}
	md := newFakeService(t, "Bearer md-key", func(c serviceCall) (int, any) {
		mu.Lock()
		defer mu.Unlock()
		switch c.Method + " " + c.Path {
		case "GET /v1/user/domains":
			return 200, []any{}
		case "POST /v1/user/domains/external":
			return 201, map[string]any{"results": []map[string]any{{"name": "getexample.com", "result": "success", "message": nil, "id": 7}}, "task_id": "t-1"}
		case "POST /v1/user/accounts":
			var body []map[string]any
			require.NoError(t, json.Unmarshal([]byte(c.Body), &body))
			res := []map[string]any{}
			for i, b := range body {
				a := map[string]any{"id": 100 + i, "user_id": 1, "email_account": b["email_account"], "is_active": true, "domain_id": 7, "forwarding_account_id": nil,
					"password": "pw-sam", "first_name": b["first_name"], "last_name": "Rivera", "created_at": "2026-11-10T16:00:00",
					"imap": imapHost, "smtp": smtpHost, "forwarding_account": nil, "provider": "maildoso", "status": "active", "is_admin": false, "exported_to_sequencers": []any{}}
				mdAccounts = append(mdAccounts, a)
				res = append(res, a)
			}
			return 202, map[string]any{"results": res, "task_id": "t-2"}
		case "GET /v1/user/accounts-lookup":
			return 200, map[string]any{"items": mdAccounts, "meta": map[string]any{"offset": 0, "limit": nil, "total": len(mdAccounts), "current": len(mdAccounts)}}
		}
		return 0, nil
	})
	ci := newFakeService(t, "Bearer ci_live_mixed", func(c serviceCall) (int, any) {
		switch {
		case c.Method == "GET" && c.Path == "/v1/domains":
			return 200, map[string]any{"domains": []any{}, "pagination": map[string]int{"total": 0, "limit": 100, "offset": 0}}
		case c.Method == "POST" && c.Path == "/v1/domain-imports/bulk":
			return 200, map[string]any{"entries": []any{map[string]any{"id": "6f1c1e8e-0000-4000-8000-000000000009", "domain_name_normalized": "getother.com", "state": "staged", "source_provider": "manual"}}, "errors": []any{}}
		case c.Method == "GET" && c.Path == "/v1/mailboxes":
			return 200, map[string]any{"mailboxes": []any{map[string]any{"id": "mb-kim", "full_email": "kim@getother.com", "first_name": "Kim", "last_name": "Lee", "status": "active"}},
				"pagination": map[string]int{"total": 1, "limit": 100, "offset": 0}}
		case c.Method == "GET" && c.Path == "/v1/mailboxes/mb-kim/credentials":
			return 200, map[string]any{"credentials": map[string]any{"email": "kim@getother.com", "password": "pw-kim", "app_password": "",
				"imap_host": "127.0.0.1", "imap_port": h.mail.imapPort, "smtp_host": "127.0.0.1", "smtp_port": h.mail.smtpPort}}
		}
		return 0, nil
	})

	var p mixedPool
	p.maildoso = h.useConnection(providers.Maildoso, md, "md-key")
	p.cheap = h.useConnection(providers.CheapInboxes, ci, "ci_live_mixed")
	var err error
	p.manual, err = h.svc.AddConnection(ctx, h.org, ConnectionInput{Service: providers.Manual, Name: "Google Workspace"})
	require.NoError(t, err)
	_, err = h.svc.AddConnection(ctx, h.org, ConnectionInput{Service: providers.Manual, APIKey: "x"})
	assert.ErrorContains(t, err, "no API key")

	// Maildoso: connect the domain, then a confirmed buy imports sam.
	_, err = h.svc.ConnectServiceDomain(ctx, h.org, p.maildoso.ID, "getexample.com")
	require.NoError(t, err)
	q, err := h.svc.QuoteService(ctx, h.org, p.maildoso.ID, QuoteInput{Kind: providers.KindMailboxes, Domain: "getexample.com", People: samPat[:1]})
	require.NoError(t, err)
	bought, err := h.svc.BuyService(ctx, h.org, p.maildoso.ID, q.ID, true)
	require.NoError(t, err)
	require.Len(t, bought.Imported, 1)
	require.Empty(t, bought.Imported[0].Error)

	// CheapInboxes: connect, and a sync imports kim. A mailbox on one
	// connection's domain is never quoted at another's.
	_, err = h.svc.ConnectServiceDomain(ctx, h.org, p.cheap.ID, "getother.com")
	require.NoError(t, err)
	_, err = h.svc.QuoteService(ctx, h.org, p.cheap.ID, QuoteInput{Kind: providers.KindMailboxes, Domain: "getexample.com", People: samPat[:1]})
	assert.ErrorContains(t, err, "not one of CheapInboxes's domains")
	_, err = h.svc.ConnectServiceDomain(ctx, h.org, p.cheap.ID, "getexample.com")
	assert.ErrorContains(t, err, "belongs to another connection of this pool")
	synced, err := h.svc.SyncService(ctx, h.org, p.cheap.ID, true)
	require.NoError(t, err)
	require.Len(t, synced.Imported, 1)
	require.Empty(t, synced.Imported[0].Error)

	// Manual: a domain and a mailbox added by hand. A service connection
	// takes its domains only through the service.
	_, err = h.svc.AddDomain(ctx, h.org, DomainInput{Domain: "getfourth.com", ConnectionID: p.maildoso.ID})
	assert.ErrorContains(t, err, "connect the domain there")
	added, err := h.svc.AddDomain(ctx, h.org, DomainInput{Domain: "getthird.com", ConnectionID: p.manual.ID})
	require.NoError(t, err)
	assert.Equal(t, p.manual.ID, added.Domain.ConnectionID.String)
	res, err := h.svc.AddMailboxes(ctx, h.org, []MailboxInput{h.mailbox("lee@getthird.com", "pw-lee")})
	require.NoError(t, err)
	require.Empty(t, res[0].Error)

	for _, c := range []*Connection{&p.maildoso, &p.cheap, &p.manual} {
		all, err := h.svc.ListConnections(ctx, h.org, c.PoolID)
		require.NoError(t, err)
		for _, v := range all {
			if v.ID == c.ID {
				*c = v
			}
		}
	}
	p.sam, err = h.store.GetMailboxByEmail(ctx, "sam@getexample.com")
	require.NoError(t, err)
	p.kim, err = h.store.GetMailboxByEmail(ctx, "kim@getother.com")
	require.NoError(t, err)
	p.lee, err = h.store.GetMailboxByEmail(ctx, "lee@getthird.com")
	require.NoError(t, err)
	return p
}

// health is each mailbox's line in the overview, by address.
func (h *harness) health() map[string]MailboxHealth {
	o, err := h.svc.Overview(h.ctx, h.org)
	require.NoError(h.t, err)
	out := map[string]MailboxHealth{}
	for _, mb := range o.Mailboxes {
		out[mb.Email] = mb
	}
	return out
}

// Two mailbox services and a manual connection in one pool send together:
// every prospect goes out, from mailboxes of every connection.
func TestConnectionsSendTogether(t *testing.T) {
	h := newHarness(t)
	ctx := h.ctx
	p := h.mixedPool()

	assert.Equal(t, p.maildoso.PoolID, p.cheap.PoolID)
	assert.Equal(t, p.maildoso.PoolID, p.manual.PoolID)
	for _, c := range []Connection{p.maildoso, p.cheap, p.manual} {
		require.Len(t, c.Domains, 1, c.Name)
		assert.Equal(t, 1, c.Mailboxes, c.Name)
	}
	assert.Equal(t, "getexample.com", p.maildoso.Domains[0].Domain)
	assert.Equal(t, "getother.com", p.cheap.Domains[0].Domain)
	assert.Equal(t, "getthird.com", p.manual.Domains[0].Domain)
	health := h.health()
	assert.Equal(t, p.maildoso.ID, health["sam@getexample.com"].ConnectionID)
	assert.Equal(t, p.cheap.ID, health["kim@getother.com"].ConnectionID)
	assert.Equal(t, p.manual.ID, health["lee@getthird.com"].ConnectionID)

	_, err := h.svc.UpdateSettings(ctx, h.org, SettingsInput{MailingAddress: "NeboAI, 123 Main St, Provo, UT 84601", Timezone: "UTC"})
	require.NoError(t, err)
	h.refresh()
	h.svc.PollReplies(ctx)
	got := h.prospects(creator("ava@example.org", "Ava", "v1"), creator("ben@example.org", "Ben", "v2"), creator("cal@example.org", "Cal", "v3"))
	ids := []string{got["ava@example.org"].ID, got["ben@example.org"].ID, got["cal@example.org"].ID}
	camp, err := h.svc.DraftOutreach(ctx, h.org, CampaignInput{Name: "Mixed", Steps: threeSteps[:2], ProspectIDs: ids, ApprovalMode: "first"})
	require.NoError(t, err)
	require.Len(t, camp.Drafts, 3)
	var drafts []string
	for _, d := range camp.Drafts {
		drafts = append(drafts, d.MessageID)
	}
	_, err = h.svc.Review(ctx, h.org, "approve", drafts)
	require.NoError(t, err)

	h.svc.Tick(ctx)
	require.Equal(t, 3, h.mail.count(), "one send from each mailbox in the pass")
	from := map[string]string{}
	senders := map[string]bool{}
	for _, to := range []string{"ava@example.org", "ben@example.org", "cal@example.org"} {
		sent := h.mail.sentTo(to)
		require.Len(t, sent, 1, to)
		from[to] = sent[0].Header.Get("From")
		for _, box := range []string{"sam@getexample.com", "kim@getother.com", "lee@getthird.com"} {
			if strings.Contains(from[to], box) {
				senders[box] = true
			}
		}
	}
	assert.Len(t, senders, 3, "the pool sent from the Maildoso, CheapInboxes and manual mailboxes")

	// The follow-up goes from the same mailbox, in the thread, whatever
	// connection that mailbox came from.
	h.advance(48*time.Hour + time.Minute)
	h.svc.Tick(ctx)
	for to, f := range from {
		sent := h.mail.sentTo(to)
		require.Len(t, sent, 2, to)
		assert.Equal(t, f, sent[1].Header.Get("From"), to)
		assert.Equal(t, sent[0].Header.Get("Message-Id"), sent[1].Header.Get("In-Reply-To"), to)
	}
}

// Maildoso's mailboxes default to 15 a day in a pool whose limit is 30; a
// mailbox's own limit beats its connection's, and the send path holds to it.
func TestMaildosoDefaultInAMixedPool(t *testing.T) {
	h := newHarness(t)
	ctx := h.ctx
	p := h.mixedPool()
	pool, err := h.mailer.GetPathway(ctx, h.org.ID, p.maildoso.PoolID)
	require.NoError(t, err)
	limits := email.LimitsOf(pool)
	limits.WarmupStart = 0 // no warm-up, so the allowance is the limit itself
	_, err = h.mailer.UpdatePathway(ctx, h.org.ID, pool.ID, email.PathwayInput{Kinds: email.KindsOf(pool), Limits: &limits})
	require.NoError(t, err)
	require.EqualValues(t, 30, limits.DailyLimit)

	health := h.health()
	assert.EqualValues(t, 15, health["sam@getexample.com"].LimitInUse, "Maildoso's terms")
	assert.EqualValues(t, 15, health["sam@getexample.com"].AllowToday)
	assert.Nil(t, health["sam@getexample.com"].DailyLimit, "it has no limit of its own")
	assert.EqualValues(t, 30, health["kim@getother.com"].LimitInUse, "the pool's")
	assert.EqualValues(t, 30, health["lee@getthird.com"].LimitInUse)

	require.NoError(t, h.mailer.SetMailboxDailyLimit(ctx, h.org.ID, p.sam.ID, ptr(int64(20))))
	assert.EqualValues(t, 20, h.health()["sam@getexample.com"].LimitInUse, "its own limit beats its connection's")
	require.NoError(t, h.mailer.SetMailboxDailyLimit(ctx, h.org.ID, p.sam.ID, nil))
	assert.EqualValues(t, 15, h.health()["sam@getexample.com"].LimitInUse)

	// The send path: at a connection default of 1, sam sends one a day.
	_, err = h.svc.UpdateConnection(ctx, h.org, p.maildoso.ID, ConnectionInput{DailyLimit: ptr(int64(1))})
	require.NoError(t, err)
	for _, mb := range []db.Mailbox{p.kim, p.lee} {
		require.NoError(t, h.store.SetMailboxStatus(ctx, db.SetMailboxStatusParams{Status: "paused", PausedReason: "held for the test", ID: mb.ID}))
	}
	_, err = h.svc.UpdateSettings(ctx, h.org, SettingsInput{MailingAddress: "NeboAI, 123 Main St, Provo, UT 84601", Timezone: "UTC"})
	require.NoError(t, err)
	h.refresh()
	got := h.prospects(creator("ivy@example.org", "Ivy", "v"), creator("jo@example.org", "Jo", "v"))
	camp, err := h.svc.DraftOutreach(ctx, h.org, CampaignInput{Name: "x", Steps: threeSteps, ProspectIDs: []string{got["ivy@example.org"].ID, got["jo@example.org"].ID}})
	require.NoError(t, err)
	require.Len(t, camp.Drafts, 2)
	_, err = h.svc.Review(ctx, h.org, "approve", []string{camp.Drafts[0].MessageID, camp.Drafts[1].MessageID})
	require.NoError(t, err)
	h.svc.Tick(ctx)
	h.advance(10 * time.Minute)
	h.svc.Tick(ctx)
	assert.Equal(t, 1, h.mail.count(), "one a day under the Maildoso connection's limit")
	held := camp.Drafts[1].MessageID
	if len(h.mail.sentTo("jo@example.org")) == 1 {
		held = camp.Drafts[0].MessageID
	}
	m, err := h.store.GetOutreachEmail(ctx, db.GetOutreachEmailParams{ID: held, OrgID: sql.NullString{String: h.org.ID, Valid: true}})
	require.NoError(t, err)
	assert.Contains(t, m.ErrorMessage.String, "daily limit")
}

// A domain moves between connections of its pool only; a connection is
// removed only once it holds no domains; a service's domain stays in its
// pool, while one added by hand may move to another pool.
func TestConnectionsMoveAndRemove(t *testing.T) {
	h := newHarness(t)
	ctx := h.ctx
	p := h.mixedPool()
	third := p.manual.Domains[0].ID

	err := h.svc.RemoveConnection(ctx, h.org, p.manual.ID)
	assert.ErrorContains(t, err, "still holds getthird.com")

	other, err := h.mailer.CreatePathway(ctx, h.org.ID, email.PathwayInput{Name: "Second pool", Provider: email.ProviderMailboxes})
	require.NoError(t, err)
	otherManual, err := h.svc.AddConnection(ctx, h.org, ConnectionInput{PoolID: other.ID, Service: providers.Manual})
	require.NoError(t, err)
	assert.Equal(t, email.ManualConnectionName, otherManual.Name)
	_, err = h.svc.MoveDomainToConnection(ctx, h.org, third, otherManual.ID)
	assert.ErrorContains(t, err, "only between connections of its own pool")
	_, err = h.mailer.MoveDomain(ctx, h.org.ID, p.maildoso.Domains[0].ID, other.ID)
	assert.ErrorContains(t, err, "stays in that pool")

	// lee moves to the Maildoso connection with its domain, and now sends
	// under Maildoso's default.
	require.NoError(t, h.store.SetMailboxServiceRef(ctx, db.SetMailboxServiceRefParams{ServiceRef: "g-1", ID: p.lee.ID, OrgID: h.org.ID}))
	moved, err := h.svc.MoveDomainToConnection(ctx, h.org, third, p.maildoso.ID)
	require.NoError(t, err)
	assert.Len(t, moved.Domains, 2)
	assert.Equal(t, 2, moved.Mailboxes)
	lee, err := h.store.GetMailboxByEmail(ctx, "lee@getthird.com")
	require.NoError(t, err)
	assert.Empty(t, lee.ServiceRef, "the old connection's id means nothing at the new one")
	assert.EqualValues(t, 15, h.health()["lee@getthird.com"].LimitInUse)

	require.NoError(t, h.svc.RemoveConnection(ctx, h.org, p.manual.ID))
	all, err := h.svc.ListConnections(ctx, h.org, p.maildoso.PoolID)
	require.NoError(t, err)
	assert.Len(t, all, 2)

	// Back to manual, then to the other pool: it joins that pool's manual
	// connection, and its mailbox goes with it.
	_, err = h.svc.MoveDomainToConnection(ctx, h.org, third, mustManual(t, h, p.maildoso.PoolID).ID)
	require.NoError(t, err)
	_, err = h.mailer.MoveDomain(ctx, h.org.ID, third, other.ID)
	require.NoError(t, err)
	d, err := h.store.GetOutreachDomain(ctx, db.GetOutreachDomainParams{ID: third, OrgID: h.org.ID})
	require.NoError(t, err)
	assert.Equal(t, otherManual.ID, d.ConnectionID.String)
	pool, err := h.store.GetMailboxPathway(ctx, lee.ID)
	require.NoError(t, err)
	assert.Equal(t, other.ID, pool.ID)
}

// mustManual adds a manual connection to a pool.
func mustManual(t *testing.T, h *harness, poolID string) Connection {
	c, err := h.svc.AddConnection(h.ctx, h.org, ConnectionInput{PoolID: poolID, Service: providers.Manual})
	require.NoError(t, err)
	return c
}
