package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// cheapInboxes is CheapInboxes' API (api.cheapinboxes.com/docs): a ci_live_
// key as a Bearer token, 120 requests a minute. It sells Google Workspace
// mailboxes; an order is charged to the card on file at checkout. A domain
// the brand owns is staged for import free, and the import completes in the
// order that buys its first mailboxes.
type cheapInboxes struct{ *client }

// The mailboxes this pool orders: Google Workspace, which CheapInboxes'
// credentials and app-password endpoints serve.
const ciMailboxProvider = "google"

const ciPage = 100

type ciDomain struct {
	ID          string   `json:"id"`
	Domain      string   `json:"domain"`
	Status      string   `json:"status"`
	Nameservers []string `json:"nameservers"`
}

type ciMailbox struct {
	ID        string `json:"id"`
	FullEmail string `json:"full_email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Status    string `json:"status"`
}

func (m ciMailbox) mailbox() Mailbox {
	return Mailbox{ID: m.ID, Email: strings.ToLower(m.FullEmail), FirstName: m.FirstName, LastName: m.LastName, Status: m.Status, Ready: m.Status == "active"}
}

func (c *cheapInboxes) Domains(ctx context.Context) ([]Domain, error) {
	var out []Domain
	for offset := 0; ; offset += ciPage {
		var page struct {
			Domains    []ciDomain `json:"domains"`
			Pagination struct {
				Total int `json:"total"`
			} `json:"pagination"`
		}
		if err := c.call(ctx, http.MethodGet, fmt.Sprintf("/v1/domains?limit=%d&offset=%d", ciPage, offset), nil, nil, &page); err != nil {
			return nil, err
		}
		for _, d := range page.Domains {
			out = append(out, Domain{ID: d.ID, Name: strings.ToLower(d.Domain), Status: d.Status, Nameservers: d.Nameservers})
		}
		if len(page.Domains) < ciPage || offset+ciPage >= page.Pagination.Total {
			return out, nil
		}
	}
}

func (c *cheapInboxes) Mailboxes(ctx context.Context) ([]Mailbox, error) {
	var out []Mailbox
	for offset := 0; ; offset += ciPage {
		var page struct {
			Mailboxes  []ciMailbox `json:"mailboxes"`
			Pagination struct {
				Total int `json:"total"`
			} `json:"pagination"`
		}
		if err := c.call(ctx, http.MethodGet, fmt.Sprintf("/v1/mailboxes?limit=%d&offset=%d", ciPage, offset), nil, nil, &page); err != nil {
			return nil, err
		}
		for _, m := range page.Mailboxes {
			out = append(out, m.mailbox())
		}
		if len(page.Mailboxes) < ciPage || offset+ciPage >= page.Pagination.Total {
			return out, nil
		}
	}
}

// Login reads the mailbox's credentials, available once it is active. Google
// Workspace signs in to SMTP and IMAP with an app password, which
// CheapInboxes makes only on request: the first call asks for one.
func (c *cheapInboxes) Login(ctx context.Context, mb Mailbox) (Login, error) {
	var res struct {
		Credentials struct {
			Email       string `json:"email"`
			Password    string `json:"password"`
			AppPassword string `json:"app_password"`
			IMAPHost    string `json:"imap_host"`
			IMAPPort    int    `json:"imap_port"`
			SMTPHost    string `json:"smtp_host"`
			SMTPPort    int    `json:"smtp_port"`
		} `json:"credentials"`
	}
	id := url.PathEscape(mb.ID)
	if err := c.call(ctx, http.MethodGet, "/v1/mailboxes/"+id+"/credentials", nil, nil, &res); err != nil {
		return Login{}, &NotReady{Reason: "CheapInboxes has no credentials for " + mb.Email + " yet (they come once it is active): " + err.Error()}
	}
	cr := res.Credentials
	password := cr.AppPassword
	if password == "" && strings.Contains(cr.SMTPHost, "gmail") {
		var ap struct {
			Status      string `json:"status"`
			AppPassword string `json:"appPassword"`
		}
		if err := c.call(ctx, http.MethodGet, "/v1/mailboxes/"+id+"/app-password", nil, nil, &ap); err != nil {
			return Login{}, err
		}
		if ap.AppPassword == "" {
			if ap.Status == "" || ap.Status == "none" || ap.Status == "failed" {
				if err := c.call(ctx, http.MethodPost, "/v1/mailboxes/"+id+"/app-password", nil, map[string]any{}, nil); err != nil {
					return Login{}, err
				}
				return Login{}, &NotReady{Reason: "asked CheapInboxes for an app password for " + mb.Email + "; sync again in a few minutes"}
			}
			return Login{}, &NotReady{Reason: "CheapInboxes is making an app password for " + mb.Email + " (" + ap.Status + "); sync again in a few minutes"}
		}
		password = ap.AppPassword
	}
	password = nonEmpty(password, cr.Password)
	user := nonEmpty(cr.Email, mb.Email)
	return Login{
		SMTPHost: cr.SMTPHost, SMTPPort: cr.SMTPPort, SMTPUsername: user, SMTPPassword: password,
		IMAPHost: cr.IMAPHost, IMAPPort: cr.IMAPPort, IMAPUsername: user, IMAPPassword: password,
	}, nil
}

// Connect stages a domain the brand owns for import. Nothing is charged;
// CheapInboxes finishes the import in the order for its first mailboxes.
func (c *cheapInboxes) Connect(ctx context.Context, domain string) (Connection, error) {
	var res struct {
		Entries []struct {
			ID    string `json:"id"`
			State string `json:"state"`
		} `json:"entries"`
		Errors []any `json:"errors"`
	}
	body := map[string]any{"domains": []string{domain}, "source_provider": "manual"}
	if err := c.call(ctx, http.MethodPost, "/v1/domain-imports/bulk", nil, body, &res); err != nil {
		return Connection{}, err
	}
	if len(res.Errors) > 0 {
		return Connection{}, &Error{Service: CheapInboxes, Msg: fmt.Sprintf("%s was not staged: %v", domain, res.Errors[0])}
	}
	conn := Connection{Domain: domain, Status: "staged", Note: "Staged at CheapInboxes, free. The import finishes in the order that buys its first mailboxes (get a quote, then buy); CheapInboxes then gives the nameservers to set at the domain's registrar, and runs its DNS."}
	if len(res.Entries) > 0 {
		conn.Status = res.Entries[0].State
	}
	return conn, nil
}

// cartDomain is one domain of a quote or checkout: new, staged for import,
// or already at CheapInboxes.
func (c *cheapInboxes) cartDomain(ctx context.Context, o Order) (map[string]any, error) {
	_, tld := splitDomain(o.Domain)
	d := map[string]any{"domain": o.Domain, "tld": tld}
	if o.Kind == KindDomain {
		return d, nil
	}
	domains, err := c.Domains(ctx)
	if err != nil {
		return nil, err
	}
	for _, have := range domains {
		if have.Name == o.Domain {
			d["isExisting"] = true
			return d, nil
		}
	}
	var staged struct {
		Entries []struct {
			ID     string `json:"id"`
			Domain string `json:"domain_name_normalized"`
			State  string `json:"state"`
		} `json:"entries"`
	}
	if err := c.call(ctx, http.MethodGet, "/v1/domain-imports", nil, nil, &staged); err != nil {
		return nil, err
	}
	for _, e := range staged.Entries {
		if strings.EqualFold(e.Domain, o.Domain) && e.State == "staged" {
			d["staged_import_id"] = e.ID
			return d, nil
		}
	}
	return nil, &Error{Service: CheapInboxes, Msg: o.Domain + " is not at CheapInboxes; connect it first"}
}

func (c *cheapInboxes) Quote(ctx context.Context, o Order) (Quote, error) {
	if o.Kind == KindConnect {
		return Quote{}, &Error{Service: CheapInboxes, Msg: "staging a domain is free at CheapInboxes; connect it, then buy its mailboxes"}
	}
	d, err := c.cartDomain(ctx, o)
	if err != nil {
		return Quote{}, err
	}
	d["mailboxes"] = map[string]any{"provider": ciMailboxProvider, "count": len(o.People)}
	var res struct {
		Quote struct {
			Totals struct {
				TotalToday     int64 `json:"total_today_cents"`
				MailboxMonthly int64 `json:"mailbox_monthly_cents"`
			} `json:"totals"`
		} `json:"quote"`
	}
	if err := c.call(ctx, http.MethodPost, "/v1/orders/quote", nil, map[string]any{"cart": map[string]any{"domains": []any{d}}}, &res); err != nil {
		return Quote{}, err
	}
	t := res.Quote.Totals
	what := fmt.Sprintf("%d Google Workspace mailbox(es) on %s", len(o.People), o.Domain)
	if o.Kind == KindDomain {
		what = "register " + o.Domain + " with " + what
	}
	summary := fmt.Sprintf("CheapInboxes: %s. %s today, charged to the card on file; then %s a month for the mailboxes.", what, dollars(t.TotalToday), dollars(t.MailboxMonthly))
	return Quote{PriceCents: &t.TotalToday, Currency: "USD", Summary: summary}, nil
}

func dollars(cents int64) string {
	return "$" + strconv.FormatInt(cents/100, 10) + "." + fmt.Sprintf("%02d", cents%100)
}

func (c *cheapInboxes) Buy(ctx context.Context, o Order, idempotencyKey string) (Purchase, error) {
	if o.Kind == KindConnect {
		return Purchase{}, &Error{Service: CheapInboxes, Msg: "staging a domain is free at CheapInboxes; connect it instead"}
	}
	d, err := c.cartDomain(ctx, o)
	if err != nil {
		return Purchase{}, err
	}
	accounts := make([]map[string]string, 0, len(o.People))
	for _, p := range o.People {
		local, _, _ := strings.Cut(p.Address(o.Domain), "@")
		accounts = append(accounts, map[string]string{"first_name": p.FirstName, "last_name": p.LastName, "email_prefix": local})
	}
	d["mailboxes"] = map[string]any{"provider": ciMailboxProvider, "accounts": accounts}
	var res struct {
		OrderID      string      `json:"order_id"`
		TotalCharged int64       `json:"total_charged_cents"`
		Domains      []ciDomain  `json:"domains"`
		Mailboxes    []ciMailbox `json:"mailboxes"`
	}
	headers := map[string]string{"Idempotency-Key": idempotencyKey}
	if err := c.call(ctx, http.MethodPost, "/v1/orders/checkout", headers, map[string]any{"domains": []any{d}}, &res); err != nil {
		return Purchase{}, err
	}
	out := Purchase{
		Summary:      fmt.Sprintf("CheapInboxes order %s: charged %s.", res.OrderID, dollars(res.TotalCharged)),
		ChargedCents: &res.TotalCharged,
		Connection:   Connection{Domain: o.Domain, Note: "CheapInboxes is setting up the domain and its mailboxes. When it lists nameservers for the domain, set them at the registrar."},
	}
	for _, dom := range res.Domains {
		if strings.EqualFold(dom.Domain, o.Domain) {
			out.Connection.Status, out.Connection.Nameservers = dom.Status, dom.Nameservers
		}
	}
	for _, m := range res.Mailboxes {
		out.Mailboxes = append(out.Mailboxes, m.mailbox())
	}
	return out, nil
}

// Delete cancels the mailbox at the end of its billing period; CheapInboxes
// keeps it working until then.
func (c *cheapInboxes) Delete(ctx context.Context, mb Mailbox) error {
	return c.call(ctx, http.MethodPost, "/v1/mailboxes/"+url.PathEscape(mb.ID)+"/cancel", nil, map[string]string{"reason": "user_request"}, nil)
}
