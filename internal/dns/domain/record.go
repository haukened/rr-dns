package domain

import (
	"fmt"
	"time"
)

// ResourceRecord represents a DNS resource record (RR) used for caching.
// These records are stored in memory with an expiration time (ExpiresAt).
type ResourceRecord struct {
	Name      string
	Type      RRType
	Class     RRClass
	ExpiresAt time.Time
	Data      []byte
}

// NewResourceRecord constructs a ResourceRecord and validates its fields.
func NewResourceRecord(name string, rrtype RRType, class RRClass, ttl uint32, data []byte) (ResourceRecord, error) {
	rr := ResourceRecord{
		Name:      name,
		Type:      rrtype,
		Class:     class,
		ExpiresAt: time.Now().Add(time.Duration(ttl) * time.Second),
		Data:      data,
	}
	if err := rr.Validate(); err != nil {
		return ResourceRecord{}, err
	}
	return rr, nil
}

// Validate checks whether the ResourceRecord fields are valid.
func (rr ResourceRecord) Validate() error {
	if rr.Name == "" {
		return fmt.Errorf("record name must not be empty")
	}
	if !rr.Type.IsValid() {
		return fmt.Errorf("invalid RRType: %d", rr.Type)
	}
	if !rr.Class.IsValid() {
		return fmt.Errorf("invalid RRClass: %d", rr.Class)
	}
	return nil
}

// TTLRemaining returns the remaining TTL duration until the record expires.
func (rr ResourceRecord) TTLRemaining() time.Duration {
	return time.Until(rr.ExpiresAt)
}

// CacheKey returns a cache key string derived from the record's name, type, and class.
func (rr ResourceRecord) CacheKey() string {
	return generateCacheKey(rr.Name, rr.Type, rr.Class)
}
