// Package transport provides network transport abstractions for DNS server implementations.
// It handles the conversion between wire format and domain objects, allowing the service
// layer to work purely with domain types while supporting multiple transport protocols.
package transport

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
