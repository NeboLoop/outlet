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

// A mailbox pool gets its domains and mailboxes by hand (manual) or from a
// mailbox service: Maildoso, CheapInboxes or Infraforge, reached with the
// brand's API key there. A mailbox made at the service joins the pool the
// one way every mailbox does, addMailbox: its SMTP and IMAP logins are
// tested, encrypted and saved on one of the pool's domains.
//
// Buying is two steps. A quote prices an order (or says the service's API
// gives no price); a buy spends a quote, unexpired and unused, only with
// confirm. Nothing else spends money.

const (
	quoteLife     = 30 * time.Minute
	deletedPrefix = "Deleted:" // a mailbox deleted at its service; it never resumes
	// dnsHostService marks a domain whose DNS its mailbox service runs.
	dnsHostService = "provider"
)

// localPart is a mailbox address's part before the @ that every service takes.
var localPart = regexp.MustCompile(`^[a-z0-9]([a-z0-9._-]*[a-z0-9])?$`)

// ServiceLimits are a pool's limits under a mailbox service's terms.
// Maildoso's terms (updated 2026-10-22) cap cold email at 15 a day per
// mailbox after 14 days of warm-up, to business addresses only: so 15 a day,
// warming up from 5 by 1 a day. The others keep the pool's normal limits.
func ServiceLimits(service string) email.Limits {
	l := email.DefaultLimits(email.ProviderMailboxes)
	if service == providers.Maildoso {
		l.DailyLimit, l.WarmupStart, l.WarmupStep = 15, 5, 1
	}
	return l
}

// ServiceStatus is a pool's mailbox service. The key is never shown, only
// whether it is set.
type ServiceStatus struct {
	PoolID  string       `json:"pool_id"`
	Service string       `json:"service"`
	KeySet  bool         `json:"key_set"`
	Limits  email.Limits `json:"limits"`
}

// serviceErr turns a mailbox service's refusal into one the caller acts on.
func serviceErr(err error) error {
	var pe *providers.Error
	if errors.As(err, &pe) {
		return refuse("%s", pe.Error())
	}
	return err
}

// servicePool is one of the brand's mailbox pools.
func (s *Service) servicePool(ctx context.Context, orgID, poolID string) (db.SendingPathway, error) {
	if poolID == "" {
		return db.SendingPathway{}, refuse("pool_id is required: the mailbox pool")
	}
	p, err := s.mailer.GetPathway(ctx, orgID, poolID)
	if err != nil {
		return p, err
	}
	if p.Provider != email.ProviderMailboxes {
		return p, refuse("%s is not a mailbox pool", p.Name)
	}
	return p, nil
}

// provider signs in to the pool's mailbox service.
func (s *Service) provider(pool db.SendingPathway) (providers.Provider, error) {
	if pool.MailboxService == "" || pool.MailboxService == providers.Manual {
		return nil, refuse("%s connects mailboxes by hand. Pick a mailbox service (maildoso, cheapinboxes or infraforge) and enter its API key first.", pool.Name)
	}
	if len(pool.MailboxServiceKeyEnc) == 0 || s.crypto == nil {
		return nil, refuse("%s has no %s API key set.", pool.Name, providers.Label(pool.MailboxService))
	}
	key, err := s.crypto.DecryptString(pool.MailboxServiceKeyEnc)
	if err != nil {
		return nil, err
	}
	p, err := providers.New(pool.MailboxService, key, s.serviceAPI[pool.MailboxService])
	return p, serviceErr(err)
}

// SetMailboxService picks a pool's mailbox service and stores its API key,
// encrypted. An empty key keeps the stored one; manual clears it. Picking a
// different service sets the pool's limits to that service's defaults
// (ServiceLimits), which the brand can change afterwards. A new key is
// tried against the service before it is saved.
func (s *Service) SetMailboxService(ctx context.Context, org db.Organization, poolID, service, key string) (ServiceStatus, error) {
	pool, err := s.servicePool(ctx, org.ID, poolID)
	if err != nil {
		return ServiceStatus{}, err
	}
	service = strings.ToLower(strings.TrimSpace(service))
	if !slices.Contains(providers.Services, service) {
		return ServiceStatus{}, refuse("service must be one of %s", strings.Join(providers.Services, ", "))
	}
	key = strings.TrimSpace(key)
	changed := service != pool.MailboxService
	keyEnc := pool.MailboxServiceKeyEnc
	switch {
	case service == providers.Manual:
		keyEnc = nil
	case key != "":
		if s.crypto == nil {
			return ServiceStatus{}, refuse("ENCRYPTION_KEY is not set on this server, so it can't store an API key.")
		}
		p, err := providers.New(service, key, s.serviceAPI[service])
		if err != nil {
			return ServiceStatus{}, serviceErr(err)
		}
		if _, err := p.Domains(ctx); err != nil {
			return ServiceStatus{}, serviceErr(err)
		}
		if keyEnc, err = s.crypto.EncryptString(key); err != nil {
			return ServiceStatus{}, err
		}
	case changed || len(keyEnc) == 0:
		return ServiceStatus{}, refuse("Enter the %s API key.", providers.Label(service))
	}
	if err := s.db.SetPathwayMailboxService(ctx, db.SetPathwayMailboxServiceParams{MailboxService: service, MailboxServiceKeyEnc: keyEnc, ID: pool.ID, OrgID: org.ID}); err != nil {
		return ServiceStatus{}, err
	}
	limits := email.LimitsOf(pool)
	if changed {
		limits = ServiceLimits(service)
		if _, err := s.mailer.UpdatePathway(ctx, org.ID, pool.ID, email.PathwayInput{Kinds: email.KindsOf(pool), Limits: &limits}); err != nil {
			return ServiceStatus{}, err
		}
	}
	return ServiceStatus{PoolID: pool.ID, Service: service, KeySet: len(keyEnc) > 0, Limits: limits}, nil
}

// ServiceInventory is what the brand has at the pool's mailbox service.
type ServiceInventory struct {
	Service   string              `json:"service"`
	Domains   []providers.Domain  `json:"domains"`
	Mailboxes []providers.Mailbox `json:"mailboxes"`
}

// MailboxServiceInventory lists the service's domains and mailboxes.
func (s *Service) MailboxServiceInventory(ctx context.Context, org db.Organization, poolID string) (ServiceInventory, error) {
	pool, err := s.servicePool(ctx, org.ID, poolID)
	if err != nil {
		return ServiceInventory{}, err
	}
	p, err := s.provider(pool)
	if err != nil {
		return ServiceInventory{}, err
	}
	out := ServiceInventory{Service: pool.MailboxService}
	if out.Domains, err = p.Domains(ctx); err != nil {
		return out, serviceErr(err)
	}
	if out.Mailboxes, err = p.Mailboxes(ctx); err != nil {
		return out, serviceErr(err)
	}
	return out, nil
}

// freeDomain refuses a domain outreach may not use, or one in use other
// than as this pool's outreach domain.
func (s *Service) freeDomain(ctx context.Context, org db.Organization, pool db.SendingPathway, domain string) error {
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
	case d.PathwayID.String != pool.ID:
		return refuse("%s belongs to another mailbox pool of this brand.", domain)
	}
	return nil
}

// addServiceDomain puts a domain whose DNS the service runs in the pool.
func (s *Service) addServiceDomain(ctx context.Context, org db.Organization, pool db.SendingPathway, domain string) (db.DomainIdentity, DNSCheck, error) {
	r, err := s.AddDomain(ctx, org, DomainInput{Domain: domain, PathwayID: pool.ID, serviceDNS: true})
	return r.Domain, r.Check, err
}

// ServiceDomain is a domain connected to, or bought at, a mailbox service.
type ServiceDomain struct {
	DomainID   string               `json:"domain_id,omitempty"`
	Connection providers.Connection `json:"connection"`
	Check      DNSCheck             `json:"check"`
}

// ConnectServiceDomain connects a domain the brand owns to the pool's
// mailbox service, where the service does that free, and adds it to the
// pool. It returns what the person must do: the nameservers to set at the
// domain's registrar. (Every supported service runs the domain's DNS
// itself, so there are no records for DigitalOcean to write.)
func (s *Service) ConnectServiceDomain(ctx context.Context, org db.Organization, poolID, domain string) (ServiceDomain, error) {
	pool, err := s.servicePool(ctx, org.ID, poolID)
	if err != nil {
		return ServiceDomain{}, err
	}
	domain = normDomain(domain)
	if err := s.freeDomain(ctx, org, pool, domain); err != nil {
		return ServiceDomain{}, err
	}
	p, err := s.provider(pool)
	if err != nil {
		return ServiceDomain{}, err
	}
	conn, err := p.Connect(ctx, domain)
	if errors.Is(err, providers.ErrConnectIsPurchase) {
		return ServiceDomain{}, refuse("%s bills for connecting a domain: get a quote (kind connect), then buy it with confirm.", providers.Label(pool.MailboxService))
	}
	if err != nil {
		return ServiceDomain{}, serviceErr(err)
	}
	d, check, err := s.addServiceDomain(ctx, org, pool, domain)
	return ServiceDomain{DomainID: d.ID, Connection: conn, Check: check}, err
}

// QuoteInput asks for a price: a new domain, connecting a domain the brand
// owns (where the service bills for it), or mailboxes by name on one of the
// pool's domains.
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

// order checks an order against the pool before it is quoted or bought.
func (s *Service) order(ctx context.Context, org db.Organization, pool db.SendingPathway, in QuoteInput) (providers.Order, error) {
	o := providers.Order{Kind: strings.ToLower(strings.TrimSpace(in.Kind)), Domain: normDomain(in.Domain), People: in.People}
	switch o.Kind {
	case providers.KindDomain, providers.KindConnect:
		if err := s.freeDomain(ctx, org, pool, o.Domain); err != nil {
			return o, err
		}
		if o.Kind == providers.KindConnect && len(o.People) > 0 {
			return o, refuse("connect takes no people; buy mailboxes once the domain is connected")
		}
	case providers.KindMailboxes:
		d, err := s.db.GetAnyDomainIdentity(ctx, o.Domain)
		if err != nil || d.OrgID != org.ID || d.Purpose != "outreach" || d.PathwayID.String != pool.ID {
			return o, refuse("%s is not one of this pool's domains; connect or buy it first", o.Domain)
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

// QuoteService prices an order at the pool's mailbox service. It buys
// nothing; BuyService spends the quote.
func (s *Service) QuoteService(ctx context.Context, org db.Organization, poolID string, in QuoteInput) (ServiceQuote, error) {
	pool, err := s.servicePool(ctx, org.ID, poolID)
	if err != nil {
		return ServiceQuote{}, err
	}
	o, err := s.order(ctx, org, pool, in)
	if err != nil {
		return ServiceQuote{}, err
	}
	p, err := s.provider(pool)
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
		ID: uuid.NewString(), OrgID: org.ID, PathwayID: pool.ID, Service: pool.MailboxService, Kind: o.Kind,
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

// BuyResult is what a buy did: the purchase, the domain added to the pool,
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
// unexpired quote of this pool, at the service the pool still uses; a quote
// buys once.
func (s *Service) BuyService(ctx context.Context, org db.Organization, poolID, quoteID string, confirm bool) (BuyResult, error) {
	if !confirm {
		return BuyResult{}, refuse("Buying spends money. Read the quote, then buy with its quote_id and confirm: true.")
	}
	pool, err := s.servicePool(ctx, org.ID, poolID)
	if err != nil {
		return BuyResult{}, err
	}
	q, err := s.db.GetMailboxServiceQuote(ctx, db.GetMailboxServiceQuoteParams{ID: quoteID, OrgID: org.ID})
	if err != nil || q.PathwayID != pool.ID {
		return BuyResult{}, refuse("no such quote for this pool; get a quote first")
	}
	if q.Service != pool.MailboxService {
		return BuyResult{}, refuse("this quote is from %s, but the pool now uses %s; get a new quote", providers.Label(q.Service), providers.Label(pool.MailboxService))
	}
	if q.UsedAt.Valid || q.ExpiresAt <= ts(s.now()) {
		return BuyResult{}, refuse("this quote has expired or was already used; get a new quote")
	}
	var in QuoteInput
	if err := json.Unmarshal([]byte(q.Request), &in); err != nil {
		return BuyResult{}, err
	}
	o, err := s.order(ctx, org, pool, in)
	if err != nil {
		return BuyResult{}, err
	}
	p, err := s.provider(pool)
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
		d, check, err := s.addServiceDomain(ctx, org, pool, o.Domain)
		if err != nil {
			return out, err
		}
		out.DomainID, out.Check = d.ID, &check
	}
	for _, mb := range bought.Mailboxes {
		res, pending := s.importServiceMailbox(ctx, org, pool, p, mb)
		if pending != "" {
			out.Pending = append(out.Pending, pending)
			continue
		}
		out.Imported = append(out.Imported, res)
	}
	return out, nil
}

// importServiceMailbox brings one of the service's mailboxes into the pool
// through addMailbox, the one path every mailbox takes. pending is set when
// the service hasn't finished the mailbox yet.
func (s *Service) importServiceMailbox(ctx context.Context, org db.Organization, pool db.SendingPathway, p providers.Provider, mb providers.Mailbox) (res MailboxResult, pending string) {
	res = MailboxResult{Email: mb.Email}
	d, err := s.db.GetAnyDomainIdentity(ctx, domainOf(mb.Email))
	if err != nil || d.OrgID != org.ID || d.PathwayID.String != pool.ID {
		res.Error = domainOf(mb.Email) + " is not one of this pool's domains; connect it first"
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
			return res, mb.Email + ": not active at " + providers.Label(pool.MailboxService) + " yet (" + mb.Status + "); " + err.Error()
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

// PoolMailbox is one of the pool's mailboxes, in a sync.
type PoolMailbox struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// SyncResult compares the pool with its mailbox service.
type SyncResult struct {
	OnlyAtService []providers.Mailbox `json:"only_at_service"` // the service has them; the pool doesn't
	OnlyInPool    []PoolMailbox       `json:"only_in_pool"`    // the pool has them; the service doesn't
	Imported      []MailboxResult     `json:"imported"`
	Pending       []string            `json:"pending"`
}

// SyncService reports the mailboxes the service has that the pool doesn't,
// and the reverse. With importNew it imports the service's missing ones that
// are on the pool's domains and ready. A pool mailbox the service also has
// gets linked to it, so it can be deleted there.
func (s *Service) SyncService(ctx context.Context, org db.Organization, poolID string, importNew bool) (SyncResult, error) {
	pool, err := s.servicePool(ctx, org.ID, poolID)
	if err != nil {
		return SyncResult{}, err
	}
	p, err := s.provider(pool)
	if err != nil {
		return SyncResult{}, err
	}
	theirs, err := p.Mailboxes(ctx)
	if err != nil {
		return SyncResult{}, serviceErr(err)
	}
	ours, err := s.db.ListPoolMailboxes(ctx, sql.NullString{String: pool.ID, Valid: true})
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
		res, pending := s.importServiceMailbox(ctx, org, pool, p, mb)
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
// of its billing period (CheapInboxes) one of the pool's mailboxes at its
// service, and pauses it in the pool for good: its threads and replies stay.
func (s *Service) DeleteServiceMailbox(ctx context.Context, org db.Organization, poolID, mailboxID string) error {
	pool, err := s.servicePool(ctx, org.ID, poolID)
	if err != nil {
		return err
	}
	mb, err := s.db.GetMailbox(ctx, db.GetMailboxParams{ID: mailboxID, OrgID: org.ID})
	if err != nil {
		return refuse("no such mailbox")
	}
	if mbPool, err := s.pool(ctx, mb); err != nil || mbPool.ID != pool.ID {
		return refuse("%s is not in this pool", mb.Email)
	}
	p, err := s.provider(pool)
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
			return refuse("%s doesn't have %s", providers.Label(pool.MailboxService), mb.Email)
		}
	}
	if err := p.Delete(ctx, ref); err != nil {
		return serviceErr(err)
	}
	return s.db.SetMailboxStatus(ctx, db.SetMailboxStatusParams{
		Status: "paused", PausedReason: fmt.Sprintf("%s at %s on %s", deletedPrefix, providers.Label(pool.MailboxService), s.now().UTC().Format(time.DateOnly)), ID: mb.ID,
	})
}
