// Package outbound runs one-to-one outreach from connected mailboxes on
// outreach domains: prospects, sequences, approvals, sending, replies and the
// safety limits. It never sends through SES or a brand's transactional path,
// and no caller can raise its limits: they are constants here.
package outbound

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/crypto"
	"github.com/outlet-sh/outlet/internal/services/emailval"
)

// The safety limits. Only a code change moves them.
const (
	MaxDailyCap       = 30               // emails per mailbox per day
	warmupStart       = 5                // a new mailbox's first day
	warmupStep        = 2                // added each day until the cap
	minGap            = 4 * time.Minute  // between two sends from one mailbox
	maxGap            = 9 * time.Minute  //
	bounceRateLimit   = 0.03             // per mailbox, last 7 days
	minBouncesToPause = 2                // one bounce alone never pauses
	complaintLimit    = 2                // per mailbox, last 7 days
	stuckAfter        = 10 * time.Minute // a send claimed this long ago crashed
	maxBatch          = 100              // items in one tool call
)

// Refusal is an error the caller can act on: what was refused and how to fix it.
type Refusal struct{ Msg string }

func (r *Refusal) Error() string { return r.Msg }

func refuse(format string, args ...any) error { return &Refusal{Msg: fmt.Sprintf(format, args...)} }

// Verifier checks one address before anything is sent to it.
type Verifier func(ctx context.Context, email string) (ok bool, reason string)

// Service is outbound for every brand on this server.
type Service struct {
	db        *db.Store
	crypto    *crypto.Service
	brands    map[string]bool // brand slugs outbound runs for (server config)
	protected []string        // domains outbound may never use
	verify    Verifier
	dns       Resolver
	doAPI     string // DigitalOcean API base, replaced in tests
	now       func() time.Time
	mu        sync.Mutex
	rnd       *rand.Rand
}

// Config is what the server's config says about outbound.
type Config struct {
	Brands           []string // brand slugs outbound is on for
	ProtectedDomains []string // never used for outbound, beyond the brand's own
}

// New makes the service. Outbound is off for every brand not in cfg.Brands.
func New(store *db.Store, cryptoSvc *crypto.Service, cfg Config) *Service {
	s := &Service{
		db:     store,
		crypto: cryptoSvc,
		brands: map[string]bool{},
		verify: emailvalVerifier,
		dns:    netResolver{},
		doAPI:  "https://api.digitalocean.com",
		now:    time.Now,
		rnd:    rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0)),
	}
	for _, b := range cfg.Brands {
		if b = strings.TrimSpace(strings.ToLower(b)); b != "" {
			s.brands[b] = true
		}
	}
	for _, d := range cfg.ProtectedDomains {
		if d = normDomain(d); d != "" {
			s.protected = append(s.protected, d)
		}
	}
	return s
}

// Enabled says whether outbound runs for this brand.
func (s *Service) Enabled(org db.Organization) bool { return s.brands[strings.ToLower(org.Slug)] }

func (s *Service) requireEnabled(org db.Organization) error {
	if !s.Enabled(org) {
		return refuse("Outbound is off for brand %q. An admin turns it on in the server config (OUTBOUND_BRANDS=%s); no tool call can.", org.Name, org.Slug)
	}
	return nil
}

func (s *Service) rand() *rand.Rand {
	s.mu.Lock()
	defer s.mu.Unlock()
	return rand.New(rand.NewPCG(s.rnd.Uint64(), s.rnd.Uint64()))
}

func (s *Service) gap() time.Duration {
	r := s.rand()
	return minGap + time.Duration(r.Int64N(int64(maxGap-minGap)))
}

func (s *Service) settings(ctx context.Context, orgID string) (db.OutboundSetting, error) {
	st, err := s.db.GetOutboundSettings(ctx, orgID)
	if errors.Is(err, sql.ErrNoRows) {
		return s.db.UpsertOutboundSettings(ctx, db.UpsertOutboundSettingsParams{
			OrgID:      orgID,
			OptOutLine: `If this isn't for you, reply "no thanks" and I won't email again.`,
			Timezone:   "America/Denver",
		})
	}
	return st, err
}

// allowance is how many emails a mailbox may send today: the warm-up ramp
// until it reaches its cap.
func allowance(mb db.OutboundMailbox, now time.Time) int64 {
	days := int64(now.Sub(parseTS(mb.WarmupStartedAt)).Hours() / 24)
	if days < 0 {
		days = 0
	}
	cap := min(mb.DailyCap, MaxDailyCap)
	return min(cap, warmupStart+warmupStep*days)
}

func normDomain(d string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(d)), ".")
}

func domainOf(email string) string {
	_, d, _ := strings.Cut(email, "@")
	return normDomain(d)
}

func normEmail(e string) string { return strings.ToLower(strings.TrimSpace(e)) }

func emailvalVerifier(ctx context.Context, email string) (bool, string) {
	res, err := emailval.Validate(ctx, email, &emailval.Options{CheckDisposable: true, AllowRole: true, Timeout: 5 * time.Second})
	if err != nil {
		return false, err.Error()
	}
	if !res.Valid() {
		return false, strings.Join(res.Messages, "; ")
	}
	return true, "ok"
}
