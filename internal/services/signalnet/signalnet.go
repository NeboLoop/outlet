// Package signalnet is the client for the Outlet Signal Network: pooled, hashed
// deliverability signals shared across Outlet instances through a hub run by outlet.sh.
// Emails never leave an instance in clear text; only SHA-256 of the lowercased address.
//
// Privacy. What an instance shares with the hub:
//   - SHA-256 hex of the trimmed, lowercased recipient address (see Hash)
//   - the outcome kind (delivered, hard_bounce, complaint, open, click) and its time
//   - its instance key, which identifies the subscriber, not any brand or sender
//
// What is never shared: addresses in clear text, names, contact fields, message
// content, subjects, URLs, IPs, brands, lists or any per-recipient metadata beyond the
// outcome. Lookups send only hashes. The hub stores per-hash counters and which
// instances reported a hash, nothing else. Note that a hash of a known address can be
// recomputed by anyone who already has that address; hashing hides addresses the hub
// has never seen, it does not anonymize addresses the hub operator already knows.
package signalnet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// Signal is what the network knows about one address across all reporting instances.
type Signal struct {
	Hash          string    `json:"hash"`
	Senders       int       `json:"senders"` // distinct instances that mailed it
	Delivered     int       `json:"delivered"`
	HardBounces   int       `json:"hard_bounces"`
	Complaints    int       `json:"complaints"`
	Engaged       int       `json:"engaged"` // opens or clicks
	LastEngagedAt time.Time `json:"last_engaged_at"`
	TrapScore     float64   `json:"trap_score"` // 0..1, hub heuristic
	Toxic         bool      `json:"toxic"`
}

// Event is one deliverability outcome reported by an instance.
type Event struct {
	Hash string    `json:"hash"`
	Kind string    `json:"kind"` // delivered, hard_bounce, complaint, open, click
	At   time.Time `json:"at"`
}

// Event kinds.
const (
	KindDelivered  = "delivered"
	KindHardBounce = "hard_bounce"
	KindComplaint  = "complaint"
	KindOpen       = "open"
	KindClick      = "click"
)

const (
	requestTimeout = 2 * time.Second
	cacheTTL       = time.Hour
	cacheMax       = 10000
	// MaxLookup and MaxReport are the hub's per-request limits.
	MaxLookup = 100
	MaxReport = 1000
)

// Client talks to the hub. A nil *Client is valid and disabled.
type Client struct {
	HubURL      string
	InstanceKey string

	mu    sync.Mutex
	cache map[string]cacheEntry
}

type cacheEntry struct {
	signal  *Signal // nil = unknown to the network
	expires time.Time
}

var httpClient = &http.Client{Timeout: requestTimeout}

// Hash returns the network identifier for an email address.
func Hash(email string) string { return hash(email) }

func (c *Client) enabled() bool {
	return c != nil && c.HubURL != "" && c.InstanceKey != ""
}

// Lookup returns the network's signal for email, or nil when disabled or unknown.
// It fails open: hub errors are logged and reported as unknown (nil, nil), so
// validation never blocks on the network.
func (c *Client) Lookup(ctx context.Context, email string) (*Signal, error) {
	if !c.enabled() {
		return nil, nil
	}
	h := Hash(email)
	if s, ok := c.cached(h); ok {
		return s, nil
	}

	var resp struct {
		Signals []Signal `json:"signals"`
	}
	if err := c.post(ctx, "/network/v1/lookup", map[string][]string{"hashes": {h}}, &resp); err != nil {
		logx.WithContext(ctx).Errorf("signalnet: lookup failed: %v", err)
		return nil, nil
	}
	var s *Signal
	for i := range resp.Signals {
		if resp.Signals[i].Hash == h {
			s = &resp.Signals[i]
		}
	}
	c.store(h, s)
	return s, nil
}

// Report sends outcomes to the hub in batches of MaxReport. No-op when disabled.
func (c *Client) Report(ctx context.Context, events []Event) error {
	if !c.enabled() {
		return nil
	}
	for len(events) > 0 {
		n := min(len(events), MaxReport)
		if err := c.post(ctx, "/network/v1/report", map[string][]Event{"events": events[:n]}, nil); err != nil {
			return err
		}
		events = events[n:]
	}
	return nil
}

func (c *Client) post(ctx context.Context, path string, body, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.HubURL, "/")+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Instance-Key", c.InstanceKey)
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return fmt.Errorf("hub returned %s", res.Status)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}

func (c *Client) cached(h string) (*Signal, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.cache[h]
	if !ok || time.Now().After(e.expires) {
		return nil, false
	}
	return e.signal, true
}

func (c *Client) store(h string, s *Signal) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cache == nil {
		c.cache = make(map[string]cacheEntry)
	}
	if len(c.cache) >= cacheMax {
		now := time.Now()
		for k, e := range c.cache {
			if now.After(e.expires) {
				delete(c.cache, k)
			}
		}
		// ponytail: evicts an arbitrary entry when still full, not LRU. Ceiling: hit rate
		// drops when more than cacheMax distinct addresses are validated per hour.
		for k := range c.cache {
			if len(c.cache) < cacheMax {
				break
			}
			delete(c.cache, k)
		}
	}
	c.cache[h] = cacheEntry{signal: s, expires: time.Now().Add(cacheTTL)}
}
