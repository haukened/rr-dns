package domain

import (
	"fmt"
	"strings"
	"time"
)

// RRType represents a DNS resource record type (e.g. A, AAAA, MX).
// See IANA DNS Parameters for assigned codes.
type RRType uint16

// RRClass represents a DNS class (usually IN for Internet).
type RRClass uint16

// RCode represents a DNS response code indicating the result of a query.
type RCode uint8

// DNSQuery represents a DNS query section containing a question for resolution.
type DNSQuery struct {
	ID    uint16
	Name  string
	Type  RRType
	Class RRClass
}

// DNSResponse represents a full DNS response including answer, authority, and additional sections.
type DNSResponse struct {
	ID         uint16
	RCode      RCode
	Answers    []ResourceRecord
	Authority  []ResourceRecord
	Additional []ResourceRecord
}

// ResourceRecord represents a DNS resource record (RR) used for caching.
// These records are stored in memory with an expiration time (ExpiresAt).
type ResourceRecord struct {
	Name      string
	Type      RRType
	Class     RRClass
	ExpiresAt time.Time
	Data      []byte
}

// AuthoritativeRecord represents a DNS record served from a zone file.
// These records do not expire from memory, and their TTL is preserved for wire responses.
type AuthoritativeRecord struct {
	Name  string
	Type  RRType
	Class RRClass
	TTL   uint32
	Data  []byte
}

// NewDNSQuery constructs a DNSQuery and validates its fields.
func NewDNSQuery(id uint16, name string, rrtype RRType, class RRClass) (DNSQuery, error) {
	q := DNSQuery{
		ID:    id,
		Name:  name,
		Type:  rrtype,
		Class: class,
	}
	if err := q.Validate(); err != nil {
		return DNSQuery{}, err
	}
	return q, nil
}

// Validate checks whether the DNSQuery fields are structurally and semantically valid.
func (q DNSQuery) Validate() error {
	if q.Name == "" {
		return fmt.Errorf("query name must not be empty")
	}
	if !q.Type.IsValid() {
		return fmt.Errorf("unsupported RRType: %d", q.Type)
	}
	if !q.Class.IsValid() {
		return fmt.Errorf("unsupported RRClass: %d", q.Class)
	}
	return nil
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

// IsValid returns true if the RRType is one of the supported types.
func (t RRType) IsValid() bool {
	switch t {
	case 1, 2, 5, 6, 12, 15, 16, 28, 33, 41, 255, 257:
		return true
	default:
		return false
	}
}

// IsValid returns true if the RRClass is one of the supported classes.
func (c RRClass) IsValid() bool {
	switch c {
	case 1, 3, 4, 255:
		return true
	default:
		return false
	}
}

// IsValid returns true if the RCode is within the supported response code range.
func (r RCode) IsValid() bool {
	return r <= 10
}

// generateCacheKey returns a string key derived from name, type, and class for caching.
func generateCacheKey(name string, t RRType, c RRClass) string {
	return fmt.Sprintf("%s:%d:%d", name, t, c)
}

// CacheKey returns a cache key string derived from the query's name, type, and class.
func (q DNSQuery) CacheKey() string {
	return generateCacheKey(q.Name, q.Type, q.Class)
}

// CacheKey returns a cache key string derived from the record's name, type, and class.
func (rr ResourceRecord) CacheKey() string {
	return generateCacheKey(rr.Name, rr.Type, rr.Class)
}

// RRTypeFromString converts a record type string to its corresponding RRType value.
func RRTypeFromString(s string) RRType {
	switch strings.ToUpper(s) {
	case "A":
		return 1
	case "NS":
		return 2
	case "CNAME":
		return 5
	case "SOA":
		return 6
	case "PTR":
		return 12
	case "MX":
		return 15
	case "TXT":
		return 16
	case "AAAA":
		return 28
	case "SRV":
		return 33
	case "OPT":
		return 41
	case "ANY":
		return 255
	case "CAA":
		return 257
	default:
		return 0 // invalid/unknown
	}
}

// NewResourceRecordFromAuthoritative converts an AuthoritativeRecord into a ResourceRecord with expiration.
func NewResourceRecordFromAuthoritative(ar AuthoritativeRecord, now time.Time) ResourceRecord {
	return ResourceRecord{
		Name:      ar.Name,
		Type:      ar.Type,
		Class:     ar.Class,
		ExpiresAt: now.Add(time.Duration(ar.TTL) * time.Second),
		Data:      ar.Data,
	}
}

// Validate checks whether the AuthoritativeRecord fields are valid.
func (ar AuthoritativeRecord) Validate() error {
	if ar.Name == "" {
		return fmt.Errorf("record name must not be empty")
	}
	if !ar.Type.IsValid() {
		return fmt.Errorf("invalid RRType: %d", ar.Type)
	}
	if !ar.Class.IsValid() {
		return fmt.Errorf("invalid RRClass: %d", ar.Class)
	}
	return nil
}
