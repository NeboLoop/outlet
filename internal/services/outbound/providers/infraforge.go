package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// infraforge is Infraforge's public API (api.infraforge.ai/public/swagger):
// the key itself in the Authorization header. Mailboxes are bought by name
// on a domain; connecting a domain the brand owns moves its DNS to
// Infraforge and is invoiced, so both go through a quote and a confirmed
// buy. Its API has no price lookup for either.
type infraforge struct{ *client }

type ifCredentials struct {
	IMAPHost     string `json:"imapHost"`
	IMAPPassword string `json:"imapPassword"`
	IMAPPort     int    `json:"imapPort"`
	IMAPUsername string `json:"imapUsername"`
	SMTPHost     string `json:"smtpHost"`
	SMTPPassword string `json:"smtpPassword"`
	SMTPPort     int    `json:"smtpPort"`
	SMTPUsername string `json:"smtpUsername"`
}

type ifMailbox struct {
	ID          string         `json:"id"`
	Email       string         `json:"email"`
	FirstName   string         `json:"firstName"`
	LastName    string         `json:"lastName"`
	Status      string         `json:"status"`
	Credentials *ifCredentials `json:"credentials"`
}

func (m ifMailbox) mailbox() Mailbox {
	mb := Mailbox{ID: m.ID, Email: strings.ToLower(m.Email), FirstName: m.FirstName, LastName: m.LastName, Status: m.Status, Ready: m.Status == "active"}
	if c := m.Credentials; c != nil && c.SMTPHost != "" && c.IMAPHost != "" && c.SMTPPassword != "" {
		mb.Login = &Login{
			SMTPHost: c.SMTPHost, SMTPPort: c.SMTPPort, SMTPUsername: nonEmpty(c.SMTPUsername, mb.Email), SMTPPassword: c.SMTPPassword,
			IMAPHost: c.IMAPHost, IMAPPort: c.IMAPPort, IMAPUsername: nonEmpty(c.IMAPUsername, mb.Email), IMAPPassword: nonEmpty(c.IMAPPassword, c.SMTPPassword),
		}
	}
	return mb
}

// many reads a list the swagger file describes as a single object: an array
// of them, or one.
func many[T any](raw json.RawMessage) ([]T, error) {
	raw = json.RawMessage(strings.TrimSpace(string(raw)))
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	if raw[0] == '[' {
		var out []T
		return out, json.Unmarshal(raw, &out)
	}
	var one T
	if err := json.Unmarshal(raw, &one); err != nil {
		return nil, err
	}
	return []T{one}, nil
}

func (f *infraforge) list(ctx context.Context, path string) (json.RawMessage, error) {
	var raw json.RawMessage
	err := f.call(ctx, http.MethodGet, path, nil, nil, &raw)
	return raw, err
}

func (f *infraforge) Domains(ctx context.Context) ([]Domain, error) {
	raw, err := f.list(ctx, "/domains")
	if err != nil {
		return nil, err
	}
	list, err := many[struct {
		ID     string `json:"id"`
		SLD    string `json:"sld"`
		TLD    string `json:"tld"`
		Status string `json:"status"`
	}](raw)
	if err != nil {
		return nil, &Error{Service: Infraforge, Msg: "unexpected answer from GET /domains"}
	}
	out := make([]Domain, 0, len(list))
	for _, d := range list {
		name := strings.ToLower(d.SLD)
		if d.TLD != "" {
			name += "." + strings.ToLower(strings.TrimPrefix(d.TLD, "."))
		}
		out = append(out, Domain{ID: d.ID, Name: name, Status: d.Status})
	}
	return out, nil
}

func (f *infraforge) Mailboxes(ctx context.Context) ([]Mailbox, error) {
	raw, err := f.list(ctx, "/mailboxes?with_credentials=true")
	if err != nil {
		return nil, err
	}
	list, err := many[ifMailbox](raw)
	if err != nil {
		return nil, &Error{Service: Infraforge, Msg: "unexpected answer from GET /mailboxes"}
	}
	out := make([]Mailbox, 0, len(list))
	for _, m := range list {
		out = append(out, m.mailbox())
	}
	return out, nil
}

func (f *infraforge) Login(ctx context.Context, mb Mailbox) (Login, error) {
	if mb.Login != nil {
		return *mb.Login, nil
	}
	raw, err := f.list(ctx, "/mailboxes/"+url.PathEscape(mb.ID)+"?with_credentials=true")
	if err != nil {
		return Login{}, err
	}
	list, err := many[ifMailbox](raw)
	if err != nil {
		return Login{}, &Error{Service: Infraforge, Msg: "unexpected answer from GET /mailboxes/{id}"}
	}
	for _, m := range list {
		if got := m.mailbox(); got.Login != nil {
			return *got.Login, nil
		}
	}
	return Login{}, &NotReady{Reason: "Infraforge has no SMTP and IMAP login for " + mb.Email + " yet (status " + mb.Status + "); sync again later"}
}

// Connect: Infraforge invoices the move of a domain's DNS, so it is a
// purchase (kind connect).
func (f *infraforge) Connect(context.Context, string) (Connection, error) {
	return Connection{}, ErrConnectIsPurchase
}

func (f *infraforge) nameservers(ctx context.Context) ([]string, error) {
	var ns struct {
		Nameservers []string `json:"nameservers"`
	}
	err := f.call(ctx, http.MethodGet, "/domains/transfer-ns-info", nil, nil, &ns)
	return ns.Nameservers, err
}

func (f *infraforge) Quote(ctx context.Context, o Order) (Quote, error) {
	switch o.Kind {
	case KindConnect:
		ns, err := f.nameservers(ctx)
		if err != nil {
			return Quote{}, err
		}
		return Quote{Summary: fmt.Sprintf("Connect %s to Infraforge: Infraforge runs its DNS (nameservers %s) and invoices the connection. Infraforge's API gives no price before the invoice.", o.Domain, strings.Join(ns, ", "))}, nil
	case KindMailboxes:
		return Quote{Summary: fmt.Sprintf("Buy %d mailbox(es) on %s at Infraforge, invoiced to your Infraforge account. Infraforge's API gives no price before the invoice.", len(o.People), o.Domain)}, nil
	case KindDomain:
		return Quote{}, &Error{Service: Infraforge, Msg: "registering a domain at Infraforge needs the registrant's contact details, which Outlet doesn't hold. Buy it in Infraforge's app; it then shows here."}
	}
	return Quote{}, &Error{Service: Infraforge, Msg: "unknown order kind " + o.Kind}
}

type ifInvoice struct {
	AmountPaid int64 `json:"amountPaid"`
	Total      int64 `json:"total"`
}

func (f *infraforge) Buy(ctx context.Context, o Order, _ string) (Purchase, error) {
	switch o.Kind {
	case KindConnect:
		raw, err := f.list(ctx, "/workspaces")
		if err != nil {
			return Purchase{}, err
		}
		workspaces, err := many[struct {
			ID string `json:"id"`
		}](raw)
		if err != nil || len(workspaces) == 0 || workspaces[0].ID == "" {
			return Purchase{}, &Error{Service: Infraforge, Msg: "the account has no workspace; create one in Infraforge first"}
		}
		var res struct {
			Domains []struct {
				Domain      string   `json:"domain"`
				Nameservers []string `json:"nameservers"`
			} `json:"domains"`
			Invoice ifInvoice `json:"invoice"`
		}
		body := map[string]any{"domains": []string{o.Domain}, "workspaceId": workspaces[0].ID}
		if err := f.call(ctx, http.MethodPost, "/domains/transfer", nil, body, &res); err != nil {
			return Purchase{}, err
		}
		out := Purchase{
			Summary:      fmt.Sprintf("Infraforge is connecting %s; invoiced %s.", o.Domain, dollars(res.Invoice.Total)),
			ChargedCents: &res.Invoice.AmountPaid,
			Connection:   Connection{Domain: o.Domain, Note: "Set the domain's nameservers at its registrar to the ones listed; Infraforge then runs its DNS."},
		}
		for _, d := range res.Domains {
			if strings.EqualFold(d.Domain, o.Domain) {
				out.Connection.Nameservers = d.Nameservers
			}
		}
		return out, nil
	case KindMailboxes:
		type mb struct {
			Email     string `json:"email"`
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
		}
		mbs := make([]mb, 0, len(o.People))
		for _, p := range o.People {
			mbs = append(mbs, mb{Email: p.Address(o.Domain), FirstName: p.FirstName, LastName: p.LastName})
		}
		var res struct {
			Invoice   ifInvoice   `json:"invoice"`
			Mailboxes []ifMailbox `json:"mailboxes"`
		}
		body := map[string]any{"domains": []any{map[string]any{"domain": o.Domain, "mailboxes": mbs}}}
		if err := f.call(ctx, http.MethodPost, "/mailboxes", nil, body, &res); err != nil {
			return Purchase{}, err
		}
		out := Purchase{Summary: fmt.Sprintf("Infraforge bought %d mailbox(es) on %s; invoiced %s.", len(res.Mailboxes), o.Domain, dollars(res.Invoice.Total)), ChargedCents: &res.Invoice.AmountPaid}
		for _, m := range res.Mailboxes {
			out.Mailboxes = append(out.Mailboxes, m.mailbox())
		}
		return out, nil
	}
	return Purchase{}, &Error{Service: Infraforge, Msg: "Infraforge can't buy a " + o.Kind + " through Outlet"}
}

func (f *infraforge) Delete(ctx context.Context, mb Mailbox) error {
	return f.call(ctx, http.MethodDelete, "/mailboxes/"+url.PathEscape(mb.ID), nil, nil, nil)
}
