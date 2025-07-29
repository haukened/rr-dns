package repo

import (
	"context"
	"time"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

// UpstreamResolver defines the interface for resolving DNS queries via upstream servers.
// This interface abstracts the infrastructure concerns of DNS forwarding while maintaining
// clean separation between the service layer and external DNS dependencies.
type UpstreamResolver interface {
	// Resolve forwards a DNS query to upstream servers and returns the response.
	// Returns an error if the query fails or times out.
	Resolve(ctx context.Context, query domain.DNSQuery) (domain.DNSResponse, error)

	// ResolveWithTimeout forwards a DNS query with a specific timeout.
	// This allows service layer to control timeout behavior without knowing infrastructure details.
	ResolveWithTimeout(query domain.DNSQuery, timeout time.Duration) (domain.DNSResponse, error)

	// Health returns the health status of the upstream resolver.
	// This can be used for monitoring and failover decisions.
	Health() error
}
