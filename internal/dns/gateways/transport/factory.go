package transport

import (
	"context"
	"fmt"

	"github.com/haukened/rr-dns/internal/dns/common/log"
	"github.com/haukened/rr-dns/internal/dns/gateways/wire"
	"github.com/haukened/rr-dns/internal/dns/services/resolver"
)

// ServerTransport defines the interface for DNS transport implementations.
// This interface is defined here temporarily until we have a higher-level
// coordination layer (e.g., in cmd/rr-dnsd).
type ServerTransport interface {
	Start(ctx context.Context, handler resolver.DNSResponder) error
	Stop() error
	Address() string
}

// NewTransport creates a new transport instance based on the specified type.
// This factory function allows for easy extension to support additional transport
// protocols in the future while maintaining a consistent interface.
func NewTransport(transportType TransportType, addr string, codec wire.DNSCodec, logger log.Logger) (ServerTransport, error) {
	switch transportType {
	case TransportUDP:
		return NewUDPTransport(addr, codec, logger), nil

	case TransportDoH:
		return nil, fmt.Errorf("DNS over HTTPS transport not yet implemented")

	case TransportDoT:
		return nil, fmt.Errorf("DNS over TLS transport not yet implemented")

	case TransportDoQ:
		return nil, fmt.Errorf("DNS over QUIC transport not yet implemented")

	default:
		return nil, fmt.Errorf("unsupported transport type: %s", transportType)
	}
}

// GetSupportedTransports returns a list of currently supported transport types.
func GetSupportedTransports() []TransportType {
	return []TransportType{
		TransportUDP,
		// Future implementations will be added here:
		// TransportDoH,
		// TransportDoT,
		// TransportDoQ,
	}
}

// IsTransportSupported checks if a given transport type is currently supported.
func IsTransportSupported(transportType TransportType) bool {
	supported := GetSupportedTransports()
	for _, t := range supported {
		if t == transportType {
			return true
		}
	}
	return false
}
