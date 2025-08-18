package lru

import (
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/haukened/rr-dns/internal/dns/domain"
	"github.com/haukened/rr-dns/internal/dns/repos/blocklist"
)

// decisionCache is an LRU-backed implementation of blocklist.DecisionCache.
// It tracks basic metrics: hits, misses, and evictions.
type decisionCache struct {
	lru *lru.Cache[string, domain.BlockDecision]
}

// disabledCache is a no-op DecisionCache used when size <= 0.
type disabledCache struct{}

// newLRU constructs the underlying LRU cache. It is a var to allow tests to
// stub error paths from the third-party constructor.
var newLRU = lru.New[string, domain.BlockDecision]

// New creates a new DecisionCache with the given capacity. If size <= 0, a
// disabled no-op cache is returned that always misses and tracks no metrics.
func New(size int) (blocklist.DecisionCache, error) {
	if size <= 0 {
		return &disabledCache{}, nil
	}

	cache, err := newLRU(size)
	if err != nil {
		return nil, err
	}
	return &decisionCache{lru: cache}, nil
}

// Get looks up a decision by name. When found, increments hits; otherwise increments misses.
func (c *decisionCache) Get(name string) (domain.BlockDecision, bool) {
	if val, ok := c.lru.Get(name); ok {
		return val, true
	}
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

// disabledCache implementation

func (d *disabledCache) Get(string) (domain.BlockDecision, bool) {
	var zero domain.BlockDecision
	return zero, false
}

func (d *disabledCache) Put(string, domain.BlockDecision) {}

func (d *disabledCache) Len() int { return 0 }

func (d *disabledCache) Purge() {}

var _ blocklist.DecisionCache = (*decisionCache)(nil)
var _ blocklist.DecisionCache = (*disabledCache)(nil)
