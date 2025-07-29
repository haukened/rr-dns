# Gateways

This directory contains gateway implementations that handle communication with external systems and protocols. Gateways provide abstraction layers for network protocols, external services, and third-party integrations while maintaining clean architectural boundaries.

## Overview

The `gateways` package provides:

- **Protocol abstractions** for network communication (UDP, TCP, HTTP)
- **External service integrations** with upstream DNS servers
- **Wire format handling** for DNS message encoding/decoding
- **Transport layer implementations** supporting multiple DNS protocols

## Architecture Role

In the CLEAN architecture, gateways serve as:

- **Infrastructure Layer**: Handle external system communication
- **Dependency Inversion**: Implement interfaces defined in domain layer
- **Protocol Abstraction**: Hide protocol details from business logic
- **External Boundaries**: Manage interactions with external systems

## Directory Structure

```
gateways/
├── transport/       # DNS transport protocol implementations
├── upstream/        # Upstream DNS server communication
└── wire/           # DNS wire format encoding/decoding
```

## Components

### [Transport (`transport/`)](transport/)

Network transport abstractions for DNS server implementations.

**Key Features:**
- Protocol-agnostic transport layer
- UDP, DoH, DoT, DoQ transport support (UDP implemented)
- Graceful startup and shutdown
- Request/response handling abstraction

**Current Implementation:**
- ✅ UDP Transport (RFC 1035)
- 🚧 DNS over HTTPS (DoH) - Planned
- 🚧 DNS over TLS (DoT) - Planned  
- 🚧 DNS over QUIC (DoQ) - Planned

### [Upstream (`upstream/`)](upstream/)

Upstream DNS resolver for forwarding queries to external DNS servers.

**Key Features:**
- Configurable upstream server lists
- Serial and parallel resolution strategies
- Context-aware operations with timeout support
- Comprehensive dependency injection for testing

**Configuration Options:**
- Multiple upstream servers
- Timeout configuration
- Resolution strategy (serial/parallel)
- Custom network dial functions

### [Wire (`wire/`)](wire/)

DNS wire format encoding and decoding for UDP transport.

**Key Features:**
- RFC 1035 compliant DNS message handling
- Label compression and decompression
- Binary protocol validation
- 100% test coverage including error paths

**Supported Operations:**
- DNS query encoding/decoding
- DNS response encoding/decoding
- Name compression handling
- Wire format validation

## Design Principles

### Protocol Independence
Gateways abstract protocol details from the service layer:

```go
// Service layer works with domain objects
type DNSService struct {
    transport ServerTransport  // Abstract transport
    upstream  QueryResolver    // Abstract upstream resolution
    codec     DNSCodec        // Abstract wire format
}
```

### Interface-Based Architecture
All gateways implement domain-defined interfaces:

```go
// Domain defines the interface
type ServerTransport interface {
    Start(ctx context.Context, handler RequestHandler) error
    Stop() error
    Address() string
}

// Gateway implements the interface
type UDPTransport struct {
    // Implementation details
}
```

### Dependency Injection
Gateways support full dependency injection for testing:

```go
// All external dependencies are injectable
transport := NewUDPTransport(":53", codec)
upstream := NewResolver(Options{
    Servers: []string{"1.1.1.1:53"},
    Codec:   codec,
    Dial:    customDialFunc,  // Injectable for testing
})
```

## Integration Patterns

### Transport Integration
```go
// Create transport with codec
transport := transport.NewUDPTransport(":53", wire.UDP)

// Start with request handler
err := transport.Start(ctx, dnsHandler)
```

### Upstream Resolution
```go
// Create upstream resolver
resolver, err := upstream.NewResolver(upstream.Options{
    Servers:  []string{"1.1.1.1:53", "1.0.0.1:53"},
    Timeout:  5 * time.Second,
    Parallel: true,
    Codec:    wire.UDP,
})

// Resolve queries
response, err := resolver.Resolve(ctx, query)
```

### Wire Format Handling
```go
// Encode query for transmission
queryBytes, err := wire.UDP.EncodeQuery(query)

// Decode response from network
response, err := wire.UDP.DecodeResponse(responseBytes, expectedID)
```

## Error Handling

Gateways provide consistent error handling patterns:

### Transport Errors
- Network binding failures
- Connection timeouts
- Protocol-specific errors

### Upstream Errors
- Server unreachable
- Query timeout
- All servers failed

### Wire Format Errors
- Malformed packets
- Compression pointer errors
- Invalid resource records

## Performance Characteristics

### Transport Performance
- **UDP**: Low latency, minimal overhead
- **Concurrency**: Each request handled in separate goroutine
- **Resource Usage**: Efficient memory allocation

### Upstream Performance
- **Parallel Resolution**: Faster response times
- **Connection Management**: Efficient network usage
- **Context Handling**: Proper timeout management

### Wire Format Performance
- **Zero-copy Operations**: Minimal memory allocations
- **Efficient Encoding**: Optimized byte operations
- **Compression Support**: Reduced packet sizes

## Testing Strategy

All gateways include comprehensive testing:

### Unit Tests
- Interface compliance testing
- Error path validation
- Edge case handling

### Integration Tests
- Real network communication (with `-short` skip flag)
- Protocol compliance validation
- Performance benchmarking

### Mock Testing
- Complete dependency injection
- Isolated component testing
- Predictable test behavior

## Future Enhancements

### Additional Transports

**DNS over HTTPS (DoH) - RFC 8484**
```go
type DoHTransport struct {
    server   *http.Server
    codec    domain.DNSCodec
    endpoint string
}
```

**DNS over TLS (DoT) - RFC 7858**
```go
type DoTTransport struct {
    listener net.Listener
    codec    domain.DNSCodec
    tlsConfig *tls.Config
}
```

**DNS over QUIC (DoQ) - RFC 9250**
```go
type DoQTransport struct {
    listener quic.Listener
    codec    domain.DNSCodec
}
```

### Enhanced Wire Formats

**DNSSEC Support**
- RRSIG record handling
- DNSKEY validation
- Chain of trust verification

**EDNS0 Extensions**
- Extended DNS features
- Larger message sizes
- Additional header fields

## Configuration

Gateway components are configured through the application config:

```yaml
# Transport configuration
transport:
  protocol: "udp"
  address: ":53"
  
# Upstream configuration  
upstream:
  servers: ["1.1.1.1:53", "1.0.0.1:53"]
  timeout: "5s"
  parallel: true

# Wire format options
wire:
  compression: true
  max_size: 512
```

## Dependencies

Gateways use well-established networking libraries:

- **Standard Library**: `net`, `context`, `encoding/binary`
- **Domain Package**: Core DNS types and interfaces
- **Third-party**: Protocol-specific libraries as needed

## Related Directories

- **[Domain](../domain/)**: Core business logic and interface definitions
- **[Common](../common/)**: Shared infrastructure services
- **[Config](../config/)**: Application configuration management
- **[Repos](../repos/)**: Data persistence and storage

Gateways bridge the gap between the clean domain logic and the messy external world, ensuring that protocol complexities don't leak into the business logic while providing robust, testable integrations with external systems.
