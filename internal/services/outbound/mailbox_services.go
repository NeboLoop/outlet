package outbound

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/services/outbound/providers"

	"github.com/google/uuid"
)

// A mailbox pool gets its domains and mailboxes from its connections: any
// number of mailbox services side by side, several of one service among
// them (two Maildoso accounts). A manual connection is domains and
// mailboxes added by hand (Google Workspace, Microsoft 365, any SMTP and
// IMAP host); Maildoso, CheapInboxes and Infraforge are reached with the
// brand's API key there. Each outreach domain belongs to one connection of
// its pool, and its mailboxes come from that connection. The pool sends from
// all its mailboxes, whatever connection they came from, under its limits.
// A mailbox made at a service joins the pool the one way every mailbox does,
// addMailbox: its SMTP and IMAP logins are tested, encrypted and saved on
// one of the connection's domains.
//
// Buying is two steps. A quote prices an order at one connection (or says
// the service's API gives no price); a buy spends a quote of that same
// connection, unexpired and unused, only with confirm. Nothing else spends
// money.

const (
	quoteLife     = 30 * time.Minute
	deletedPrefix = "Deleted:" // a mailbox deleted at its service; it never resumes
	// dnsHostService marks a domain whose DNS its mailbox service runs.
	dnsHostService = "provider"
)

// localPart is a mailbox address's part before the @ that every service takes.
var localPart = regexp.MustCompile(`^[a-z0-9]([a-z0-9._-]*[a-z0-9])?$`)

// ConnectionLimits are the limits a connection's mailboxes send under in
// place of the pool's; each empty uses the pool's. A mailbox's own daily
// limit still beats the connection's.
type ConnectionLimits struct {
	DailyLimit  *int64 `json:"daily_limit,omitempty"`
	WarmupStart *int64 `json:"warmup_start,omitempty"` // the first day's allowance
	WarmupStep  *int64 `json:"warmup_step,omitempty"`  // added each day
}

// serviceLimits are a new connection's limits under its service's terms;
// empty uses the pool's. Maildoso's terms (updated 2026-10-22) allow cold
// email at 15 a day per mailbox only after 14 days of warm-up, to business
// addresses only: so 15 a day, warming up from 5 by 1 a day.
func serviceLimits(service string) ConnectionLimits {
	if service == providers.Maildoso {
		return ConnectionLimits{DailyLimit: ptrInt(15), WarmupStart: ptrInt(5), WarmupStep: ptrInt(1)}
	}
	return ConnectionLimits{}
}

func ptrInt(n int64) *int64 { return &n }

func (l ConnectionLimits) check() error {
	if l.DailyLimit != nil && *l.DailyLimit < 1 {
		return refuse("A connection's daily limit is at least 1; leave it empty to use the pool's.")
	}
	if l.WarmupStart != nil && *l.WarmupStart < 0 || l.WarmupStep != nil && *l.WarmupStep < 0 {
		return refuse("A connection's warm-up start and step can't be negative; leave them empty to use the pool's.")
	}
	return nil
}

func ptrOf(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	return &v.Int64
}

// Connection is one of a pool's mailbox service connections. The key is
// never shown, only whether it is set.
type Connection struct {
	ID      string `json:"id"`
	PoolID  string `json:"pool_id"`
	Service string `json:"service"`
	Name    string `json:"name"`
	KeySet  bool   `json:"key_set"`
	ConnectionLimits
	Domains   []ConnectionDomain `json:"domains"`
	Mailboxes int                `json:"mailboxes"`
	CreatedAt string             `json:"created_at"`
}

// ConnectionDomain is one of a connection's outreach domains.
type ConnectionDomain struct {
	ID     string `json:"id"`
	Domain string `json:"domain"`
}

// ConnectionInput adds a connection, or changes one. Adding: PoolID empty
// picks the pool carrying outreach (made when the brand has none); Name
// empty names it after its service; limits all empty take the service's
// terms (Maildoso: 15 a day, warming up from 5 by 1), else are set as given.
// Changing: the service and pool stay; Name empty keeps it; APIKey empty
// keeps the stored key; the limits are set as given, each empty for the
// pool's.
type ConnectionInput struct {
	PoolID  string `json:"pool_id,omitempty"`
	Service string `json:"service,omitempty"`
	Name    string `json:"name,omitempty"`
	APIKey  string `json:"api_key,omitempty"`
	ConnectionLimits
}

// serviceErr turns a mailbox service's refusal into one the caller acts on.
func serviceErr(err error) error {
	var pe *providers.Error
	if errors.As(err, &pe) {
		return refuse("%s", pe.Error())
	}
	return err
}

// mailboxPool is one of the brand's mailbox pools.
func (s *Service) mailboxPool(ctx context.Context, orgID, poolID string) (db.SendingPathway, error) {
	p, err := s.mailer.GetPathway(ctx, orgID, poolID)
	if err != nil {
		return p, err
	}
	if p.Provider != email.ProviderMailboxes {
		return p, refuse("%s is not a mailbox pool", p.Name)
	}
	return p, nil
}

// connection is one of the brand's connections.
func (s *Service) connection(ctx context.Context, orgID, id string) (db.MailboxConnection, error) {
	if id == "" {
		return db.MailboxConnection{}, refuse("connection_id is required: the mailbox service connection")
	}
	c, err := s.db.GetMailboxConnection(ctx, db.GetMailboxConnectionParams{ID: id, OrgID: orgID})
	if err != nil {
		return c, refuse("no such connection")
	}
	return c, nil
}

// provider signs in to a connection's mailbox service.
func (s *Service) provider(c db.MailboxConnection) (providers.Provider, error) {
	if c.Service == providers.Manual {
		return nil, refuse("%s connects domains and mailboxes by hand; it has no mailbox service to reach.", c.Name)
	}
	if len(c.ApiKeyEnc) == 0 || s.crypto == nil {
		return nil, refuse("%s has no %s API key set.", c.Name, providers.Label(c.Service))
	}
	key, err := s.crypto.DecryptString(c.ApiKeyEnc)
	if err != nil {
		return nil, err
	}
	p, err := providers.New(c.Service, key, s.serviceAPI[c.Service])
	return p, serviceErr(err)
}

// checkedKey tries a key against its service, then encrypts it.
func (s *Service) checkedKey(ctx context.Context, service, key string) ([]byte, error) {
	if s.crypto == nil {
		return nil, refuse("ENCRYPTION_KEY is not set on this server, so it can't store an API key.")
	}
	p, err := providers.New(service, key, s.serviceAPI[service])
	if err != nil {
		return nil, serviceErr(err)
	}
	if _, err := p.Domains(ctx); err != nil {
		return nil, serviceErr(err)
	}
	return s.crypto.EncryptString(key)
}

// connectionView reads a connection with its domains and mailbox count.
func (s *Service) connectionView(ctx context.Context, c db.MailboxConnection) (Connection, error) {
	out := Connection{ID: c.ID, PoolID: c.PathwayID, Service: c.Service, Name: c.Name, KeySet: len(c.ApiKeyEnc) > 0, Domains: []ConnectionDomain{}, CreatedAt: c.CreatedAt,
		ConnectionLimits: ConnectionLimits{DailyLimit: ptrOf(c.DailyLimit), WarmupStart: ptrOf(c.WarmupStart), WarmupStep: ptrOf(c.WarmupStep)}}
	id := sql.NullString{String: c.ID, Valid: true}
	domains, err := s.db.ListConnectionDomains(ctx, id)
	if err != nil {
		return out, err
	}
	for _, d := range domains {
		out.Domains = append(out.Domains, ConnectionDomain{ID: d.ID, Domain: d.Domain})
	}
	mbs, err := s.db.ListConnectionMailboxes(ctx, id)
	out.Mailboxes = len(mbs)
	return out, err
}

// ListConnections is the brand's connections, oldest first: one pool's, or
// (poolID empty) every pool's.
func (s *Service) ListConnections(ctx context.Context, org db.Organization, poolID string) ([]Connection, error) {
	all, err := s.db.ListMailboxConnections(ctx, org.ID)
	if err != nil {
		return nil, err
	}
	out := []Connection{}
	for _, c := range all {
		if poolID != "" && c.PathwayID != poolID {
			continue
		}
		v, err := s.connectionView(ctx, c)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// AddConnection adds a mailbox service connection to a pool. A service's
// API key is required and tried against the service before it is saved,
// encrypted; manual takes none.
func (s *Service) AddConnection(ctx context.Context, org db.Organization, in ConnectionInput) (Connection, error) {
	service := strings.ToLower(strings.TrimSpace(in.Service))
	if !slices.Contains(providers.Services, service) {
		return Connection{}, refuse("service must be one of %s", strings.Join(providers.Services, ", "))
	}
	key := strings.TrimSpace(in.APIKey)
	if service == providers.Manual && key != "" {
		return Connection{}, refuse("A manual connection takes no API key: its domains and mailboxes are added by hand.")
	}
	if service != providers.Manual && key == "" {
		return Connection{}, refuse("Enter the %s API key.", providers.Label(service))
	}
	limits := in.ConnectionLimits
	if limits == (ConnectionLimits{}) {
		limits = serviceLimits(service)
	}
	if err := limits.check(); err != nil {
		return Connection{}, err
	}
	var pool db.SendingPathway
	var err error
	if in.PoolID != "" {
		pool, err = s.mailboxPool(ctx, org.ID, in.PoolID)
	} else {
		pool, err = s.mailer.DomainPathway(ctx, org.ID, "outreach", "")
	}
	if err != nil {
		return Connection{}, err
	}
	var keyEnc []byte
	if key != "" {
		if keyEnc, err = s.checkedKey(ctx, service, key); err != nil {
			return Connection{}, err
		}
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = providers.Label(service)
		if service == providers.Manual {
			name = email.ManualConnectionName
		}
	}
	c, err := s.db.CreateMailboxConnection(ctx, db.CreateMailboxConnectionParams{
		ID: uuid.NewString(), OrgID: org.ID, PathwayID: pool.ID, Service: service, Name: name, ApiKeyEnc: keyEnc,
		DailyLimit: nullInt(limits.DailyLimit), WarmupStart: nullInt(limits.WarmupStart), WarmupStep: nullInt(limits.WarmupStep),
	})
	if err != nil {
		return Connection{}, err
	}
	return s.connectionView(ctx, c)
}

// UpdateConnection renames a connection, replaces its API key (tried
// first), and sets its mailboxes' limits: daily limit and warm-up. Its service and pool
// stay; a different service is another connection.
func (s *Service) UpdateConnection(ctx context.Context, org db.Organization, id string, in ConnectionInput) (Connection, error) {
	c, err := s.connection(ctx, org.ID, id)
	if err != nil {
		return Connection{}, err
	}
	if svc := strings.ToLower(strings.TrimSpace(in.Service)); svc != "" && svc != c.Service {
		return Connection{}, refuse("A connection keeps its service; add a %s connection instead.", providers.Label(svc))
	}
	if err := in.ConnectionLimits.check(); err != nil {
		return Connection{}, err
	}
	key := strings.TrimSpace(in.APIKey)
	switch {
	case key != "" && c.Service == providers.Manual:
		return Connection{}, refuse("A manual connection takes no API key.")
	case key != "":
		if c.ApiKeyEnc, err = s.checkedKey(ctx, c.Service, key); err != nil {
			return Connection{}, err
		}
	}
	if name := strings.TrimSpace(in.Name); name != "" {
		c.Name = name
	}
	c.DailyLimit, c.WarmupStart, c.WarmupStep = nullInt(in.DailyLimit), nullInt(in.WarmupStart), nullInt(in.WarmupStep)
	if err := s.db.UpdateMailboxConnection(ctx, db.UpdateMailboxConnectionParams{
		Name: c.Name, ApiKeyEnc: c.ApiKeyEnc, DailyLimit: c.DailyLimit, WarmupStart: c.WarmupStart, WarmupStep: c.WarmupStep, ID: c.ID, OrgID: org.ID,
	}); err != nil {
		return Connection{}, err
	}
	return s.connectionView(ctx, c)
}

// RemoveConnection removes a connection that holds no domains; its quotes
// go with it. Nothing changes at the service.
func (s *Service) RemoveConnection(ctx context.Context, org db.Organization, id string) error {
	c, err := s.connection(ctx, org.ID, id)
	if err != nil {
		return err
	}
	domains, err := s.db.ListConnectionDomains(ctx, sql.NullString{String: c.ID, Valid: true})
	if err != nil {
		return err
	}
	if len(domains) > 0 {
		names := make([]string, 0, len(domains))
		for _, d := range domains {
			names = append(names, d.Domain)
		}
		return refuse("%s still holds %s; move them to another connection of this pool first.", c.Name, strings.Join(names, ", "))
	}
	_, err = s.db.DeleteMailboxConnection(ctx, db.DeleteMailboxConnectionParams{ID: c.ID, OrgID: org.ID})
	return err
}

// MoveDomainToConnection moves an outreach domain, with its mailboxes, to
// another connection of its pool. The mailboxes' ids at the old service are
// cleared; a sync of the new connection links them again.
func (s *Service) MoveDomainToConnection(ctx context.Context, org db.Organization, domainID, connectionID string) (Connection, error) {
	d, err := s.db.GetOutreachDomain(ctx, db.GetOutreachDomainParams{ID: domainID, OrgID: org.ID})
	if err != nil {
		return Connection{}, refuse("no such outreach domain")
	}
	c, err := s.connection(ctx, org.ID, connectionID)
	if err != nil {
		return Connection{}, err
	}
	if d.PathwayID.String != c.PathwayID {
		return Connection{}, refuse("%s is in another mailbox pool than %s; a domain moves only between connections of its own pool.", d.Domain, c.Name)
	}
	if d.ConnectionID.String != c.ID {
		err = s.db.ExecTx(ctx, func(q *db.Queries) error {
			if _, err := q.SetDomainConnection(ctx, db.SetDomainConnectionParams{ConnectionID: sql.NullString{String: c.ID, Valid: true}, ID: d.ID, OrgID: org.ID}); err != nil {
				return err
			}
			return q.ClearDomainMailboxServiceRefs(ctx, d.ID)
		})
		if err != nil {
			return Connection{}, err
		}
	}
	return s.connectionView(ctx, c)
}

// ServiceInventory is what the brand has at a connection's mailbox service.
type ServiceInventory struct {
	Service   string              `json:"service"`
	Domains   []providers.Domain  `json:"domains"`
	Mailboxes []providers.Mailbox `json:"mailboxes"`
}

// MailboxServiceInventory lists a connection's domains and mailboxes at its
// service.
func (s *Service) MailboxServiceInventory(ctx context.Context, org db.Organization, connectionID string) (ServiceInventory, error) {
	c, err := s.connection(ctx, org.ID, connectionID)
	if err != nil {
		return ServiceInventory{}, err
	}
	p, err := s.provider(c)
	if err != nil {
		return ServiceInventory{}, err
	}
	out := ServiceInventory{Service: c.Service}
	if out.Domains, err = p.Domains(ctx); err != nil {
		return out, serviceErr(err)
	}
	if out.Mailboxes, err = p.Mailboxes(ctx); err != nil {
		return out, serviceErr(err)
	}
	return out, nil
}

// freeDomain refuses a domain outreach may not use, or one in use other
// than as this connection's outreach domain.
func (s *Service) freeDomain(ctx context.Context, org db.Organization, c db.MailboxConnection, domain string) error {
	if err := s.allowedDomain(ctx, org, domain); err != nil {
		return err
	}
	d, err := s.db.GetAnyDomainIdentity(ctx, domain)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil
	case err != nil:
		return err
	case d.OrgID != org.ID || d.Purpose != "outreach":
		return refuse("%s is already in use (another brand's, or this brand's sending domain); outreach needs a domain of its own.", domain)
	case d.PathwayID.String != c.PathwayID:
		return refuse("%s belongs to another mailbox pool of this brand.", domain)
	case d.ConnectionID.String != c.ID:
		return refuse("%s belongs to another connection of this pool; move it to %s first.", domain, c.Name)
	}
	return nil
}

// addServiceDomain puts a domain whose DNS the service runs on the
// connection.
func (s *Service) addServiceDomain(ctx context.Context, org db.Organization, c db.MailboxConnection, domain string) (db.DomainIdentity, DNSCheck, error) {
	r, err := s.AddDomain(ctx, org, DomainInput{Domain: domain, ConnectionID: c.ID, serviceDNS: true})
	return r.Domain, r.Check, err
}

// ServiceDomain is a domain connected to, or bought at, a mailbox service.
type ServiceDomain struct {
	DomainID   string               `json:"domain_id,omitempty"`
	Connection providers.Connection `json:"connection"`
	Check      DNSCheck             `json:"check"`
}

// ConnectServiceDomain connects a domain the brand owns to a connection's
// mailbox service, where the service does that free, and adds it to the
// connection. It returns what the person must do: the nameservers to set at the
// domain's registrar. (Every supported service runs the domain's DNS
// itself, so there are no records for DigitalOcean to write.)
func (s *Service) ConnectServiceDomain(ctx context.Context, org db.Organization, connectionID, domain string) (ServiceDomain, error) {
	c, err := s.connection(ctx, org.ID, connectionID)
	if err != nil {
		return ServiceDomain{}, err
	}
	domain = normDomain(domain)
	if err := s.freeDomain(ctx, org, c, domain); err != nil {
		return ServiceDomain{}, err
	}
	p, err := s.provider(c)
	if err != nil {
		return ServiceDomain{}, err
	}
	conn, err := p.Connect(ctx, domain)
	if errors.Is(err, providers.ErrConnectIsPurchase) {
		return ServiceDomain{}, refuse("%s bills for connecting a domain: get a quote (kind connect), then buy it with confirm.", providers.Label(c.Service))
	}
	if err != nil {
		return ServiceDomain{}, serviceErr(err)
	}
	d, check, err := s.addServiceDomain(ctx, org, c, domain)
	return ServiceDomain{DomainID: d.ID, Connection: conn, Check: check}, err
}

// QuoteInput asks for a price: a new domain, connecting a domain the brand
// owns (where the service bills for it), or mailboxes by name on one of the
// connection's domains.
type QuoteInput struct {
	Kind   string             `json:"kind"`
	Domain string             `json:"domain"`
	People []providers.Person `json:"people,omitempty"`
}

// ServiceQuote is a price to confirm. PriceCents is empty when the
// service's API gives no price.
type ServiceQuote struct {
	ID         string `json:"id"`
	Service    string `json:"service"`
	Kind       string `json:"kind"`
	Domain     string `json:"domain"`
	Mailboxes  int    `json:"mailboxes"`
	PriceCents *int64 `json:"price_cents,omitempty"`
	PriceKnown bool   `json:"price_known"`
	Currency   string `json:"currency,omitempty"`
	Summary    string `json:"summary"`
	ExpiresAt  string `json:"expires_at"`
}

// order checks an order against the connection before it is quoted or
// bought.
func (s *Service) order(ctx context.Context, org db.Organization, c db.MailboxConnection, in QuoteInput) (providers.Order, error) {
	o := providers.Order{Kind: strings.ToLower(strings.TrimSpace(in.Kind)), Domain: normDomain(in.Domain), People: in.People}
	switch o.Kind {
	case providers.KindDomain, providers.KindConnect:
		if err := s.freeDomain(ctx, org, c, o.Domain); err != nil {
			return o, err
		}
		if o.Kind == providers.KindConnect && len(o.People) > 0 {
			return o, refuse("connect takes no people; buy mailboxes once the domain is connected")
		}
	case providers.KindMailboxes:
		d, err := s.db.GetAnyDomainIdentity(ctx, o.Domain)
		if err != nil || d.OrgID != org.ID || d.Purpose != "outreach" || d.ConnectionID.String != c.ID {
			return o, refuse("%s is not one of %s's domains; connect or buy it there first", o.Domain, c.Name)
		}
		if len(o.People) == 0 {
			return o, refuse("people is required: each mailbox's first_name and last_name")
		}
	default:
		return o, refuse("kind must be domain, connect or mailboxes")
	}
	if len(o.People) > maxBatch {
		return o, refuse("at most %d mailboxes per order", maxBatch)
	}
	seen := map[string]bool{}
	for i, p := range o.People {
		p.FirstName, p.LastName, p.Local = strings.TrimSpace(p.FirstName), strings.TrimSpace(p.LastName), strings.TrimSpace(p.Local)
		if p.FirstName == "" || p.LastName == "" {
			return o, refuse("every mailbox needs a first_name and a last_name")
		}
		addr := p.Address(o.Domain)
		if local, _, _ := strings.Cut(addr, "@"); !localPart.MatchString(local) {
			return o, refuse("%q makes the address %s; pass local, the part before the @ (letters, digits, dots, dashes)", p.FirstName+" "+p.LastName, addr)
		}
		if seen[addr] {
			return o, refuse("%s is in the order twice", addr)
		}
		seen[addr] = true
		if _, err := s.db.GetMailboxByEmail(ctx, addr); err == nil {
			return o, refuse("%s is already a connected mailbox", addr)
		}
		o.People[i] = p
	}
	return o, nil
}

// QuoteService prices an order at a connection's mailbox service. It buys
// nothing; BuyService spends the quote, at the same connection only.
func (s *Service) QuoteService(ctx context.Context, org db.Organization, connectionID string, in QuoteInput) (ServiceQuote, error) {
	c, err := s.connection(ctx, org.ID, connectionID)
	if err != nil {
		return ServiceQuote{}, err
	}
	o, err := s.order(ctx, org, c, in)
	if err != nil {
		return ServiceQuote{}, err
	}
	p, err := s.provider(c)
	if err != nil {
		return ServiceQuote{}, err
	}
	q, err := p.Quote(ctx, o)
	if err != nil {
		return ServiceQuote{}, serviceErr(err)
	}
	raw, err := json.Marshal(o)
	if err != nil {
		return ServiceQuote{}, err
	}
	price := sql.NullInt64{}
	if q.PriceCents != nil {
		price = sql.NullInt64{Int64: *q.PriceCents, Valid: true}
	}
	row, err := s.db.CreateMailboxServiceQuote(ctx, db.CreateMailboxServiceQuoteParams{
		ID: uuid.NewString(), OrgID: org.ID, ConnectionID: c.ID, Service: c.Service, Kind: o.Kind,
		Request: string(raw), PriceCents: price, Currency: q.Currency, Summary: q.Summary, ExpiresAt: ts(s.now().Add(quoteLife)),
	})
	if err != nil {
		return ServiceQuote{}, err
	}
	return ServiceQuote{
		ID: row.ID, Service: row.Service, Kind: row.Kind, Domain: o.Domain, Mailboxes: len(o.People),
		PriceCents: q.PriceCents, PriceKnown: q.PriceCents != nil, Currency: q.Currency, Summary: q.Summary, ExpiresAt: row.ExpiresAt,
	}, nil
}

// BuyResult is what a buy did: the purchase, the domain added to the
// connection,
// and the mailboxes imported (or still being made at the service).
type BuyResult struct {
	Summary      string               `json:"summary"`
	ChargedCents *int64               `json:"charged_cents,omitempty"`
	DomainID     string               `json:"domain_id,omitempty"`
	Connection   providers.Connection `json:"connection"`
	Check        *DNSCheck            `json:"check,omitempty"`
	Imported     []MailboxResult      `json:"imported"`
	Pending      []string             `json:"pending"`
}

// BuyService places the order a quote priced. It needs confirm and an
// unexpired quote of this same connection; a quote buys once.
func (s *Service) BuyService(ctx context.Context, org db.Organization, connectionID, quoteID string, confirm bool) (BuyResult, error) {
	if !confirm {
		return BuyResult{}, refuse("Buying spends money. Read the quote, then buy with its quote_id and confirm: true.")
	}
	c, err := s.connection(ctx, org.ID, connectionID)
	if err != nil {
		return BuyResult{}, err
	}
	q, err := s.db.GetMailboxServiceQuote(ctx, db.GetMailboxServiceQuoteParams{ID: quoteID, OrgID: org.ID})
	if err != nil || q.ConnectionID != c.ID || q.Service != c.Service {
		return BuyResult{}, refuse("no such quote for %s; get a quote there first", c.Name)
	}
	if q.UsedAt.Valid || q.ExpiresAt <= ts(s.now()) {
		return BuyResult{}, refuse("this quote has expired or was already used; get a new quote")
	}
	var in QuoteInput
	if err := json.Unmarshal([]byte(q.Request), &in); err != nil {
		return BuyResult{}, err
	}
	o, err := s.order(ctx, org, c, in)
	if err != nil {
		return BuyResult{}, err
	}
	p, err := s.provider(c)
	if err != nil {
		return BuyResult{}, err
	}
	n, err := s.db.UseMailboxServiceQuote(ctx, db.UseMailboxServiceQuoteParams{Now: sql.NullString{String: ts(s.now()), Valid: true}, ID: q.ID, OrgID: org.ID})
	if err != nil {
		return BuyResult{}, err
	}
	if n == 0 {
		return BuyResult{}, refuse("this quote has expired or was already used; get a new quote")
	}
	bought, err := p.Buy(ctx, o, q.ID)
	if err != nil {
		return BuyResult{}, serviceErr(err)
	}
	out := BuyResult{Summary: bought.Summary, ChargedCents: bought.ChargedCents, Connection: bought.Connection, Imported: []MailboxResult{}, Pending: []string{}}
	if o.Kind != providers.KindMailboxes {
		d, check, err := s.addServiceDomain(ctx, org, c, o.Domain)
		if err != nil {
			return out, err
		}
		out.DomainID, out.Check = d.ID, &check
	}
	for _, mb := range bought.Mailboxes {
		res, pending := s.importServiceMailbox(ctx, org, c, p, mb)
		if pending != "" {
			out.Pending = append(out.Pending, pending)
			continue
		}
		out.Imported = append(out.Imported, res)
	}
	return out, nil
}

// importServiceMailbox brings one of the service's mailboxes into the pool
// through addMailbox, the one path every mailbox takes, on one of the
// connection's domains. pending is set when the service hasn't finished the
// mailbox yet.
func (s *Service) importServiceMailbox(ctx context.Context, org db.Organization, c db.MailboxConnection, p providers.Provider, mb providers.Mailbox) (res MailboxResult, pending string) {
	res = MailboxResult{Email: mb.Email}
	d, err := s.db.GetAnyDomainIdentity(ctx, domainOf(mb.Email))
	if err != nil || d.OrgID != org.ID || d.ConnectionID.String != c.ID {
		res.Error = domainOf(mb.Email) + " is not one of " + c.Name + "'s domains; connect it there first"
		return res, ""
	}
	login, err := p.Login(ctx, mb)
	var nr *providers.NotReady
	if errors.As(err, &nr) {
		return res, mb.Email + ": " + nr.Reason
	}
	if err != nil {
		res.Error = serviceErr(err).Error()
		return res, ""
	}
	name := strings.TrimSpace(mb.FirstName + " " + mb.LastName)
	if name == "" {
		name, _, _ = strings.Cut(mb.Email, "@")
	}
	created, err := s.addMailbox(ctx, org, map[string]db.DomainIdentity{d.Domain: d}, MailboxInput{
		Email: mb.Email, FromName: name,
		SMTPHost: login.SMTPHost, SMTPPort: login.SMTPPort, SMTPUsername: login.SMTPUsername, SMTPPassword: login.SMTPPassword,
		IMAPHost: login.IMAPHost, IMAPPort: login.IMAPPort, IMAPUsername: login.IMAPUsername, IMAPPassword: login.IMAPPassword,
	})
	if err != nil {
		if !mb.Ready {
			return res, mb.Email + ": not active at " + providers.Label(c.Service) + " yet (" + mb.Status + "); " + err.Error()
		}
		res.Error = err.Error()
		return res, ""
	}
	res.ID = created.ID
	if err := s.db.SetMailboxServiceRef(ctx, db.SetMailboxServiceRefParams{ServiceRef: mb.ID, ID: created.ID, OrgID: org.ID}); err != nil {
		res.Error = err.Error()
	}
	return res, ""
}

// PoolMailbox is one of the connection's mailboxes in the pool, in a sync.
type PoolMailbox struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// SyncResult compares a connection's mailboxes in the pool with its
// mailbox service.
type SyncResult struct {
	OnlyAtService []providers.Mailbox `json:"only_at_service"` // the service has them; the pool doesn't
	OnlyInPool    []PoolMailbox       `json:"only_in_pool"`    // the connection's domains have them in the pool; the service doesn't
	Imported      []MailboxResult     `json:"imported"`
	Pending       []string            `json:"pending"`
}

// SyncService reports the mailboxes a connection's service has that the
// pool doesn't, and the connection's mailboxes in the pool the service
// lacks. With importNew it imports the service's missing ones that are on
// the connection's domains and ready. A mailbox the service also has gets
// linked to it, so it can be deleted there.
func (s *Service) SyncService(ctx context.Context, org db.Organization, connectionID string, importNew bool) (SyncResult, error) {
	c, err := s.connection(ctx, org.ID, connectionID)
	if err != nil {
		return SyncResult{}, err
	}
	p, err := s.provider(c)
	if err != nil {
		return SyncResult{}, err
	}
	theirs, err := p.Mailboxes(ctx)
	if err != nil {
		return SyncResult{}, serviceErr(err)
	}
	ours, err := s.db.ListConnectionMailboxes(ctx, sql.NullString{String: c.ID, Valid: true})
	if err != nil {
		return SyncResult{}, err
	}
	out := SyncResult{OnlyAtService: []providers.Mailbox{}, OnlyInPool: []PoolMailbox{}, Imported: []MailboxResult{}, Pending: []string{}}
	byEmail := map[string]db.Mailbox{}
	for _, mb := range ours {
		byEmail[mb.Email] = mb
	}
	atService := map[string]bool{}
	for _, mb := range theirs {
		atService[mb.Email] = true
		if have, ok := byEmail[mb.Email]; ok {
			if have.ServiceRef != mb.ID {
				if err := s.db.SetMailboxServiceRef(ctx, db.SetMailboxServiceRefParams{ServiceRef: mb.ID, ID: have.ID, OrgID: org.ID}); err != nil {
					return out, err
				}
			}
			continue
		}
		if !importNew {
			out.OnlyAtService = append(out.OnlyAtService, mb)
			continue
		}
		res, pending := s.importServiceMailbox(ctx, org, c, p, mb)
		switch {
		case pending != "":
			out.Pending = append(out.Pending, pending)
			out.OnlyAtService = append(out.OnlyAtService, mb)
		case res.Error != "":
			out.Imported = append(out.Imported, res)
			out.OnlyAtService = append(out.OnlyAtService, mb)
		default:
			out.Imported = append(out.Imported, res)
		}
	}
	for _, mb := range ours {
		if !atService[mb.Email] {
			out.OnlyInPool = append(out.OnlyInPool, PoolMailbox{ID: mb.ID, Email: mb.Email})
		}
	}
	return out, nil
}

// DeleteServiceMailbox deletes (Maildoso, Infraforge) or cancels at the end
// of its billing period (CheapInboxes) one of a connection's mailboxes at
// its service, and pauses it in the pool for good: its threads and replies
// stay.
func (s *Service) DeleteServiceMailbox(ctx context.Context, org db.Organization, connectionID, mailboxID string) error {
	c, err := s.connection(ctx, org.ID, connectionID)
	if err != nil {
		return err
	}
	mb, err := s.db.GetMailbox(ctx, db.GetMailboxParams{ID: mailboxID, OrgID: org.ID})
	if err != nil {
		return refuse("no such mailbox")
	}
	if d, err := s.db.GetDomainIdentity(ctx, mb.DomainID); err != nil || d.ConnectionID.String != c.ID {
		return refuse("%s is not on one of %s's domains", mb.Email, c.Name)
	}
	p, err := s.provider(c)
	if err != nil {
		return err
	}
	ref := providers.Mailbox{ID: mb.ServiceRef, Email: mb.Email}
	if ref.ID == "" {
		theirs, err := p.Mailboxes(ctx)
		if err != nil {
			return serviceErr(err)
		}
		for _, t := range theirs {
			if t.Email == mb.Email {
				ref.ID = t.ID
			}
		}
		if ref.ID == "" {
			return refuse("%s doesn't have %s", providers.Label(c.Service), mb.Email)
		}
	}
	if err := p.Delete(ctx, ref); err != nil {
		return serviceErr(err)
	}
	return s.db.SetMailboxStatus(ctx, db.SetMailboxStatusParams{
		Status: "paused", PausedReason: fmt.Sprintf("%s at %s on %s", deletedPrefix, providers.Label(c.Service), s.now().UTC().Format(time.DateOnly)), ID: mb.ID,
	})
}
