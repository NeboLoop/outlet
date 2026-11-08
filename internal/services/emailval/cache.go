package emailval

import (
	"sync"
	"time"
)

// ttlCacheMaxEntries bounds memory during bulk jobs over millions of domains.
const ttlCacheMaxEntries = 200_000

// ttlCache is a mutex-guarded, package-level per-key cache with a fixed TTL.
type ttlCache[V any] struct {
	mu  sync.Mutex
	ttl time.Duration
	m   map[string]ttlEntry[V]
}

type ttlEntry[V any] struct {
	val     V
	expires time.Time
}

func newTTLCache[V any](ttl time.Duration) *ttlCache[V] {
	return &ttlCache[V]{ttl: ttl, m: make(map[string]ttlEntry[V])}
}

func (c *ttlCache[V]) get(key string) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.m[key]
	if !ok || time.Now().After(e.expires) {
		var zero V
		return zero, false
	}
	return e.val, true
}

func (c *ttlCache[V]) set(key string, val V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	if len(c.m) >= ttlCacheMaxEntries {
		for k, e := range c.m {
			if now.After(e.expires) {
				delete(c.m, k)
			}
		}
		// ponytail: still full of live entries means a huge job; dropping everything is cheaper than an LRU.
		if len(c.m) >= ttlCacheMaxEntries {
			clear(c.m)
		}
	}
	c.m[key] = ttlEntry[V]{val: val, expires: now.Add(c.ttl)}
}

// reset empties the cache (tests).
func (c *ttlCache[V]) reset() {
	c.mu.Lock()
	clear(c.m)
	c.mu.Unlock()
}
