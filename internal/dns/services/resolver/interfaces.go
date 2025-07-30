package resolver

import (
	"context"
	"net"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

type UpstreamClient interface {
	Resolve(ctx context.Context, query domain.DNSQuery) (domain.DNSResponse, error)
}

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
