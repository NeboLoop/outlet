package outbound

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
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

// Each mailbox service is faked from its published API spec: the paths,
// the auth header, the request bodies and the answers. Mailboxes the fake
// makes log in to the harness's real local SMTP and IMAP server, so an
// import runs the same login tests every mailbox passes. Nothing here
// reaches the internet.

type serviceCall struct {
	Method, Path, Query, Auth, Idempotency string
	Body                                   string
}

// fakeService is one service's API: route answers "METHOD /path" (the query
// is in the call).
type fakeService struct {
	t     *testing.T
	mu    sync.Mutex
	calls []serviceCall
	auth  string
	also  []string // more accounts' keys the fake takes
	route func(c serviceCall) (int, any)
	srv   *httptest.Server
}

func newFakeService(t *testing.T, auth string, route func(c serviceCall) (int, any)) *fakeService {
	f := &fakeService{t: t, auth: auth, route: route}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		c := serviceCall{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Auth: r.Header.Get("Authorization"), Idempotency: r.Header.Get("Idempotency-Key"), Body: string(body)}
		f.mu.Lock()
		f.calls = append(f.calls, c)
		f.mu.Unlock()
		if c.Auth != f.auth && !slices.Contains(f.also, c.Auth) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"detail":"Authentication required"}`))
			return
		}
		status, out := f.route(c)
		if status == 0 {
			status = http.StatusNotFound
			out = map[string]any{"error": map[string]string{"code": "NOT_FOUND", "message": c.Method + " " + c.Path + " is not in the spec"}}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if out != nil {
			_ = json.NewEncoder(w).Encode(out)
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

// called lists the calls to one endpoint.
func (f *fakeService) called(method, path string) []serviceCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []serviceCall
	for _, c := range f.calls {
		if c.Method == method && c.Path == path {
			out = append(out, c)
		}
	}
	return out
}

// useConnection points the service at the fake and gives the brand's
// mailbox pool (made if missing) a connection to it.
func (h *harness) useConnection(service string, f *fakeService, key string) Connection {
	if h.svc.serviceAPI == nil {
		h.svc.serviceAPI = map[string]string{}
	}
	h.svc.serviceAPI[service] = f.srv.URL
	c, err := h.svc.AddConnection(h.ctx, h.org, ConnectionInput{Service: service, APIKey: key})
	require.NoError(h.t, err)
	return c
}

// noKey fails if the API key shows up in anything the service returned.
func noKey(t *testing.T, key string, values ...any) {
	t.Helper()
	for _, v := range values {
		raw, err := json.Marshal(v)
		require.NoError(t, err)
		assert.NotContains(t, string(raw), key)
		if e, ok := v.(error); ok && e != nil {
			assert.NotContains(t, e.Error(), key)
		}
	}
}

func harnessHost(h *harness) (smtp, imap map[string]any) {
	return map[string]any{"smtp_host": "127.0.0.1", "port": h.mail.smtpPort}, map[string]any{"imap_host": "127.0.0.1", "port": h.mail.imapPort}
}

var samPat = []providers.Person{{FirstName: "Sam", LastName: "Rivera", Local: "sam"}, {FirstName: "Pat", LastName: "Rivera", Local: "pat"}}

func TestMaildoso(t *testing.T) {
	h := newHarness(t)
	ctx := h.ctx
	const key = "md-pat-secret-0001"
	smtpHost, imapHost := harnessHost(h)
	passwords := map[string]string{"sam@getexample.com": "pw-sam", "pat@getexample.com": "pw-pat", "lee@getexample.com": "pw-lee"}
	var mu sync.Mutex
	accounts := []map[string]any{} // what Maildoso has
	account := func(id int, address, first, status string) map[string]any {
		return map[string]any{"id": id, "user_id": 1, "email_account": address, "is_active": true, "domain_id": 7, "forwarding_account_id": nil,
			"password": passwords[address], "first_name": first, "last_name": "Rivera", "created_at": "2026-11-10T16:00:00",
			"imap": imapHost, "smtp": smtpHost, "forwarding_account": nil, "provider": "maildoso", "status": status, "is_admin": false, "exported_to_sequencers": []any{}}
	}
	f := newFakeService(t, "Bearer "+key, func(c serviceCall) (int, any) {
		mu.Lock()
		defer mu.Unlock()
		switch c.Method + " " + c.Path {
		case "GET /v1/user/domains":
			return 200, []map[string]any{{"id": 7, "user_id": 1, "domain_name": "getexample.com", "is_hidden": false, "enable_tracking": false, "linked_accounts": 0,
				"created_at": "2026-11-10T16:00:00", "domain_type": "EXTERNAL", "domain_status": "AWAITING_NS", "name_servers": []string{"ns1.maildoso.net", "ns2.maildoso.net"}, "accounts_limit": 5}}
		case "POST /v1/user/domains/external":
			var body struct {
				Domains   []string `json:"domains"`
				Providers []string `json:"providers"`
			}
			require.NoError(t, json.Unmarshal([]byte(c.Body), &body))
			assert.Equal(t, []string{"getexample.com"}, body.Domains)
			assert.Equal(t, []string{"maildoso"}, body.Providers)
			return 201, map[string]any{"results": []map[string]any{{"name": "getexample.com", "result": "success", "message": nil, "id": 7}}, "task_id": "t-1"}
		case "POST /v1/user/accounts":
			var body []map[string]any
			require.NoError(t, json.Unmarshal([]byte(c.Body), &body))
			res := []map[string]any{}
			for i, b := range body {
				a := account(100+i, b["email_account"].(string), b["first_name"].(string), "active")
				accounts = append(accounts, a)
				res = append(res, a)
			}
			return 202, map[string]any{"results": res, "task_id": "t-2"}
		case "GET /v1/user/accounts-lookup":
			return 200, map[string]any{"items": accounts, "meta": map[string]any{"offset": 0, "limit": nil, "total": len(accounts), "current": len(accounts)}}
		case "DELETE /v1/user/accounts":
			var ids []int
			require.NoError(t, json.Unmarshal([]byte(c.Body), &ids))
			kept := accounts[:0]
			for _, a := range accounts {
				if a["id"] != ids[0] {
					kept = append(kept, a)
				}
			}
			accounts = kept
			return 204, nil
		}
		return 0, nil
	})

	// A wrong key is refused before it is saved.
	h.svc.serviceAPI = map[string]string{providers.Maildoso: f.srv.URL}
	_, err := h.svc.AddConnection(ctx, h.org, ConnectionInput{Service: providers.Maildoso, APIKey: "wrong-key"})
	assert.ErrorContains(t, err, "API key was refused")
	_, err = h.svc.AddConnection(ctx, h.org, ConnectionInput{Service: providers.Maildoso})
	assert.ErrorContains(t, err, "Enter the Maildoso API key")
	all, err := h.svc.ListConnections(ctx, h.org, "")
	require.NoError(t, err)
	assert.Empty(t, all, "nothing is saved without a working key")

	// A Maildoso connection's mailboxes default to its terms, 15 a day; the
	// pool keeps its own limits for its other connections.
	conn, err := h.svc.AddConnection(ctx, h.org, ConnectionInput{Service: providers.Maildoso, APIKey: key})
	require.NoError(t, err)
	assert.Equal(t, "Maildoso", conn.Name)
	require.NotNil(t, conn.DailyLimit)
	assert.Equal(t, int64(15), *conn.DailyLimit)
	assert.True(t, conn.KeySet)
	pool, err := h.mailer.GetPathway(ctx, h.org.ID, conn.PoolID)
	require.NoError(t, err)
	assert.Equal(t, email.DefaultLimits(email.ProviderMailboxes).DailyLimit, pool.DailyLimit)
	// The brand can change the default; saving with no new key keeps the key.
	conn, err = h.svc.UpdateConnection(ctx, h.org, conn.ID, ConnectionInput{Name: "Maildoso main", DailyLimit: ptr(int64(12))})
	require.NoError(t, err)
	assert.Equal(t, int64(12), *conn.DailyLimit)
	assert.Equal(t, "Maildoso main", conn.Name)
	assert.True(t, conn.KeySet)
	_, err = h.svc.UpdateConnection(ctx, h.org, conn.ID, ConnectionInput{APIKey: "wrong-key", DailyLimit: ptr(int64(15))})
	assert.ErrorContains(t, err, "API key was refused")
	conn, err = h.svc.UpdateConnection(ctx, h.org, conn.ID, ConnectionInput{DailyLimit: ptr(int64(15))})
	require.NoError(t, err)
	views, err := h.mailer.ListPathways(ctx, h.org.ID)
	require.NoError(t, err)
	all, err = h.svc.ListConnections(ctx, h.org, "")
	require.NoError(t, err)
	noKey(t, key, conn, views, all)

	// Connect a domain the brand owns: Maildoso returns its nameservers and
	// the domain joins the pool.
	got, err := h.svc.ConnectServiceDomain(ctx, h.org, conn.ID, "getexample.com")
	require.NoError(t, err)
	assert.Equal(t, []string{"ns1.maildoso.net", "ns2.maildoso.net"}, got.Connection.Nameservers)
	assert.True(t, got.Check.OK(), "MX, SPF and DMARC are in DNS; DKIM is the service's own selector")
	d, err := h.store.GetOutreachDomain(ctx, db.GetOutreachDomainParams{ID: got.DomainID, OrgID: h.org.ID})
	require.NoError(t, err)
	assert.Equal(t, pool.ID, d.PathwayID.String)
	assert.Equal(t, conn.ID, d.ConnectionID.String)
	assert.Equal(t, "provider", d.DnsHost)
	_, err = h.svc.ConnectServiceDomain(ctx, h.org, conn.ID, "neboai.com")
	assert.ErrorContains(t, err, "protected")

	// Mailboxes: a quote first. Maildoso's API has no price.
	q, err := h.svc.QuoteService(ctx, h.org, conn.ID, QuoteInput{Kind: providers.KindMailboxes, Domain: "getexample.com", People: samPat})
	require.NoError(t, err)
	assert.False(t, q.PriceKnown)
	assert.Nil(t, q.PriceCents)
	assert.Equal(t, 2, q.Mailboxes)
	_, err = h.svc.QuoteService(ctx, h.org, conn.ID, QuoteInput{Kind: providers.KindConnect, Domain: "other.com"})
	assert.ErrorContains(t, err, "free at Maildoso")

	// Nothing reaches the buy endpoint without confirm and a valid quote.
	_, err = h.svc.BuyService(ctx, h.org, conn.ID, q.ID, false)
	assert.ErrorContains(t, err, "confirm")
	_, err = h.svc.BuyService(ctx, h.org, conn.ID, "no-such-quote", true)
	assert.ErrorContains(t, err, "no such quote")
	assert.Empty(t, f.called("POST", "/v1/user/accounts"))

	// Confirmed: Maildoso creates them and each is imported after logging in
	// to SMTP and IMAP.
	res, err := h.svc.BuyService(ctx, h.org, conn.ID, q.ID, true)
	require.NoError(t, err)
	require.Len(t, res.Imported, 2)
	for _, r := range res.Imported {
		assert.Empty(t, r.Error)
		assert.NotEmpty(t, r.ID)
	}
	assert.Empty(t, res.Pending)
	sam, err := h.store.GetMailboxByEmail(ctx, "sam@getexample.com")
	require.NoError(t, err)
	assert.Equal(t, "100", sam.ServiceRef)
	assert.Equal(t, "active", sam.Status)
	noKey(t, key, q, res)

	// A quote buys once.
	_, err = h.svc.BuyService(ctx, h.org, conn.ID, q.ID, true)
	assert.ErrorContains(t, err, "expired or was already used")
	assert.Len(t, f.called("POST", "/v1/user/accounts"), 1)

	// Sync: Maildoso has lee, the pool doesn't.
	mu.Lock()
	accounts = append(accounts, account(200, "lee@getexample.com", "Lee", "active"))
	mu.Unlock()
	sync, err := h.svc.SyncService(ctx, h.org, conn.ID, false)
	require.NoError(t, err)
	require.Len(t, sync.OnlyAtService, 1)
	assert.Equal(t, "lee@getexample.com", sync.OnlyAtService[0].Email)
	assert.Empty(t, sync.OnlyInPool)
	assert.Empty(t, sync.Imported)
	noKey(t, key, sync)
	// Importing tests lee's logins like any mailbox; the harness has no lee.
	sync, err = h.svc.SyncService(ctx, h.org, conn.ID, true)
	require.NoError(t, err)
	require.Len(t, sync.Imported, 1)
	assert.Contains(t, sync.Imported[0].Error, "SMTP login failed")
	_, err = h.store.GetMailboxByEmail(ctx, "lee@getexample.com")
	assert.ErrorIs(t, err, sql.ErrNoRows)

	// Delete sam at Maildoso: it stays in the pool, paused for good.
	require.NoError(t, h.svc.DeleteServiceMailbox(ctx, h.org, conn.ID, sam.ID))
	del := f.called("DELETE", "/v1/user/accounts")
	require.Len(t, del, 1)
	assert.JSONEq(t, `[100]`, del[0].Body)
	sam, err = h.store.GetMailboxByEmail(ctx, "sam@getexample.com")
	require.NoError(t, err)
	assert.Equal(t, "paused", sam.Status)
	assert.Contains(t, sam.PausedReason, "Deleted: at Maildoso")
	assert.ErrorContains(t, h.svc.Pause(ctx, h.org, "mailbox", sam.ID, true), "deleted")
	sync, err = h.svc.SyncService(ctx, h.org, conn.ID, false)
	require.NoError(t, err)
	require.Len(t, sync.OnlyInPool, 1)
	assert.Equal(t, "sam@getexample.com", sync.OnlyInPool[0].Email)

	// Every call carried the token as a Bearer token.
	for _, c := range f.calls {
		assert.True(t, c.Auth == "Bearer "+key || c.Auth == "Bearer wrong-key", c.Path)
	}
}

// A quote buys only at the connection that gave it, even another account
// of the same service, and only until it expires.
func TestQuotesExpireAndStayWithTheirConnection(t *testing.T) {
	h := newHarness(t)
	const key, key2 = "md-pat-secret-0002", "md-pat-secret-0022"
	f := newFakeService(t, "Bearer "+key, func(c serviceCall) (int, any) {
		switch c.Method + " " + c.Path {
		case "GET /v1/user/domains":
			return 200, []any{}
		case "POST /v1/user/domains/search":
			return 200, []map[string]any{{"name": "freshexample.com", "status": "free", "available": "true", "error": nil}}
		}
		return 0, nil
	})
	f.also = []string{"Bearer " + key2}
	ci := newFakeService(t, "Bearer ci_live_x", func(c serviceCall) (int, any) {
		if c.Method+" "+c.Path == "GET /v1/domains" {
			return 200, map[string]any{"domains": []any{}, "pagination": map[string]int{"total": 0}}
		}
		return 0, nil
	})
	a := h.useConnection(providers.Maildoso, f, key)
	b := h.useConnection(providers.Maildoso, f, key2)
	c := h.useConnection(providers.CheapInboxes, ci, "ci_live_x")
	assert.Equal(t, a.PoolID, b.PoolID)
	assert.Equal(t, a.PoolID, c.PoolID, "every connection is in the brand's one pool")

	q, err := h.svc.QuoteService(h.ctx, h.org, a.ID, QuoteInput{Kind: providers.KindDomain, Domain: "freshexample.com"})
	require.NoError(t, err)
	assert.False(t, q.PriceKnown)
	assert.Contains(t, q.Summary, "no price")

	for _, other := range []Connection{b, c} {
		_, err = h.svc.BuyService(h.ctx, h.org, other.ID, q.ID, true)
		assert.ErrorContains(t, err, "no such quote for "+other.Name)
	}
	assert.Empty(t, f.called("POST", "/v1/user/domains"), "Maildoso's buy endpoint was never called")
	assert.Empty(t, ci.called("POST", "/v1/orders/checkout"))

	h.advance(31 * time.Minute)
	_, err = h.svc.BuyService(h.ctx, h.org, a.ID, q.ID, true)
	assert.ErrorContains(t, err, "expired")
	assert.Empty(t, f.called("POST", "/v1/user/domains"))
}

func TestCheapInboxes(t *testing.T) {
	h := newHarness(t)
	ctx := h.ctx
	const key = "ci_live_secret_0003"
	var mu sync.Mutex
	staged := []map[string]any{}
	mailboxes := []map[string]any{}
	domains := []map[string]any{}
	creds := map[string]map[string]any{}
	f := newFakeService(t, "Bearer "+key, func(c serviceCall) (int, any) {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case c.Method == "GET" && c.Path == "/v1/domains":
			assert.Contains(t, c.Query, "limit=")
			return 200, map[string]any{"domains": domains, "pagination": map[string]int{"total": len(domains), "limit": 100, "offset": 0}}
		case c.Method == "POST" && c.Path == "/v1/domain-imports/bulk":
			var body struct {
				Domains        []string `json:"domains"`
				SourceProvider string   `json:"source_provider"`
			}
			require.NoError(t, json.Unmarshal([]byte(c.Body), &body))
			assert.Equal(t, "manual", body.SourceProvider)
			e := map[string]any{"id": "6f1c1e8e-0000-4000-8000-000000000001", "domain_name_normalized": body.Domains[0], "state": "staged", "source_provider": "manual"}
			staged = append(staged, e)
			return 200, map[string]any{"entries": []any{e}, "errors": []any{}}
		case c.Method == "GET" && c.Path == "/v1/domain-imports":
			return 200, map[string]any{"entries": staged}
		case c.Method == "POST" && c.Path == "/v1/orders/quote":
			var body struct {
				Cart struct {
					Domains []map[string]any `json:"domains"`
				} `json:"cart"`
			}
			require.NoError(t, json.Unmarshal([]byte(c.Body), &body))
			assert.Equal(t, "6f1c1e8e-0000-4000-8000-000000000001", body.Cart.Domains[0]["staged_import_id"])
			assert.Equal(t, "com", body.Cart.Domains[0]["tld"])
			assert.Equal(t, map[string]any{"provider": "google", "count": float64(2)}, body.Cart.Domains[0]["mailboxes"])
			return 200, map[string]any{"quote": map[string]any{"items": []any{}, "totals": map[string]any{"total_today_cents": 700, "mailbox_monthly_cents": 700}}}
		case c.Method == "POST" && c.Path == "/v1/orders/checkout":
			var body struct {
				Domains []struct {
					Domain         string `json:"domain"`
					StagedImportID string `json:"staged_import_id"`
					Mailboxes      struct {
						Provider string              `json:"provider"`
						Accounts []map[string]string `json:"accounts"`
					} `json:"mailboxes"`
				} `json:"domains"`
			}
			require.NoError(t, json.Unmarshal([]byte(c.Body), &body))
			d := body.Domains[0]
			assert.NotEmpty(t, d.StagedImportID)
			out := []map[string]any{}
			for i, a := range d.Mailboxes.Accounts {
				id := []string{"mb-sam", "mb-pat"}[i]
				m := map[string]any{"id": id, "full_email": a["email_prefix"] + "@" + d.Domain, "first_name": a["first_name"], "last_name": a["last_name"], "status": "provisioning"}
				mailboxes = append(mailboxes, m)
				out = append(out, m)
			}
			domains = append(domains, map[string]any{"id": "d-1", "domain": d.Domain, "status": "pending_ns_verification", "nameservers": []string{"ns1.cheapinboxes.com", "ns2.cheapinboxes.com"}})
			return 200, map[string]any{"order_id": "ord_1", "status": "provisioning", "total_charged_cents": 700, "domains": domains, "mailboxes": out}
		case c.Method == "GET" && c.Path == "/v1/mailboxes":
			return 200, map[string]any{"mailboxes": mailboxes, "pagination": map[string]int{"total": len(mailboxes), "limit": 100, "offset": 0}}
		case c.Method == "GET" && strings.HasSuffix(c.Path, "/credentials"):
			id := strings.Split(c.Path, "/")[3]
			if cr, ok := creds[id]; ok {
				return 200, map[string]any{"credentials": cr}
			}
			return 409, map[string]any{"error": map[string]string{"code": "NOT_READY", "message": "mailbox is not active"}}
		case c.Method == "POST" && strings.HasSuffix(c.Path, "/cancel"):
			return 200, map[string]any{"success": true, "mailbox_id": strings.Split(c.Path, "/")[3]}
		}
		return 0, nil
	})
	conn := h.useConnection(providers.CheapInboxes, f, key)
	assert.Nil(t, conn.DailyLimit, "CheapInboxes' mailboxes keep the pool's limits")

	// Connect stages the import; nothing is charged.
	got, err := h.svc.ConnectServiceDomain(ctx, h.org, conn.ID, "getexample.com")
	require.NoError(t, err)
	assert.Equal(t, "staged", got.Connection.Status)
	assert.Contains(t, got.Connection.Note, "first mailboxes")

	// The quote is CheapInboxes' own price for the order.
	q, err := h.svc.QuoteService(ctx, h.org, conn.ID, QuoteInput{Kind: providers.KindMailboxes, Domain: "getexample.com", People: samPat})
	require.NoError(t, err)
	require.True(t, q.PriceKnown)
	assert.Equal(t, int64(700), *q.PriceCents)
	assert.Contains(t, q.Summary, "$7.00 today")
	_, err = h.svc.BuyService(ctx, h.org, conn.ID, q.ID, false)
	assert.Error(t, err)
	assert.Empty(t, f.called("POST", "/v1/orders/checkout"), "no checkout without confirm")

	// Checkout charges the card; the mailboxes are still being made.
	res, err := h.svc.BuyService(ctx, h.org, conn.ID, q.ID, true)
	require.NoError(t, err)
	co := f.called("POST", "/v1/orders/checkout")
	require.Len(t, co, 1)
	assert.Equal(t, q.ID, co[0].Idempotency, "a retried checkout can't charge twice")
	assert.Contains(t, co[0].Body, `"email_prefix":"sam"`)
	assert.Equal(t, int64(700), *res.ChargedCents)
	assert.Equal(t, []string{"ns1.cheapinboxes.com", "ns2.cheapinboxes.com"}, res.Connection.Nameservers)
	assert.Empty(t, res.Imported)
	assert.Len(t, res.Pending, 2)

	// Once active, a sync imports them with their credentials.
	mu.Lock()
	for _, m := range mailboxes {
		m["status"] = "active"
	}
	creds["mb-sam"] = map[string]any{"email": "sam@getexample.com", "password": "pw-sam", "app_password": "", "imap_host": "127.0.0.1", "imap_port": h.mail.imapPort, "smtp_host": "127.0.0.1", "smtp_port": h.mail.smtpPort}
	creds["mb-pat"] = map[string]any{"email": "pat@getexample.com", "password": "pw-pat", "app_password": "", "imap_host": "127.0.0.1", "imap_port": h.mail.imapPort, "smtp_host": "127.0.0.1", "smtp_port": h.mail.smtpPort}
	mu.Unlock()
	sync, err := h.svc.SyncService(ctx, h.org, conn.ID, true)
	require.NoError(t, err)
	require.Len(t, sync.Imported, 2)
	for _, r := range sync.Imported {
		assert.Empty(t, r.Error)
	}
	assert.Empty(t, sync.OnlyAtService)
	pat, err := h.store.GetMailboxByEmail(ctx, "pat@getexample.com")
	require.NoError(t, err)
	assert.Equal(t, "mb-pat", pat.ServiceRef)

	// Delete cancels at the end of the billing period.
	require.NoError(t, h.svc.DeleteServiceMailbox(ctx, h.org, conn.ID, pat.ID))
	assert.Len(t, f.called("POST", "/v1/mailboxes/mb-pat/cancel"), 1)
	noKey(t, key, got, q, res, sync)
}

// A Google Workspace mailbox signs in with an app password, which
// CheapInboxes makes on request: the first sync asks for it.
func TestCheapInboxesAsksForAnAppPassword(t *testing.T) {
	f := newFakeService(t, "Bearer ci_live_y", func(c serviceCall) (int, any) {
		switch c.Method + " " + c.Path {
		case "GET /v1/mailboxes/mb-1/credentials":
			return 200, map[string]any{"credentials": map[string]any{"email": "a@getexample.com", "password": "pw", "app_password": "", "imap_host": "imap.gmail.com", "imap_port": 993, "smtp_host": "smtp.gmail.com", "smtp_port": 587}}
		case "GET /v1/mailboxes/mb-1/app-password":
			return 200, map[string]any{"status": "none", "appPassword": nil}
		case "POST /v1/mailboxes/mb-1/app-password":
			return 202, map[string]any{"success": true, "status": "queued"}
		}
		return 0, nil
	})
	p, err := providers.New(providers.CheapInboxes, "ci_live_y", f.srv.URL)
	require.NoError(t, err)
	_, err = p.Login(context.Background(), providers.Mailbox{ID: "mb-1", Email: "a@getexample.com"})
	var nr *providers.NotReady
	require.ErrorAs(t, err, &nr)
	assert.Contains(t, nr.Reason, "app password")
	assert.Len(t, f.called("POST", "/v1/mailboxes/mb-1/app-password"), 1)
}

func TestInfraforge(t *testing.T) {
	h := newHarness(t)
	ctx := h.ctx
	const key = "if-secret-0004"
	var mu sync.Mutex
	mailboxes := []map[string]any{}
	f := newFakeService(t, key, func(c serviceCall) (int, any) {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case c.Method == "GET" && c.Path == "/domains":
			return 200, []map[string]any{}
		case c.Method == "GET" && c.Path == "/domains/transfer-ns-info":
			return 200, map[string]any{"nameservers": []string{"ns1.premiumdns.zone", "ns2.premiumdns.zone"}}
		case c.Method == "GET" && c.Path == "/workspaces":
			return 200, []map[string]any{{"id": "wks_1", "name": "Main"}}
		case c.Method == "POST" && c.Path == "/domains/transfer":
			assert.JSONEq(t, `{"domains":["getexample.com"],"workspaceId":"wks_1"}`, c.Body)
			return 200, map[string]any{"domains": []map[string]any{{"domain": "getexample.com", "id": "dom_1", "nameservers": []string{"ns1.premiumdns.zone", "ns2.premiumdns.zone"}}},
				"invoice": map[string]any{"amountPaid": 2852, "total": 2852, "currencyCode": "USD", "status": "paid"}}
		case c.Method == "POST" && c.Path == "/mailboxes":
			var body struct {
				Domains []struct {
					Domain    string              `json:"domain"`
					Mailboxes []map[string]string `json:"mailboxes"`
				} `json:"domains"`
			}
			require.NoError(t, json.Unmarshal([]byte(c.Body), &body))
			out := []map[string]any{}
			for i, m := range body.Domains[0].Mailboxes {
				mb := map[string]any{"id": []string{"mbx_sam", "mbx_pat"}[i], "email": m["email"], "firstName": m["firstName"], "lastName": m["lastName"], "status": "active"}
				mailboxes = append(mailboxes, mb)
				out = append(out, mb)
			}
			return 200, map[string]any{"mailboxes": out, "invoice": map[string]any{"amountPaid": 600, "total": 600}}
		case c.Method == "GET" && strings.HasPrefix(c.Path, "/mailboxes/"):
			assert.Equal(t, "with_credentials=true", c.Query)
			for _, mb := range mailboxes {
				if mb["id"] == strings.TrimPrefix(c.Path, "/mailboxes/") {
					address := mb["email"].(string)
					pw := map[string]string{"sam@getexample.com": "pw-sam", "pat@getexample.com": "pw-pat"}[address]
					with := map[string]any{}
					for k, v := range mb {
						with[k] = v
					}
					with["credentials"] = map[string]any{"smtpHost": "127.0.0.1", "smtpPort": h.mail.smtpPort, "smtpUsername": address, "smtpPassword": pw,
						"imapHost": "127.0.0.1", "imapPort": h.mail.imapPort, "imapUsername": address, "imapPassword": pw}
					return 200, with
				}
			}
			return 400, map[string]any{"message": "no such mailbox"}
		case c.Method == "GET" && c.Path == "/mailboxes":
			return 200, mailboxes
		case c.Method == "DELETE" && strings.HasPrefix(c.Path, "/mailboxes/"):
			return 202, nil
		}
		return 0, nil
	})
	conn := h.useConnection(providers.Infraforge, f, key)
	assert.Nil(t, conn.DailyLimit)

	// Infraforge invoices connecting a domain, so it is a confirmed buy.
	_, err := h.svc.ConnectServiceDomain(ctx, h.org, conn.ID, "getexample.com")
	assert.ErrorContains(t, err, "bills for connecting")
	q, err := h.svc.QuoteService(ctx, h.org, conn.ID, QuoteInput{Kind: providers.KindConnect, Domain: "getexample.com"})
	require.NoError(t, err)
	assert.False(t, q.PriceKnown)
	assert.Contains(t, q.Summary, "ns1.premiumdns.zone")
	assert.Empty(t, f.called("POST", "/domains/transfer"))
	res, err := h.svc.BuyService(ctx, h.org, conn.ID, q.ID, true)
	require.NoError(t, err)
	assert.Equal(t, []string{"ns1.premiumdns.zone", "ns2.premiumdns.zone"}, res.Connection.Nameservers)
	assert.NotEmpty(t, res.DomainID)

	// Registering a domain needs registrant details Outlet doesn't hold.
	_, err = h.svc.QuoteService(ctx, h.org, conn.ID, QuoteInput{Kind: providers.KindDomain, Domain: "freshexample.com"})
	assert.ErrorContains(t, err, "registrant")

	// Mailboxes by name, then imported with the credentials Infraforge gives.
	q, err = h.svc.QuoteService(ctx, h.org, conn.ID, QuoteInput{Kind: providers.KindMailboxes, Domain: "getexample.com", People: samPat})
	require.NoError(t, err)
	assert.Empty(t, f.called("POST", "/mailboxes"))
	res, err = h.svc.BuyService(ctx, h.org, conn.ID, q.ID, true)
	require.NoError(t, err)
	require.Len(t, res.Imported, 2)
	for _, r := range res.Imported {
		assert.Empty(t, r.Error)
	}
	buy := f.called("POST", "/mailboxes")
	require.Len(t, buy, 1)
	assert.Contains(t, buy[0].Body, `"email":"sam@getexample.com"`)

	sync, err := h.svc.SyncService(ctx, h.org, conn.ID, false)
	require.NoError(t, err)
	assert.Empty(t, sync.OnlyAtService)
	assert.Empty(t, sync.OnlyInPool)

	sam, err := h.store.GetMailboxByEmail(ctx, "sam@getexample.com")
	require.NoError(t, err)
	require.NoError(t, h.svc.DeleteServiceMailbox(ctx, h.org, conn.ID, sam.ID))
	assert.Len(t, f.called("DELETE", "/mailboxes/mbx_sam"), 1)

	// The key goes in the Authorization header as is.
	for _, c := range f.calls {
		assert.Equal(t, key, c.Auth)
	}
	noKey(t, key, q, res, sync)
}
