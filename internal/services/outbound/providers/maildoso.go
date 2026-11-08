package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// maildoso is Maildoso's API (developers.maildoso.com, openapi-basic.json):
// a personal access token as a Bearer token.
//
// Maildoso's terms (updated 2026-10-22) cap cold email at 15 a day per
// mailbox, after 14 days of warm-up, to business addresses only. The pool's
// limits follow them when the pool picks Maildoso.
type maildoso struct{ *client }

type mdDomain struct {
	ID          int64    `json:"id"`
	DomainName  string   `json:"domain_name"`
	Status      string   `json:"domain_status"`
	NameServers []string `json:"name_servers"`
}

type mdHost struct {
	IMAPHost string `json:"imap_host"`
	SMTPHost string `json:"smtp_host"`
	Port     int    `json:"port"`
}

type mdAccount struct {
	ID        int64   `json:"id"`
	Email     string  `json:"email_account"`
	Password  string  `json:"password"`
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Status    string  `json:"status"`
	IMAP      *mdHost `json:"imap"`
	SMTP      *mdHost `json:"smtp"`
}

type mdResult struct {
	Name    string `json:"name"`
	Result  string `json:"result"`
	Message string `json:"message"`
	ID      *int64 `json:"id"`
}

func (a mdAccount) mailbox() Mailbox {
	mb := Mailbox{ID: strconv.FormatInt(a.ID, 10), Email: strings.ToLower(a.Email), FirstName: a.FirstName, LastName: a.LastName, Status: a.Status, Ready: a.Status == "active"}
	// Google accounts carry no hosts: they connect by OAuth, which a pool can't.
	if a.IMAP != nil && a.SMTP != nil && a.Password != "" {
		mb.Login = &Login{
			SMTPHost: a.SMTP.SMTPHost, SMTPPort: a.SMTP.Port, SMTPUsername: mb.Email, SMTPPassword: a.Password,
			IMAPHost: a.IMAP.IMAPHost, IMAPPort: a.IMAP.Port, IMAPUsername: mb.Email, IMAPPassword: a.Password,
		}
	}
	return mb
}

func (m *maildoso) Domains(ctx context.Context) ([]Domain, error) {
	var list []mdDomain
	if err := m.call(ctx, http.MethodGet, "/v1/user/domains", nil, nil, &list); err != nil {
		return nil, err
	}
	out := make([]Domain, 0, len(list))
	for _, d := range list {
		out = append(out, Domain{ID: strconv.FormatInt(d.ID, 10), Name: strings.ToLower(d.DomainName), Status: d.Status, Nameservers: d.NameServers})
	}
	return out, nil
}

func (m *maildoso) accounts(ctx context.Context, keyword string) ([]mdAccount, error) {
	path := "/v1/user/accounts-lookup"
	if keyword != "" {
		path += "?keyword=" + url.QueryEscape(keyword)
	}
	var page struct {
		Items []mdAccount `json:"items"`
	}
	if err := m.call(ctx, http.MethodGet, path, nil, nil, &page); err != nil {
		return nil, err
	}
	return page.Items, nil
}

func (m *maildoso) Mailboxes(ctx context.Context) ([]Mailbox, error) {
	accounts, err := m.accounts(ctx, "")
	if err != nil {
		return nil, err
	}
	out := make([]Mailbox, 0, len(accounts))
	for _, a := range accounts {
		out = append(out, a.mailbox())
	}
	return out, nil
}

func (m *maildoso) Login(ctx context.Context, mb Mailbox) (Login, error) {
	if mb.Login != nil {
		return *mb.Login, nil
	}
	accounts, err := m.accounts(ctx, mb.Email)
	if err != nil {
		return Login{}, err
	}
	for _, a := range accounts {
		if strings.EqualFold(a.Email, mb.Email) {
			if got := a.mailbox(); got.Login != nil {
				return *got.Login, nil
			}
			return Login{}, &NotReady{Reason: "Maildoso has no SMTP and IMAP login for " + mb.Email + " (a Google account connects by OAuth only)"}
		}
	}
	return Login{}, &NotReady{Reason: "Maildoso doesn't list " + mb.Email + " yet"}
}

// Connect adds a domain the brand owns. Maildoso then runs its DNS: the
// domain's nameservers must point at Maildoso within 24 hours.
func (m *maildoso) Connect(ctx context.Context, domain string) (Connection, error) {
	var res struct {
		Results []mdResult `json:"results"`
	}
	body := map[string]any{"domains": []string{domain}, "providers": []string{Maildoso}}
	if err := m.call(ctx, http.MethodPost, "/v1/user/domains/external", nil, body, &res); err != nil {
		return Connection{}, err
	}
	if err := mdFailed(res.Results); err != nil {
		return Connection{}, err
	}
	conn := Connection{Domain: domain, Note: "Maildoso now runs this domain's DNS, and replaces its MX, SPF, DKIM and DMARC records. Set the domain's nameservers at its registrar to the ones listed within 24 hours; Maildoso marks it ACTIVE, usually within 2 hours."}
	var list []mdDomain
	path := "/v1/user/domains?exact_match=true&keyword=" + url.QueryEscape(domain)
	if err := m.call(ctx, http.MethodGet, path, nil, nil, &list); err != nil {
		return conn, err
	}
	for _, d := range list {
		if strings.EqualFold(d.DomainName, domain) {
			conn.Nameservers, conn.Status = d.NameServers, d.Status
		}
	}
	return conn, nil
}

func mdFailed(results []mdResult) error {
	for _, r := range results {
		if strings.EqualFold(r.Result, "failure") || r.ID == nil {
			return &Error{Service: Maildoso, Msg: nonEmpty(r.Message, r.Name+" was not accepted")}
		}
	}
	return nil
}

func (m *maildoso) Quote(ctx context.Context, o Order) (Quote, error) {
	switch o.Kind {
	case KindDomain:
		var found []struct {
			Name      string  `json:"name"`
			Status    *string `json:"status"`
			Available *string `json:"available"`
			Error     *string `json:"error"`
		}
		if err := m.call(ctx, http.MethodPost, "/v1/user/domains/search", nil, map[string]any{"domains": []string{o.Domain}}, &found); err != nil {
			return Quote{}, err
		}
		for _, f := range found {
			if f.Error != nil && *f.Error != "" {
				return Quote{}, &Error{Service: Maildoso, Msg: o.Domain + ": " + *f.Error}
			}
		}
		return Quote{Summary: "Register " + o.Domain + " at Maildoso. Maildoso's API gives no price; it bills the payment method on your Maildoso account."}, nil
	case KindMailboxes:
		return Quote{Summary: fmt.Sprintf("Create %d mailbox(es) on %s at Maildoso. Maildoso's API gives no price; it bills them on your Maildoso plan.", len(o.People), o.Domain)}, nil
	case KindConnect:
		return Quote{}, &Error{Service: Maildoso, Msg: "connecting a domain is free at Maildoso; connect it instead of buying"}
	}
	return Quote{}, &Error{Service: Maildoso, Msg: "unknown order kind " + o.Kind}
}

func (m *maildoso) Buy(ctx context.Context, o Order, _ string) (Purchase, error) {
	switch o.Kind {
	case KindDomain:
		var res struct {
			Results []mdResult `json:"results"`
		}
		if err := m.call(ctx, http.MethodPost, "/v1/user/domains", nil, map[string]any{"domains": []string{o.Domain}}, &res); err != nil {
			return Purchase{}, err
		}
		if err := mdFailed(res.Results); err != nil {
			return Purchase{}, err
		}
		return Purchase{Summary: "Maildoso is registering " + o.Domain + " and runs its DNS.", Connection: Connection{Domain: o.Domain, Note: "Maildoso registered the domain and runs its DNS."}}, nil
	case KindMailboxes:
		type insert struct {
			Email     string `json:"email_account"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
		}
		body := make([]insert, 0, len(o.People))
		for _, p := range o.People {
			body = append(body, insert{Email: p.Address(o.Domain), FirstName: p.FirstName, LastName: p.LastName})
		}
		var res struct {
			Results []mdAccount `json:"results"`
		}
		if err := m.call(ctx, http.MethodPost, "/v1/user/accounts", nil, body, &res); err != nil {
			return Purchase{}, err
		}
		out := Purchase{Summary: fmt.Sprintf("Maildoso created %d mailbox(es) on %s.", len(res.Results), o.Domain)}
		for _, a := range res.Results {
			out.Mailboxes = append(out.Mailboxes, a.mailbox())
		}
		return out, nil
	}
	return Purchase{}, &Error{Service: Maildoso, Msg: "Maildoso can't buy a " + o.Kind}
}

// Delete deletes the account. Maildoso deletes its own accounts at once,
// with every message in them.
func (m *maildoso) Delete(ctx context.Context, mb Mailbox) error {
	id, err := strconv.ParseInt(mb.ID, 10, 64)
	if err != nil {
		return &Error{Service: Maildoso, Msg: "not a Maildoso account id: " + mb.ID}
	}
	return m.call(ctx, http.MethodDelete, "/v1/user/accounts", nil, []int64{id}, nil)
}
