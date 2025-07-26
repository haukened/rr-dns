package domain

import "fmt"

// DNSQuery represents a DNS query section containing a question for resolution.
type DNSQuery struct {
	ID    uint16
	Name  string
	Type  RRType
	Class RRClass
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

// CacheKey returns a cache key string derived from the query's name, type, and class.
func (q DNSQuery) CacheKey() string {
	return generateCacheKey(q.Name, q.Type, q.Class)
}
