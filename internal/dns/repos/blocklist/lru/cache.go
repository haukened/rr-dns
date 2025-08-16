package lru

import (
	"sync/atomic"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/haukened/rr-dns/internal/dns/domain"
	"github.com/haukened/rr-dns/internal/dns/repos/blocklist"
)

// decisionCache is an LRU-backed implementation of blocklist.DecisionCache.
// It tracks basic metrics: hits, misses, and evictions.
type decisionCache struct {
	lru       *lru.Cache[string, domain.BlockDecision]
	hits      uint64
	misses    uint64
	evictions uint64
}

// disabledCache is a no-op DecisionCache used when size <= 0.
type disabledCache struct{}

// New creates a new DecisionCache with the given capacity. If size <= 0, a
// disabled no-op cache is returned that always misses and tracks no metrics.
func New(size int) (blocklist.DecisionCache, error) {
	if size <= 0 {
		return &disabledCache{}, nil
	}

	var dc decisionCache
	// Use NewWithEvict to observe evictions, including Purge-induced ones.
	cache, err := lru.NewWithEvict(size, func(_ string, _ domain.BlockDecision) {
		atomic.AddUint64(&dc.evictions, 1)
	})
	if err != nil {
		return nil, err
	}
	dc.lru = cache
	return &dc, nil
}

// Get looks up a decision by name. When found, increments hits; otherwise increments misses.
func (c *decisionCache) Get(name string) (domain.BlockDecision, bool) {
	if val, ok := c.lru.Get(name); ok {
		atomic.AddUint64(&c.hits, 1)
		return val, true
	}
	atomic.AddUint64(&c.misses, 1)
	var zero domain.BlockDecision
	return zero, false
}

// Put stores a decision by name.
func (c *decisionCache) Put(name string, d domain.BlockDecision) {
	c.lru.Add(name, d)
}

// Len returns the number of entries in the cache.
func (c *decisionCache) Len() int { return c.lru.Len() }

// Purge clears all entries. Evictions are counted via the eviction callback.
func (c *decisionCache) Purge() { c.lru.Purge() }

// Stats returns cumulative hit/miss/eviction counters.
func (c *decisionCache) Stats() (hits, misses, evictions uint64) {
	return atomic.LoadUint64(&c.hits), atomic.LoadUint64(&c.misses), atomic.LoadUint64(&c.evictions)
}

// disabledCache implementation

func (d *disabledCache) Get(string) (domain.BlockDecision, bool) {
	var zero domain.BlockDecision
	return zero, false
}

func (d *disabledCache) Put(string, domain.BlockDecision) {}

func (d *disabledCache) Len() int { return 0 }

func (d *disabledCache) Purge() {}

func (d *disabledCache) Stats() (uint64, uint64, uint64) { return 0, 0, 0 }

var _ blocklist.DecisionCache = (*decisionCache)(nil)
var _ blocklist.DecisionCache = (*disabledCache)(nil)
