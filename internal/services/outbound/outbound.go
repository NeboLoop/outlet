// Package outbound is a brand's one-to-one outreach. It runs on Outlet's own
// lists, contacts, sequences, email queue, suppression, validation, merge
// tags and tracking. What it adds: the brand's outreach domains and connected
// mailboxes, the rules cold email needs, the outreach sender, and reading
// replies. Outreach goes through the brand's sending pathway that carries it
// (normally its mailbox pool); the pace, warm-up, daily limits and health
// thresholds are that pathway's, enforced in email.Send.
package outbound

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/crypto"
	"github.com/outlet-sh/outlet/internal/services/email"
)

const (
	stuckAfter = 10 * time.Minute // a send claimed this long ago crashed
	maxBatch   = 100              // items in one call
)

// Refusal is an error the caller can act on: what was refused and how to fix it.
type Refusal struct{ Msg string }

func (r *Refusal) Error() string { return r.Msg }

func refuse(format string, args ...any) error { return &Refusal{Msg: fmt.Sprintf(format, args...)} }

// Verifier checks one address for a brand before anything is sent to it
// (Outlet's validation service).
type Verifier func(ctx context.Context, orgID, address string) (ok bool, reason string)

// Config is what the server gives outbound.
type Config struct {
	ProtectedDomains []string // never used for outreach, beyond each brand's sending domains
	TrackingTarget   string   // this server's public host; go.<domain> CNAMEs point at it
	Verify           Verifier
	Mailer           *email.Service         // the one Send, and signed tracking links
	Sequences        *email.SequenceService // the one sequence engine
}

// Service is outbound for every brand on this server.
type Service struct {
	db        *db.Store
	crypto    *crypto.Service
	mailer    *email.Service
	seq       *email.SequenceService
	protected []string
	tracking  string
	verify    Verifier
	dns       Resolver
	doAPI     string // DigitalOcean API base, replaced in tests
	now       func() time.Time
	mu        sync.Mutex
	rnd       *rand.Rand
}

// New makes the service. Outbound is off for a brand until its admin turns it on.
func New(store *db.Store, cryptoSvc *crypto.Service, cfg Config) *Service {
	s := &Service{
		db: store, crypto: cryptoSvc, mailer: cfg.Mailer, seq: cfg.Sequences,
		tracking: normDomain(cfg.TrackingTarget), verify: cfg.Verify,
		dns: netResolver{}, doAPI: "https://api.digitalocean.com", now: time.Now,
		rnd: rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0)),
	}
	for _, d := range cfg.ProtectedDomains {
		if d = normDomain(d); d != "" {
			s.protected = append(s.protected, d)
		}
	}
	return s
}

// Enabled says whether outbound runs for this brand.
func (s *Service) Enabled(org db.Organization) bool { return org.OutboundEnabled == 1 }

// SetEnabled turns outbound on or off for a brand. Only the admin app calls
// it, for the brand's admin; no MCP tool can.
func (s *Service) SetEnabled(ctx context.Context, orgID string, on bool) error {
	return s.db.SetOutboundEnabled(ctx, db.SetOutboundEnabledParams{OutboundEnabled: boolInt(on), ID: orgID})
}

func (s *Service) requireEnabled(org db.Organization) error {
	if !s.Enabled(org) {
		return refuse("Outbound is off for brand %q. The brand's admin turns it on in the Outlet app (Outbound, Settings); no tool call can.", org.Name)
	}
	return nil
}

func (s *Service) rand() *rand.Rand {
	s.mu.Lock()
	defer s.mu.Unlock()
	return rand.New(rand.NewPCG(s.rnd.Uint64(), s.rnd.Uint64()))
}

// pool is the mailbox pool (sending pathway) a mailbox belongs to.
func (s *Service) pool(ctx context.Context, mb db.Mailbox) (db.SendingPathway, error) {
	return s.db.GetMailboxPathway(ctx, mb.ID)
}

func ts(t time.Time) string { return t.UTC().Format(email.TimeLayout) }

func parseTS(s string) time.Time {
	t, _ := time.ParseInLocation(email.TimeLayout, s, time.UTC)
	return t
}

func boolInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func normDomain(d string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(d)), ".")
}

func domainOf(address string) string {
	_, d, _ := strings.Cut(address, "@")
	return normDomain(d)
}

func normEmail(e string) string { return strings.ToLower(strings.TrimSpace(e)) }
