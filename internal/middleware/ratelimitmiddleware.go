package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RateLimitConfig defines rate limiting parameters
type RateLimitConfig struct {
	// MaxAttempts is the maximum number of attempts allowed within the window
	MaxAttempts int
	// WindowSeconds is the time window in seconds
	WindowSeconds int
	// BlockDurationSeconds is how long to block after exceeding the limit
	BlockDurationSeconds int
}

// DefaultAuthRateLimitConfig returns sensible defaults for auth endpoints
func DefaultAuthRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		MaxAttempts:          5,   // 5 attempts
		WindowSeconds:        60,  // per minute
		BlockDurationSeconds: 300, // block for 5 minutes after exceeding
	}
}

// rateLimitEntry tracks attempts from a client on an endpoint
type rateLimitEntry struct {
	attempts  int
	firstSeen time.Time
	blockedAt time.Time
}

// RateLimitMiddleware provides rate limiting per client address and endpoint
type RateLimitMiddleware struct {
	config  RateLimitConfig
	entries map[string]*rateLimitEntry
	mu      sync.RWMutex
}

// NewRateLimitMiddleware creates a new rate limit middleware with the given config
func NewRateLimitMiddleware(config RateLimitConfig) *RateLimitMiddleware {
	rl := &RateLimitMiddleware{
		config:  config,
		entries: make(map[string]*rateLimitEntry),
	}

	// Start cleanup goroutine to remove old entries
	go rl.cleanup()

	return rl
}

// cleanup periodically removes stale entries
func (m *RateLimitMiddleware) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for ip, entry := range m.entries {
			// Remove entries that are older than the block duration
			if now.Sub(entry.firstSeen) > time.Duration(m.config.BlockDurationSeconds)*time.Second {
				delete(m.entries, ip)
			}
		}
		m.mu.Unlock()
	}
}

// getClientIP is the address of the client that sent the request. A
// proxy's own address is on a private or loopback network, so the client is
// the nearest address that isn't: RemoteAddr itself when Outlet is reached
// directly (forwarding headers are then ignored, as anyone could write
// them), otherwise the X-Forwarded-For chain read from the right, the end the
// proxies wrote, falling back to the X-Real-IP a proxy sets. When every hop
// is private it is the farthest one known.
func getClientIP(r *http.Request) string {
	remote := r.RemoteAddr
	if host, _, err := net.SplitHostPort(remote); err == nil {
		remote = host
	}
	if !trustedProxy(remote) {
		return remote
	}
	hops := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	farthest := ""
	for i := len(hops) - 1; i >= 0; i-- {
		hop := strings.TrimSpace(hops[i])
		if hop == "" {
			continue
		}
		if !trustedProxy(hop) {
			return hop
		}
		farthest = hop
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return xri
	}
	if farthest != "" {
		return farthest
	}
	return remote
}

// trustedProxy reports whether an address is a proxy's: one on a private or
// loopback network. Anything that isn't an address is not trusted.
func trustedProxy(addr string) bool {
	ip, err := netip.ParseAddr(addr)
	if err != nil {
		return false
	}
	return ip.IsPrivate() || ip.IsLoopback()
}

// limitKey is what a request is counted under: the client and the endpoint.
// Each endpoint has its own count, so wrong passwords on sign-in don't block
// "Forgot password", a reset link or an emailed sign-in.
func limitKey(r *http.Request) string {
	return getClientIP(r) + " " + r.URL.Path
}

// Handle wraps an HTTP handler with rate limiting
func (m *RateLimitMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := limitKey(r)

		m.mu.Lock()
		entry, exists := m.entries[key]
		now := time.Now()

		if !exists {
			// First request from this client to this endpoint
			m.entries[key] = &rateLimitEntry{
				attempts:  1,
				firstSeen: now,
			}
			m.mu.Unlock()
			next(w, r)
			return
		}

		// Check if currently blocked
		if !entry.blockedAt.IsZero() {
			blockExpiry := entry.blockedAt.Add(time.Duration(m.config.BlockDurationSeconds) * time.Second)
			if now.Before(blockExpiry) {
				m.mu.Unlock()
				remainingSeconds := int(blockExpiry.Sub(now).Seconds())
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", strconv.Itoa(remainingSeconds))
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":         "Too many attempts. Please try again later.",
					"retry_after":   remainingSeconds,
					"blocked_until": blockExpiry.Format(time.RFC3339),
				})
				return
			}
			// Block expired, reset the entry
			entry.attempts = 1
			entry.firstSeen = now
			entry.blockedAt = time.Time{}
			m.mu.Unlock()
			next(w, r)
			return
		}

		// Check if window has expired
		windowExpiry := entry.firstSeen.Add(time.Duration(m.config.WindowSeconds) * time.Second)
		if now.After(windowExpiry) {
			// Window expired, reset
			entry.attempts = 1
			entry.firstSeen = now
			m.mu.Unlock()
			next(w, r)
			return
		}

		// Within window, check attempts
		entry.attempts++
		if entry.attempts > m.config.MaxAttempts {
			// Rate limit exceeded, block the IP
			entry.blockedAt = now
			m.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", strconv.Itoa(m.config.BlockDurationSeconds))
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":         "Too many attempts. Please try again later.",
				"retry_after":   m.config.BlockDurationSeconds,
				"blocked_until": now.Add(time.Duration(m.config.BlockDurationSeconds) * time.Second).Format(time.RFC3339),
			})
			return
		}

		m.mu.Unlock()
		next(w, r)
	}
}

// RecordFailure allows logic layers to record a failed authentication attempt
// This allows the rate limiter to track failed logins even for correct IPs with wrong passwords
func (m *RateLimitMiddleware) RecordFailure(r *http.Request) {
	key := limitKey(r)

	m.mu.Lock()
	defer m.mu.Unlock()

	entry, exists := m.entries[key]
	if !exists {
		m.entries[key] = &rateLimitEntry{
			attempts:  1,
			firstSeen: time.Now(),
		}
		return
	}

	// Don't increment if already blocked
	if !entry.blockedAt.IsZero() {
		return
	}

	entry.attempts++
	if entry.attempts > m.config.MaxAttempts {
		entry.blockedAt = time.Now()
	}
}

// ResetForIP clears rate limit tracking for a request's client on its
// endpoint (e.g., after successful login)
func (m *RateLimitMiddleware) ResetForIP(r *http.Request) {
	key := limitKey(r)

	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.entries, key)
}
