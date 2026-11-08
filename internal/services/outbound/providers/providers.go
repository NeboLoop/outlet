// Package providers talks to the mailbox services a brand's mailbox pool can
// get its domains and mailboxes from: Maildoso, CheapInboxes and Infraforge.
// Each is one Provider over the service's own published API. The package
// only speaks to the service; the outbound service decides what to do with
// the answers (it imports mailboxes through its one login-tested path).
//
// Nothing here spends money except Buy, and the outbound service calls Buy
// only for an unexpired quote the person confirmed.
package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// The mailbox services. manual means the pool's mailboxes are connected by
// hand; it has no Provider.
const (
	Manual       = "manual"
	Maildoso     = "maildoso"
	CheapInboxes = "cheapinboxes"
	Infraforge   = "infraforge"
)

// Services lists the services a pool can pick, manual first.
var Services = []string{Manual, Maildoso, CheapInboxes, Infraforge}

// Label is a service as people say it.
func Label(service string) string {
	switch service {
	case Maildoso:
		return "Maildoso"
	case CheapInboxes:
		return "CheapInboxes"
	case Infraforge:
		return "Infraforge"
	}
	return "Manual"
}

// DefaultBase is each service's API base URL.
var DefaultBase = map[string]string{
	Maildoso:     "https://api.maildoso.com",
	CheapInboxes: "https://api.cheapinboxes.com",
	Infraforge:   "https://api.infraforge.ai/public",
}

// What a purchase is.
const (
	KindDomain    = "domain"    // register a new domain
	KindConnect   = "connect"   // connect a domain the brand owns, where the service bills for it
	KindMailboxes = "mailboxes" // create mailboxes by name on a domain
)

// Person is one mailbox to create. Local is the part before the @; empty
// makes first.last.
type Person struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Local     string `json:"local,omitempty"`
}

// Address is the person's address on a domain.
func (p Person) Address(domain string) string {
	local := strings.ToLower(strings.TrimSpace(p.Local))
	if local == "" {
		local = strings.ToLower(strings.Join(strings.Fields(p.FirstName+"."+p.LastName), ""))
	}
	return local + "@" + domain
}

// Order is something to buy: a domain, a connection, or mailboxes on a domain.
type Order struct {
	Kind   string   `json:"kind"`
	Domain string   `json:"domain"`
	People []Person `json:"people,omitempty"`
}

// Quote is what an order costs. PriceCents is nil when the service's API
// gives no price.
type Quote struct {
	PriceCents *int64 `json:"price_cents,omitempty"`
	Currency   string `json:"currency,omitempty"`
	Summary    string `json:"summary"`
}

// Domain is one of the service's domains.
type Domain struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Status      string   `json:"status"`
	Nameservers []string `json:"nameservers,omitempty"`
}

// Login is how a mailbox signs in to SMTP and IMAP.
type Login struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	IMAPHost     string
	IMAPPort     int
	IMAPUsername string
	IMAPPassword string
}

// Mailbox is one of the service's mailboxes. Login is set when the service
// handed it over with the mailbox.
type Mailbox struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Status    string `json:"status"`
	Ready     bool   `json:"ready"`
	Login     *Login `json:"-"`
}

// Connection is what connecting a domain needs from the person.
type Connection struct {
	Domain      string   `json:"domain"`
	Nameservers []string `json:"nameservers,omitempty"`
	Status      string   `json:"status,omitempty"`
	Note        string   `json:"note"`
}

// Purchase is what a buy did.
type Purchase struct {
	Summary      string     `json:"summary"`
	ChargedCents *int64     `json:"charged_cents,omitempty"`
	Mailboxes    []Mailbox  `json:"mailboxes,omitempty"`
	Connection   Connection `json:"connection"`
}

// Provider is one mailbox service, signed in with the brand's API key.
type Provider interface {
	// Domains and Mailboxes list what the brand has at the service.
	Domains(ctx context.Context) ([]Domain, error)
	Mailboxes(ctx context.Context) ([]Mailbox, error)
	// Login fetches one mailbox's SMTP and IMAP login. NotReady says the
	// service hasn't finished it yet.
	Login(ctx context.Context, mailbox Mailbox) (Login, error)
	// Connect starts using a domain the brand already owns, where the
	// service does that free; a service that bills for it refuses with
	// ErrConnectIsPurchase.
	Connect(ctx context.Context, domain string) (Connection, error)
	// Quote prices an order without buying anything.
	Quote(ctx context.Context, o Order) (Quote, error)
	// Buy places an order. It spends money.
	Buy(ctx context.Context, o Order, idempotencyKey string) (Purchase, error)
	// Delete deletes or cancels one mailbox.
	Delete(ctx context.Context, mailbox Mailbox) error
}

// ErrConnectIsPurchase says the service bills for connecting a domain, so
// it goes through a quote and a confirmed buy (kind connect).
var ErrConnectIsPurchase = errors.New("this service bills for connecting a domain")

// NotReady says a mailbox's login isn't available yet.
type NotReady struct{ Reason string }

func (e *NotReady) Error() string { return e.Reason }

// Error is a service's refusal, in words a person can act on. It never
// carries the API key.
type Error struct {
	Service string
	Status  int
	Msg     string
}

func (e *Error) Error() string {
	if e.Status == 0 {
		return Label(e.Service) + ": " + e.Msg
	}
	return fmt.Sprintf("%s answered HTTP %d: %s", Label(e.Service), e.Status, e.Msg)
}

// New signs in to a service with the brand's key. base is the API base URL;
// empty uses the service's own.
func New(service, key, base string) (Provider, error) {
	if base == "" {
		base = DefaultBase[service]
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, &Error{Service: service, Msg: "no API key is set for this pool's mailbox service"}
	}
	c := &client{service: service, base: strings.TrimSuffix(base, "/"), http: &http.Client{Timeout: 30 * time.Second}}
	switch service {
	case Maildoso:
		c.auth = "Bearer " + key
		return &maildoso{c}, nil
	case CheapInboxes:
		c.auth = "Bearer " + key
		return &cheapInboxes{c}, nil
	case Infraforge:
		c.auth = key
		return &infraforge{c}, nil
	}
	return nil, &Error{Service: service, Msg: "unknown mailbox service; use maildoso, cheapinboxes or infraforge"}
}

// client is one service's HTTP API.
type client struct {
	service string
	base    string
	auth    string
	http    *http.Client
}

// call sends one request and decodes the JSON answer into out (when not nil).
func (c *client) call(ctx context.Context, method, path string, headers map[string]string, in, out any) error {
	var body io.Reader
	if in != nil {
		raw, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.auth)
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return &Error{Service: c.service, Msg: "could not reach the API: " + scrub(err.Error(), c.auth)}
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	switch {
	case resp.StatusCode == http.StatusUnauthorized || (resp.StatusCode == http.StatusForbidden && len(raw) == 0):
		return &Error{Service: c.service, Status: resp.StatusCode, Msg: "the API key was refused; check it in the pool's settings"}
	case resp.StatusCode >= 300:
		return &Error{Service: c.service, Status: resp.StatusCode, Msg: scrub(errorText(raw), c.auth)}
	}
	if out != nil && len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return &Error{Service: c.service, Status: resp.StatusCode, Msg: "unexpected answer from " + method + " " + path}
		}
	}
	return nil
}

// errorText pulls the message out of a service's error body.
func errorText(raw []byte) string {
	var shapes struct {
		Error   json.RawMessage `json:"error"`
		Message string          `json:"message"`
		Detail  json.RawMessage `json:"detail"`
	}
	if json.Unmarshal(raw, &shapes) == nil {
		var nested struct {
			Message string `json:"message"`
		}
		var s string
		switch {
		case json.Unmarshal(shapes.Error, &nested) == nil && nested.Message != "":
			return nested.Message
		case json.Unmarshal(shapes.Error, &s) == nil && s != "":
			return s
		case shapes.Message != "":
			return shapes.Message
		case len(shapes.Detail) > 0:
			if json.Unmarshal(shapes.Detail, &s) == nil {
				return s
			}
			return string(shapes.Detail)
		}
	}
	text := strings.TrimSpace(string(raw))
	if len(text) > 300 {
		text = text[:300]
	}
	if text == "" {
		text = "no details"
	}
	return text
}

// scrub removes the key from text that came back from the service.
func scrub(text, auth string) string {
	key := strings.TrimPrefix(auth, "Bearer ")
	if key == "" {
		return text
	}
	return strings.ReplaceAll(text, key, "[key]")
}

func splitDomain(domain string) (sld, tld string) {
	i := strings.Index(domain, ".")
	if i < 0 {
		return domain, ""
	}
	return domain[:i], domain[i+1:]
}

func nonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
