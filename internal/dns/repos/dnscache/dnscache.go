package dnscache

import (
	"errors"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/haukened/rr-dns/internal/dns/domain"
	"github.com/haukened/rr-dns/internal/dns/services/resolver"
)

var (
	ErrMultipleKeys = errors.New("multiple records with different keys provided")
)

// dnsCache is an in-memory TTL-aware cache using an LRU strategy to store DNS resource records.
// It provides methods to add, retrieve, and automatically evict expired entries.
// Each cache key can store multiple resource records, as DNS queries often return multiple records.
type dnsCache struct {
	lru *lru.Cache[string, []domain.ResourceRecord]
}

// New returns a new dnsCache instance of the given size using an LRU backing store.
func New(size int) (*dnsCache, error) {
	cache, err := lru.New[string, []domain.ResourceRecord](size)
	if err != nil {
		return nil, err
	}
	return &dnsCache{lru: cache}, nil
}

// Set replaces the existing records for the given key with the provided records.
// all records passed should use the same key
func (c *dnsCache) Set(records []domain.ResourceRecord) error {
	if len(records) == 0 {
		return nil
	}
	// make sure all records have the same cache key
	key := records[0].CacheKey()
	for _, record := range records {
		if record.CacheKey() != key {
			return ErrMultipleKeys
		}
	}
	c.lru.Add(key, records)
	return nil
}

// Get retrieves resource records from the cache if present and not expired.
// If any records are expired, they are removed from the cache.
// Returns all valid (non-expired) records for the key and a boolean indicating if any were found.
func (c *dnsCache) Get(key string) ([]domain.ResourceRecord, bool) {
	if records, found := c.lru.Get(key); found {
		var validRecords []domain.ResourceRecord

		// Filter out expired records
		for _, record := range records {
			if !record.IsExpired() {
				validRecords = append(validRecords, record)
			}
		}

		// Update cache with only valid records or remove if none remain
		if len(validRecords) > 0 {
			c.lru.Add(key, validRecords)
			return validRecords, true
		} else {
			c.lru.Remove(key)
		}
	}
	return nil, false
}

// Delete removes the entry for the given key from the cache.
func (c *dnsCache) Delete(key string) {
	c.lru.Remove(key)
}

// Len returns the number of cache entries (keys) currently stored in the cache.
// Note: Each entry may contain multiple resource records.
func (c *dnsCache) Len() int {
	return c.lru.Len()
}

// Keys returns a slice of all current cache keys.
func (c *dnsCache) Keys() []string {
	return c.lru.Keys()
}

var _ resolver.Cache = (*dnsCache)(nil)
