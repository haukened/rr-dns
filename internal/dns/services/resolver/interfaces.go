package resolver

import (
	"context"
	"net"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

// UpstreamClient defines an interface for DNS upstream resolution.
// Implementations of this interface are responsible for sending DNS queries
// to an upstream server and returning the corresponding DNS response.
// The Resolve method takes a context for cancellation and timeout control,
// as well as a DNSQuery object, and returns a DNSResponse or an error.
type UpstreamClient interface {
	Resolve(ctx context.Context, query domain.DNSQuery) (domain.DNSResponse, error)
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
//   - Get(key string): Retrieves a resource record by key, returning the record and a boolean indicating existence.
//   - Delete(key string): Removes a resource record from the cache by key.
//   - Len(): Returns the number of records currently stored in the cache.
//   - Keys(): Returns a slice of all keys currently stored in the cache.
type Cache interface {
	Set(record *domain.ResourceRecord)
	Get(key string) (*domain.ResourceRecord, bool)
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

// ServerTransport defines the interface for DNS server transport implementations.
// Different transport types (UDP, DoH, DoT, DoQ) can implement this interface
// while providing the same request handling contract to the service layer.
type ServerTransport interface {
	// Start begins listening for requests and handling them via the provided handler.
	// The transport handles all network protocol concerns and wire format conversion.
	Start(ctx context.Context, handler DNSResponder) error

	// Stop gracefully shuts down the transport, closing connections and cleaning up resources.
	Stop() error

	// Address returns the network address the transport is bound to.
	Address() string
}

// ZoneCache defines the interface for in-memory authoritative record storage
type ZoneCache interface {
	// Find returns authoritative records matching the FQDN and RRType
	Find(fqdn string, rrType domain.RRType) ([]*domain.AuthoritativeRecord, bool)

	// ReplaceZone replaces all records for a zone with new records
	ReplaceZone(zoneRoot string, records []*domain.AuthoritativeRecord) error

	// RemoveZone removes all records for a zone
	RemoveZone(zoneRoot string) error

	// All returns a snapshot of all zone data (for admin/diagnostic purposes)
	All() map[string][]*domain.AuthoritativeRecord

	// Zones returns a list of all zone roots currently cached
	Zones() []string

	// Count returns the total number of records across all zones
	Count() int
}
