package emailval

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// DisposableProvider reports whether a domain is disposable/temporary.
type DisposableProvider interface {
	IsDisposable(domain string) bool
}

// inMemoryDisposableProvider is a map-backed provider.
type inMemoryDisposableProvider struct {
	set map[string]struct{}
}

func (p inMemoryDisposableProvider) IsDisposable(domain string) bool {
	_, ok := p.set[strings.ToLower(domain)]
	return ok
}

// NewStaticDisposableProvider builds a provider from a list of domains.
func NewStaticDisposableProvider(domains []string) DisposableProvider {
	set := make(map[string]struct{}, len(domains))
	for _, d := range domains {
		set[strings.ToLower(strings.TrimSpace(d))] = struct{}{}
	}
	return inMemoryDisposableProvider{set: set}
}

// DefaultDisposableProvider returns the built-in list merged with the community blocklist
// (disposable-email-domains), which Validate refreshes in the background at most once per 24h.
func DefaultDisposableProvider() DisposableProvider {
	return defaultDisposable{}
}

// builtinDisposable is the compiled-in fallback used before (or without) a remote download.
var builtinDisposable = NewStaticDisposableProvider(defaultDisposableDomains)

type defaultDisposable struct{}

func (defaultDisposable) IsDisposable(domain string) bool {
	if builtinDisposable.IsDisposable(domain) {
		return true
	}
	if set := remoteDisposable.set.Load(); set != nil {
		_, ok := (*set)[strings.ToLower(domain)]
		return ok
	}
	return false
}

// disposableListURL is the community-maintained blocklist; swapped in tests.
var disposableListURL = "https://raw.githubusercontent.com/disposable-email-domains/disposable-email-domains/main/disposable_email_blocklist.conf"

const (
	disposableRefreshEvery = 24 * time.Hour
	disposableRetryAfter   = 15 * time.Minute // after a failed download
	disposableFetchTimeout = 60 * time.Second
)

// remoteDisposable is the package-level downloaded list.
var remoteDisposable = &remoteDisposableList{}

type remoteDisposableList struct {
	set atomic.Pointer[map[string]struct{}]

	mu        sync.Mutex
	fetchedAt time.Time     // last successful download
	triedAt   time.Time     // last download attempt
	loading   chan struct{} // non-nil while a download is in flight
}

// ensure starts a download when the list is missing or stale. Only the very first load is waited
// for, and never past ctx; refreshes run in the background. Failures keep the previous/builtin list.
func (l *remoteDisposableList) ensure(ctx context.Context) {
	l.mu.Lock()
	loaded := l.set.Load() != nil
	due := time.Since(l.fetchedAt) > disposableRefreshEvery && time.Since(l.triedAt) > disposableRetryAfter
	if l.loading == nil && due {
		l.triedAt = time.Now()
		l.loading = make(chan struct{})
		go l.fetch(l.loading)
	}
	wait := l.loading
	l.mu.Unlock()

	if !loaded && wait != nil {
		select {
		case <-wait:
		case <-ctx.Done():
		}
	}
}

func (l *remoteDisposableList) fetch(done chan struct{}) {
	ctx, cancel := context.WithTimeout(context.Background(), disposableFetchTimeout)
	defer cancel()
	set, err := downloadDisposableList(ctx, disposableListURL)

	l.mu.Lock()
	if err == nil {
		l.set.Store(&set)
		l.fetchedAt = time.Now()
	}
	l.loading = nil
	l.mu.Unlock()
	close(done)
}

func downloadDisposableList(ctx context.Context, url string) (map[string]struct{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("disposable list: HTTP %d", resp.StatusCode)
	}
	set := make(map[string]struct{}, 1<<17)
	sc := bufio.NewScanner(io.LimitReader(resp.Body, 32<<20))
	for sc.Scan() {
		line := strings.ToLower(strings.TrimSpace(sc.Text()))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		set[line] = struct{}{}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(set) == 0 {
		return nil, fmt.Errorf("disposable list: empty")
	}
	return set, nil
}

// defaultDisposableDomains is a minimal list of known disposable email domains.
// Expand this list or use an external service for comprehensive coverage.
var defaultDisposableDomains = []string{
	// Popular disposable services
	"mailinator.com",
	"guerrillamail.com",
	"guerrillamail.org",
	"guerrillamail.net",
	"guerrillamail.biz",
	"guerrillamail.de",
	"10minutemail.com",
	"10minutemail.net",
	"tempmail.com",
	"tempmail.net",
	"temp-mail.org",
	"throwaway.email",
	"throwawaymail.com",
	"fakeinbox.com",
	"fakemailgenerator.com",
	"getnada.com",
	"getairmail.com",
	"dispostable.com",
	"mailnesia.com",
	"maildrop.cc",
	"yopmail.com",
	"yopmail.fr",
	"sharklasers.com",
	"trashmail.com",
	"trashmail.net",
	"trashmail.org",
	"spamgourmet.com",
	"mytrashmail.com",
	"mailcatch.com",
	"mailexpire.com",
	"tempinbox.com",
	"tempr.email",
	"discard.email",
	"discardmail.com",
	"spambog.com",
	"spambog.de",
	"spambog.ru",
	"emailondeck.com",
	"mohmal.com",
	"burnermail.io",
	"mintemail.com",
	"tempail.com",
	"emailfake.com",
	"crazymailing.com",
}
