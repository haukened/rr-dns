package resolver

import (
	"context"
	"net"
	"time"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

// UpstreamClient defines an interface for DNS upstream resolution.
// Implementations of this interface are responsible for sending DNS queries
// to an upstream server and returning the corresponding DNS response.
// The Resolve method takes a context for cancellation and timeout control,
// as well as a DNSQuery object, and returns a DNSResponse or an error.
type UpstreamClient interface {
	Resolve(ctx context.Context, query domain.DNSQuery, now time.Time) (domain.DNSResponse, error)
}

// Blocklist defines an interface for checking whether a DNS query is blocked.
// Implementations should provide logic to determine if a given DNSQuery
// should be considered blocked, typically for filtering or security purposes.
type Blocklist interface {
	// current no-op. Future roadmap for blocking will expand this interface.
	IsBlocked(q domain.DNSQuery) bool
}

// Cache defines the interface for a DNS resource record cache.
// It provides methods to create a new cache, store, retrieve, and delete records,
// as well as to query cache statistics and keys.
//
// Methods:
//   - New(size int): Creates a new cache with the specified size.
//   - Set(record *domain.ResourceRecord): Stores a resource record in the cache.
//   - Get(key string): Retrieves resource records by key, returning the records and a boolean indicating existence.
//   - Delete(key string): Removes a resource record from the cache by key.
//   - Len(): Returns the number of cache entries currently stored in the cache.
//   - Keys(): Returns a slice of all keys currently stored in the cache.
type Cache interface {
	Set(record []domain.ResourceRecord) error
	Get(key string) ([]domain.ResourceRecord, bool)
	Delete(key string)
	Len() int
	Keys() []string
}

// DNSResponder defines an interface for handling DNS queries and generating responses.
// Implementations of this interface process DNS requests, abstracting away network protocol details.
// The HandleRequest method receives the query, client address, and context, and returns a DNS response.
type DNSResponder interface {
	// HandleRequest processes a DNS query and returns a DNS response.
	// The transport handles all network protocol details - the handler only sees domain objects.
	HandleRequest(ctx context.Context, query domain.DNSQuery, clientAddr net.Addr) domain.DNSResponse
}

// ZoneCache defines the interface for in-memory authoritative record storage with value-based records
type ZoneCache interface {
	// Find returns authoritative resource records matching the DNS query (value-based)
	FindRecords(query domain.DNSQuery) ([]domain.ResourceRecord, bool)

	// PutZone replaces all records for a zone with new records (value-based)
	PutZone(zoneRoot string, records []domain.ResourceRecord)

	// RemoveZone removes all records for a zone
	RemoveZone(zoneRoot string)

	// Zones returns a list of all zone roots currently cached
	Zones() []string

	// Count returns the total number of records across all zones
	Count() int
}
