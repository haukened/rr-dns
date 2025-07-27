package memcache

import (
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/haukened/rr-dns/internal/dns/domain"
)

// dnsCache is an in-memory TTL-aware cache using an LRU strategy to store DNS resource records.
// It provides methods to add, retrieve, and automatically evict expired entries.
type dnsCache struct {
	lru *lru.Cache[string, *domain.ResourceRecord]
}

// New returns a new dnsCache instance of the given size using an LRU backing store.
func New(size int) (*dnsCache, error) {
	cache, err := lru.New[string, *domain.ResourceRecord](size)
	if err != nil {
		return nil, err
	}
	return &dnsCache{lru: cache}, nil
}

// Set adds a resource record to the cache with the given TTL. It replaces any existing entry for the key.
func (c *dnsCache) Set(record *domain.ResourceRecord) {
	c.lru.Add(record.CacheKey(), record)
}

// Get retrieves a resource record from the cache if present and not expired.
// If the record is expired, it is evicted and the method returns false.
func (c *dnsCache) Get(key string) (*domain.ResourceRecord, bool) {
	if record, found := c.lru.Get(key); found {
		if time.Now().Before(record.ExpiresAt) {
			return record, true
		}
		c.lru.Remove(key)
	}
	return nil, false
}

// Delete removes the entry for the given key from the cache.
func (c *dnsCache) Delete(key string) {
	c.lru.Remove(key)
}

// Len returns the number of items currently stored in the cache.
func (c *dnsCache) Len() int {
	return c.lru.Len()
}

// Keys returns a slice of all current cache keys.
func (c *dnsCache) Keys() []string {
	return c.lru.Keys()
}
