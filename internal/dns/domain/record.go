package domain

import (
	"fmt"
	"time"

	"github.com/haukened/rr-dns/internal/dns/common/utils"
)

// ResourceRecord represents a DNS resource record (RR) used for caching.
// These records are stored in memory with an expiration time (ExpiresAt).
type ResourceRecord struct {
	Name      string
	Type      RRType
	Class     RRClass
	ttl       uint32
	expiresAt *time.Time // nil if record is authoritative
	Data      []byte     // Wire-Encoded representation of the record
	Text      string     // Human-readable representation of the record
}

// NewAuthoritativeResourceRecord constructs an authoritative ResourceRecord (non-expiring).
func NewAuthoritativeResourceRecord(name string, rrtype RRType, class RRClass, ttl uint32, data []byte, text string) (ResourceRecord, error) {
	rr := ResourceRecord{
		Name:      utils.CanonicalDNSName(name),
		Type:      rrtype,
		Class:     class,
		ttl:       ttl,
		expiresAt: nil,
		Data:      data,
		Text:      text,
	}
	if err := rr.Validate(); err != nil {
		return ResourceRecord{}, err
	}
	return rr, nil
}

// NewCachedResourceRecord constructs a cached ResourceRecord with an expiration time.
// accepts the current time to calculate the expiration, keeping domain logic encapsulated.
func NewCachedResourceRecord(name string, rrtype RRType, class RRClass, ttl uint32, data []byte, text string, now time.Time) (ResourceRecord, error) {
	exp := now.Add(time.Duration(ttl) * time.Second)
	rr := ResourceRecord{
		Name:      utils.CanonicalDNSName(name),
		Type:      rrtype,
		Class:     class,
		ttl:       ttl,
		expiresAt: &exp,
		Data:      data,
		Text:      text,
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
	if rr.Text == "" && len(rr.Data) == 0 {
		return fmt.Errorf("either Text or Data must be set")
	}
	return nil
}

// TTLRemaining returns the remaining TTL duration until the record expires.
func (rr ResourceRecord) TTLRemaining() time.Duration {
	if rr.expiresAt == nil {
		return time.Duration(rr.ttl) * time.Second
	}
	ttl := time.Until(*rr.expiresAt)
	if ttl < 0 {
		return 0
	}
	return ttl
}

// CacheKey returns a cache key string derived from the record's name, type, and class.
func (rr ResourceRecord) CacheKey() string {
	return GenerateCacheKey(rr.Name, rr.Type, rr.Class)
}

// TTL returns the effective TTL value for wire encoding.
// If the record is cached (expiresAt set), it computes the remaining TTL.
// If it's authoritative (expiresAt is nil), it returns the original TTL.
func (rr ResourceRecord) TTL() uint32 {
	// zone records are authoritative, so they do not have an expiration time
	// so we can confidently return the original TTL
	if rr.expiresAt == nil {
		return rr.ttl
	}
	// cached resource records have an expiration time
	// because for messages returned by an upstream DNS server
	// TTL only has meaning in the context of the time of arrival
	ttl := time.Until(*rr.expiresAt).Seconds()
	if ttl <= 0 {
		return 0
	}
	return uint32(ttl)
}

// IsExpired returns true if the record has an expiration time that has passed.
func (rr ResourceRecord) IsExpired() bool {
	if rr.expiresAt == nil {
		return false
	}
	return time.Now().After(*rr.expiresAt)
}

// IsAuthoritative returns true if the record has no expiration time set.
func (rr ResourceRecord) IsAuthoritative() bool {
	return rr.expiresAt == nil
}
