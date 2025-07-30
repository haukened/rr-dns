# Transport Package

The `transport` package provides network transport abstractions for DNS server implementations. It handles the conversion between DNS wire format and domain objects, allowing the service layer to work purely with domain types while supporting multiple transport protocols.

## Architecture

The transport layer sits between the network and service layers:

```
Network Packets → Transport Layer → Service Layer (Domain Objects)
                      ↓
               Wire Format Codec
```

## Design Principles

- **Protocol Abstraction**: Service layer is unaware of transport protocol details
- **Wire Format Handling**: Transport layer handles encoding/decoding using domain codecs
- **Extensibility**: New transport protocols can be added without changing service logic
- **Interface Compliance**: Implements interfaces defined in the service layer (Dependency Inversion Principle)

## Implementation Details

The transport package implements interfaces defined in the service layer:

```go
// Interfaces defined in service layer (e.g., internal/dns/services/resolver)
// Transport implementations comply with these contracts:

// ServerTransport interface (defined in service layer)
// - Start(ctx context.Context, handler DNSResponder) error
// - Stop() error  
// - Address() string

// DNSResponder interface (defined in service layer)
// - HandleRequest(ctx context.Context, query domain.DNSQuery, clientAddr net.Addr) domain.DNSResponse
```

## Current Implementation

### UDP Transport
- **Protocol**: Standard DNS over UDP (RFC 1035)
- **Packet Size**: 512 bytes (standard DNS UDP limit)
- **Concurrency**: Each packet handled in separate goroutine
- **Graceful Shutdown**: Context cancellation and stop channel support
- **Error Handling**: Comprehensive logging of decode/encode failures

## Future Transport Implementations

The architecture is designed to support additional protocols:

- **DNS over HTTPS (DoH)**: RFC 8484 - HTTP/2 based transport
- **DNS over TLS (DoT)**: RFC 7858 - TLS encrypted DNS
- **DNS over QUIC (DoQ)**: RFC 9250 - QUIC based transport

## Usage Example

```go
// Import the service layer interface definitions
import "github.com/haukened/rr-dns/internal/dns/services/resolver"

// Create transport (implements resolver.ServerTransport)
transport := transport.NewUDPTransport(":53", wire.UDP)

// Create service handler (implements resolver.DNSResponder)
handler := &dnsService{...}

// Start transport
err := transport.Start(ctx, handler)
if err != nil {
    log.Fatal("Failed to start transport", err)
}

// Graceful shutdown
defer transport.Stop()
```

## Request Flow

1. **Network Packet Arrives** (UDP/HTTPS/TLS)
2. **Transport Receives Raw Data**
3. **Transport Decodes** using wire codec → `domain.DNSQuery`
4. **Transport Calls** `handler.HandleRequest(query, clientAddr)`
5. **Service Processes** query → `domain.DNSResponse`
6. **Transport Encodes** response using wire codec → `[]byte`
7. **Transport Sends** response back to client

## Benefits

- **Protocol Independence**: Same service logic works with any transport
- **Testing**: Easy to mock transport and service layers independently
- **Future-Proof**: New protocols can be added without breaking existing code
- **Performance**: Each transport can optimize for its specific protocol requirements
