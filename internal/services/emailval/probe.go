package emailval

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"
)

// ProbeResult is the outcome of an SMTP RCPT probe for one address.
type ProbeResult struct {
	// Deliverable is nil when the server gave no definitive answer.
	Deliverable *bool
	// CatchAll is true when the domain accepted a random mailbox.
	CatchAll bool
	// SubStatus explains a rejection or an unknown result (Sub* constants), empty when accepted.
	SubStatus string
}

const (
	// defaultProbeTimeout bounds one SMTP session's network I/O when Options.Timeout is unset.
	defaultProbeTimeout = 10 * time.Second
	// probeIntervalLargeProvider spaces probes from one exit to Google or Microsoft.
	probeIntervalLargeProvider = time.Second
	// probeIntervalOther spaces probes from one exit to any other MX host.
	probeIntervalOther = 2 * time.Second
	// catchAllTTL is how long a domain's catch-all verdict is reused.
	catchAllTTL = 24 * time.Hour
	// probeCacheSweepSize triggers pruning of expired entries in package-level maps.
	probeCacheSweepSize = 10000
)

// probeMailbox probes email on domain's MX hosts (preference order).
// It owns proxy selection, per-provider rate limits, catch-all detection and response parsing.
func probeMailbox(ctx context.Context, domain, email string, mxHosts []string, opts *Options) ProbeResult {
	if opts == nil {
		opts = &Options{}
	}
	if len(mxHosts) == 0 {
		return ProbeResult{SubStatus: SubFailedSMTPConnection}
	}

	switch probeProvider(mxHosts[0]) {
	case "yahoo":
		// Yahoo/AOL accept every RCPT and bounce later, so a probe proves nothing
		// and only burns exit reputation. Mailbox existence is unknown: report catch-all.
		return ProbeResult{CatchAll: true}
	}

	catchAll, known := catchAllVerdicts.get(domain)
	if known && catchAll {
		yes := true
		return ProbeResult{Deliverable: &yes, CatchAll: true}
	}

	exit, ok := pickExit(opts)
	if !ok {
		return ProbeResult{SubStatus: SubFailedSMTPConnection}
	}

	cfg := opts.SMTP
	if cfg == nil {
		cfg = DefaultSMTPConfig()
	}
	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 2
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultProbeTimeout
	}

	result := ProbeResult{SubStatus: SubFailedSMTPConnection}
	for i, host := range mxHosts {
		if i >= maxRetries {
			break
		}
		host = strings.TrimSuffix(host, ".")
		if !cfg.SkipRateLimiting {
			provider := probeProvider(host)
			if err := probeLimiter.wait(ctx, exit.key+"|"+provider, probeInterval(provider)); err != nil {
				return ProbeResult{SubStatus: SubTimeoutExceeded}
			}
		}

		// The I/O deadline starts after the rate-limit wait.
		probeCtx, cancel := context.WithTimeout(ctx, timeout)
		out := smtpProbe(probeCtx, exit.dialer, net.JoinHostPort(host, "25"), domain, email, !known, cfg)
		cancel()

		if exit.proxy {
			fleetHealth.report(exit.key, out)
		}
		if out.catchAll != nil {
			catchAllVerdicts.set(domain, *out.catchAll)
		}
		result = out.result
		if !retryOnNextMX(result.SubStatus) {
			break
		}
	}
	return result
}

// retryOnNextMX reports whether another MX host might answer where this one could not.
func retryOnNextMX(subStatus string) bool {
	switch subStatus {
	case SubFailedSMTPConnection, SubForcibleDisconnect, SubTimeoutExceeded,
		SubMailServerNoResponse, SubMailServerTempError:
		return true
	}
	return false
}

// probeProvider names the mailbox provider behind an MX host: google, microsoft,
// yahoo, or the MX host itself for everyone else.
func probeProvider(mxHost string) string {
	h := strings.ToLower(strings.TrimSuffix(mxHost, "."))
	hasSuffix := func(suffixes ...string) bool {
		for _, s := range suffixes {
			if h == s || strings.HasSuffix(h, "."+s) {
				return true
			}
		}
		return false
	}
	switch {
	case hasSuffix("google.com", "googlemail.com", "smtp.goog"):
		return "google"
	case hasSuffix("outlook.com", "hotmail.com"):
		return "microsoft"
	case hasSuffix("yahoodns.net"):
		return "yahoo"
	}
	return h
}

func probeInterval(provider string) time.Duration {
	switch provider {
	case "google", "microsoft":
		return probeIntervalLargeProvider
	}
	return probeIntervalOther
}

// probeLimiter spaces probes per (exit, provider) to protect exit reputation.
var probeLimiter = &rateLimiter{}

type rateLimiter struct {
	mu   sync.Mutex
	next map[string]time.Time // earliest start of the next probe per key
}

// wait blocks until key may probe again, reserving the following slot interval later.
func (r *rateLimiter) wait(ctx context.Context, key string, interval time.Duration) error {
	r.mu.Lock()
	now := time.Now()
	if r.next == nil {
		r.next = make(map[string]time.Time)
	}
	if len(r.next) > probeCacheSweepSize {
		for k, t := range r.next {
			if t.Before(now) {
				delete(r.next, k)
			}
		}
	}
	start := r.next[key]
	if start.Before(now) {
		start = now
	}
	r.next[key] = start.Add(interval)
	r.mu.Unlock()

	d := start.Sub(now)
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// catchAllVerdicts caches per-domain catch-all results.
var catchAllVerdicts = &catchAllCache{}

type catchAllCache struct {
	mu      sync.Mutex
	entries map[string]catchAllEntry
}

type catchAllEntry struct {
	catchAll bool
	expires  time.Time
}

func (c *catchAllCache) get(domain string) (catchAll, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[domain]
	if !ok || time.Now().After(e.expires) {
		return false, false
	}
	return e.catchAll, true
}

func (c *catchAllCache) set(domain string, catchAll bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	if c.entries == nil {
		c.entries = make(map[string]catchAllEntry)
	}
	if len(c.entries) > probeCacheSweepSize {
		for k, e := range c.entries {
			if now.After(e.expires) {
				delete(c.entries, k)
			}
		}
	}
	c.entries[domain] = catchAllEntry{catchAll: catchAll, expires: now.Add(catchAllTTL)}
}
