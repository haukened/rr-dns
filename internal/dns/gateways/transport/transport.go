// Package transport provides network transport abstractions for DNS server implementations.
// It handles the conversion between wire format and domain objects, allowing the service
// layer to work purely with domain types while supporting multiple transport protocols.
package transport

import (
	"context"
	"net"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

// ServerTransport defines the interface for DNS server transport implementations.
// Different transport types (UDP, DoH, DoT, DoQ) can implement this interface
// while providing the same request handling contract to the service layer.
type ServerTransport interface {
	// Start begins listening for requests and handling them via the provided handler.
	// The transport handles all network protocol concerns and wire format conversion.
	Start(ctx context.Context, handler RequestHandler) error

	// Stop gracefully shuts down the transport, closing connections and cleaning up resources.
	Stop() error

	// Address returns the network address the transport is bound to.
	Address() string
}

// RequestHandler defines how the service layer receives and processes DNS requests.
// The transport layer converts wire format to domain objects before calling this interface,
// and converts the response back to wire format for transmission.
type RequestHandler interface {
	// HandleRequest processes a DNS query and returns a DNS response.
	// The transport handles all network protocol details - the handler only sees domain objects.
	HandleRequest(ctx context.Context, query domain.DNSQuery, clientAddr net.Addr) domain.DNSResponse
}

// TransportType represents the different types of DNS transport protocols supported.
type TransportType string

const (
	// TransportUDP represents standard DNS over UDP (RFC 1035)
	TransportUDP TransportType = "udp"

	// TransportDoH represents DNS over HTTPS (RFC 8484) - future implementation
	TransportDoH TransportType = "doh"

	// TransportDoT represents DNS over TLS (RFC 7858) - future implementation
	TransportDoT TransportType = "dot"

	// TransportDoQ represents DNS over QUIC (RFC 9250) - future implementation
	TransportDoQ TransportType = "doq"
)
