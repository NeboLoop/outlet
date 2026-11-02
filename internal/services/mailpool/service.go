// Package mailpool is the mailbox pool: warmed mailboxes on pool-only
// domains that a second deployment of Outlet sends customers' outreach
// from. Each mailbox belongs to one tenant at a time, ramps up its daily
// allowance, keeps a minimum gap between sends, and pauses itself on bad
// list health. Sending reuses the email package's SMTP pool and message
// builder; bounces and complaints land in the existing email_bounce and
// email_complaint tables; events go out through the webhook dispatcher.
//
// The pool is off unless Pool.Enabled; off, every method answers 503.
package mailpool

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/outlet-sh/outlet/internal/config"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/events"
	"github.com/outlet-sh/outlet/internal/services/crypto"
	"github.com/outlet-sh/outlet/internal/services/email"
)

// timeFormat is how the pool writes every time: UTC, sortable as text.
const timeFormat = "2006-01-02 15:04:05"

// Service runs the mailbox pool.
type Service struct {
	cfg    config.PoolConfig
	store  *db.Store
	crypto *crypto.Service
	bus    *events.Subject

	now      func() time.Time
	lookupMX func(ctx context.Context, domain string) ([]*net.MX, error) // nil uses DNS

	mu    sync.Mutex
	locks map[string]*sync.Mutex     // per mailbox: allowance check + reservation
	smtp  map[string]*email.SMTPPool // per mailbox: its own SMTP account
}

// New builds the pool service. bus may be nil (no webhooks).
func New(cfg config.PoolConfig, store *db.Store, cryptoSvc *crypto.Service, bus *events.Subject) *Service {
	return &Service{
		cfg:    cfg,
		store:  store,
		crypto: cryptoSvc,
		bus:    bus,
		now:    time.Now,
		locks:  map[string]*sync.Mutex{},
		smtp:   map[string]*email.SMTPPool{},
	}
}

// on refuses every call while the pool is off.
func (s *Service) on() error {
	if !s.cfg.IsEnabled() {
		return errorx.NewCodeError(http.StatusServiceUnavailable, "pool is off")
	}
	return nil
}

func fmtTime(t time.Time) string { return t.UTC().Format(timeFormat) }

func parseTime(v string) time.Time {
	t, err := time.ParseInLocation(timeFormat, v, time.UTC)
	if err != nil {
		return time.Time{}
	}
	return t
}

func dayStart(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func randomID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand: %v", err))
	}
	return hex.EncodeToString(b)
}

func domainOf(address string) string {
	if i := strings.LastIndex(address, "@"); i >= 0 {
		return strings.ToLower(address[i+1:])
	}
	return ""
}

func nullString(v string) sql.NullString { return sql.NullString{String: v, Valid: v != ""} }

func (s *Service) mailboxLock(id string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	l, ok := s.locks[id]
	if !ok {
		l = &sync.Mutex{}
		s.locks[id] = l
	}
	return l
}

// smtpFor returns the mailbox's SMTP pool, made on first use with the
// mailbox's own credentials.
func (s *Service) smtpFor(mb db.PoolMailbox) (*email.SMTPPool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ok := s.smtp[mb.ID]; ok {
		return p, nil
	}
	if s.crypto == nil {
		return nil, errors.New("ENCRYPTION_KEY is not set: mailbox passwords can't be read")
	}
	pass, err := s.crypto.DecryptString(mb.SmtpPasswordEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt mailbox password: %w", err)
	}
	port := int(mb.SmtpPort)
	p := email.NewSMTPPool(email.SMTPPoolConfig{
		Host:     mb.SmtpHost,
		Port:     port,
		User:     mb.SmtpUsername,
		Pass:     pass,
		UseTLS:   port == 465 || port == 587,
		PoolSize: 1,
	})
	s.smtp[mb.ID] = p
	return p, nil
}

func (s *Service) closeSMTP(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ok := s.smtp[id]; ok {
		p.Close()
		delete(s.smtp, id)
	}
}

// emit sends a pool event to the webhooks of the brand holding the tenant.
func (s *Service) emit(topic string, ev events.PoolEvent) {
	if s.bus == nil || ev.OrgID == "" {
		return
	}
	ev.Timestamp = s.now().UTC()
	_ = events.Emit(s.bus, topic, ev)
}

// allowance is how many emails the mailbox may send on the UTC day of now:
// min(Ceiling, RampStart + RampStep*whole days since its warm start).
func (s *Service) allowance(mb db.PoolMailbox, now time.Time) int {
	days := int(now.Sub(parseTime(mb.WarmStartedAt)) / (24 * time.Hour))
	if days < 0 {
		days = 0
	}
	n := s.cfg.RampStart + s.cfg.RampStep*days
	if n > s.cfg.Ceiling {
		n = s.cfg.Ceiling
	}
	return n
}

// MailboxCapacity is one mailbox as the hub sees it.
type MailboxCapacity struct {
	ID             string
	Address        string
	Status         string // active, paused, resting, retired
	PausedReason   string
	PauseKind      string // list, policy, provider, admin, domain
	Movable        bool   // paused for a provider fault: the tenant's queue may move to its other healthy mailboxes
	RemainingToday int
	NextAllowedAt  time.Time
}

// capacity reads a mailbox's state, today's allowance left and when it may
// send next. A paused domain shows as a paused mailbox.
func (s *Service) capacity(ctx context.Context, mb db.PoolMailbox, now time.Time) (MailboxCapacity, error) {
	c := MailboxCapacity{
		ID:           mb.ID,
		Address:      mb.Address,
		Status:       mb.Status,
		PausedReason: mb.PausedReason.String,
		PauseKind:    mb.PausedKind.String,
		Movable:      mb.Status == "paused" && mb.PausedKind.String == "provider",
	}
	if mb.Status == "active" {
		d, err := s.store.GetPoolDomain(ctx, mb.DomainID)
		if err != nil {
			return c, err
		}
		if d.Status != "active" {
			c.Status = "paused"
			c.PauseKind = "domain"
			c.PausedReason = "its domain " + d.Domain + " is " + d.Status
			if d.PausedReason.Valid {
				c.PausedReason += ": " + d.PausedReason.String
			}
		}
	}
	if c.Status != "active" {
		return c, nil
	}
	used, err := s.store.CountPoolMailboxAttemptsSince(ctx, db.CountPoolMailboxAttemptsSinceParams{
		MailboxID: mb.ID, Since: fmtTime(dayStart(now)),
	})
	if err != nil {
		return c, err
	}
	c.RemainingToday = s.allowance(mb, now) - int(used)
	if c.RemainingToday < 0 {
		c.RemainingToday = 0
	}
	c.NextAllowedAt = now
	last, err := s.store.LastPoolMailboxAttemptAt(ctx, mb.ID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return c, err
	default:
		if next := parseTime(last).Add(time.Duration(s.cfg.MinGapSeconds) * time.Second); next.After(now) {
			c.NextAllowedAt = next
		}
	}
	if c.RemainingToday == 0 {
		if tomorrow := dayStart(now).Add(24 * time.Hour); tomorrow.After(c.NextAllowedAt) {
			c.NextAllowedAt = tomorrow
		}
	}
	return c, nil
}

// Capacity is a tenant's mailboxes and whether repeat pauses hold it.
type Capacity struct {
	TenantHeld bool
	Mailboxes  []MailboxCapacity
}

// Capacity lists the tenant's mailboxes with today's allowance left.
func (s *Service) Capacity(ctx context.Context, orgID, tenant string) (*Capacity, error) {
	if err := s.on(); err != nil {
		return nil, err
	}
	if tenant == "" {
		return nil, errorx.NewBadRequestError("tenant is required")
	}
	return s.tenantCapacity(ctx, orgID, tenant)
}

func (s *Service) tenantCapacity(ctx context.Context, orgID, tenant string) (*Capacity, error) {
	now := s.now()
	held, err := s.tenantHeld(ctx, orgID, tenant, now)
	if err != nil {
		return nil, err
	}
	mbs, err := s.store.ListTenantPoolMailboxes(ctx, db.ListTenantPoolMailboxesParams{
		OrgID: nullString(orgID), Tenant: nullString(tenant),
	})
	if err != nil {
		return nil, err
	}
	out := &Capacity{TenantHeld: held, Mailboxes: []MailboxCapacity{}}
	for _, mb := range mbs {
		c, err := s.capacity(ctx, mb, now)
		if err != nil {
			return nil, err
		}
		out.Mailboxes = append(out.Mailboxes, c)
	}
	return out, nil
}

// tenantHeld: RepeatPauses list or policy pauses in RepeatPauseDays hold
// the tenant until an admin clears it.
func (s *Service) tenantHeld(ctx context.Context, orgID, tenant string, now time.Time) (bool, error) {
	n, err := s.store.CountTenantPoolPausesSince(ctx, db.CountTenantPoolPausesSinceParams{
		OrgID:  nullString(orgID),
		Tenant: nullString(tenant),
		Since:  fmtTime(now.Add(-time.Duration(s.cfg.RepeatPauseDays) * 24 * time.Hour)),
	})
	if err != nil {
		return false, err
	}
	return int(n) >= s.cfg.RepeatPauses, nil
}

// Assign gives the tenant up to count more free mailboxes: active or
// rested, on an active domain, and never held by another tenant inside
// RestDays. It answers the tenant's mailboxes after.
func (s *Service) Assign(ctx context.Context, orgID, tenant string, count int) (*Capacity, int, error) {
	if err := s.on(); err != nil {
		return nil, 0, err
	}
	if tenant == "" || count < 1 {
		return nil, 0, errorx.NewBadRequestError("tenant and a count of at least 1 are required")
	}
	now := s.now()
	free, err := s.store.AssignablePoolMailboxes(ctx, db.AssignablePoolMailboxesParams{
		RestCutoff: nullString(fmtTime(now.Add(-time.Duration(s.cfg.RestDays) * 24 * time.Hour))),
		OrgID:      nullString(orgID),
		Tenant:     nullString(tenant),
		MaxCount:   int64(count),
	})
	if err != nil {
		return nil, 0, err
	}
	assigned := 0
	for _, mb := range free {
		n, err := s.store.AssignPoolMailbox(ctx, db.AssignPoolMailboxParams{
			OrgID: nullString(orgID), Tenant: nullString(tenant), AssignedAt: nullString(fmtTime(now)), ID: mb.ID,
		})
		if err != nil {
			return nil, 0, err
		}
		assigned += int(n)
	}
	c, err := s.tenantCapacity(ctx, orgID, tenant)
	if err != nil {
		return nil, 0, err
	}
	if len(c.Mailboxes) == 0 {
		return nil, 0, errorx.NewCodeError(http.StatusServiceUnavailable, "no mailboxes: none is free to assign")
	}
	return c, assigned, nil
}

// Release frees every mailbox of the tenant. They rest RestDays before
// another tenant gets them; the same tenant may take them back sooner.
// Old threads still name the first tenant (each send keeps its tenant).
func (s *Service) Release(ctx context.Context, orgID, tenant string) (int, error) {
	if err := s.on(); err != nil {
		return 0, err
	}
	if tenant == "" {
		return 0, errorx.NewBadRequestError("tenant is required")
	}
	n, err := s.store.ReleaseTenantPoolMailboxes(ctx, db.ReleaseTenantPoolMailboxesParams{
		ReleasedAt: nullString(fmtTime(s.now())), OrgID: nullString(orgID), Tenant: nullString(tenant),
	})
	return int(n), err
}

func newID() string { return uuid.New().String() }
