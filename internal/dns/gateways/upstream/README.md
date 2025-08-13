# Upstream DNS Resolver

This package provides a highly configurable upstream DNS resolver infrastructure component for forwarding DNS queries to external DNS servers. The implementation follows CLEAN architecture principles with comprehensive dependency injection for maximum testability.

## Overview

The `upstream.Resolver` implements DNS forwarding functionality with:

- **Configurable Resolution Strategies** - Serial or parallel server attempts
- **Complete Dependency Injection** - All external dependencies injectable for testing
- **Context-Aware Operations** - Full context cancellation and timeout support
- **Standardized Error Handling** - Consistent error messages and wrapping
- **Production-Ready Design** - Handles network failures, timeouts, and edge cases

## Architecture

### CLEAN Architecture Compliance

- **Infrastructure Layer**: Handles low-level networking and DNS wire protocol
- **Codec Dependency**: Depends on `wire.DNSCodec` (gateways/wire) for DNS message encoding/decoding
- **No Upward Dependencies**: Zero dependencies on service or application layers
- **Testable Design**: All external interactions are injectable and mockable

### Key Components

```go
type Resolver struct {
    servers  []string        // Upstream DNS servers
    timeout  time.Duration   // Default query timeout
    codec    wire.DNSCodec   // DNS encoding/decoding
    parallel bool            // Resolution strategy
    dial     DialFunc        // Network connection function
}
```

## Usage

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/haukened/rr-dns/internal/dns/domain"
    "github.com/haukened/rr-dns/internal/dns/gateways/upstream"
    "github.com/haukened/rr-dns/internal/dns/gateways/wire"
)

func main() {
    // Create resolver with minimal configuration
    resolver, err := upstream.NewResolver(upstream.Options{
        Servers:  []string{"1.1.1.1:53", "1.0.0.1:53"},
        Timeout:  5 * time.Second,
        Parallel: true, // Enable parallel resolution
        Codec:    myDNSCodec, // Implement wire.DNSCodec
    })
    if err != nil {
        // Handle configuration error
        return
    }
    
    // Create DNS query
    query, _ := domain.NewQuestion(1234, "example.com.", domain.RRTypeA, domain.RRClassIN)
    
    // Resolve with context
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    now := time.Now()
    answers, err := resolver.Resolve(ctx, query, now)
    if err != nil {
        // Handle resolution error
        return
    }
    
    // Process response
    fmt.Printf("Query resolved with %d answers\n", len(answers))
}
```

### Advanced Configuration

```go
// Custom dial function for testing or special network requirements
customDial := func(ctx context.Context, network, address string) (net.Conn, error) {
    // Custom connection logic
    return net.DialTimeout(network, address, 2*time.Second)
}

resolver, err := upstream.NewResolver(upstream.Options{
    Servers:  []string{"192.168.1.1:53", "10.0.0.1:53"},
    Timeout:  3 * time.Second,
    Parallel: false, // Use serial resolution
    Codec:    myCodec, // wire.DNSCodec
    Dial:     customDial, // Custom network behavior
})
```

## Configuration Options

### Required Parameters

- **`Servers`**: List of upstream DNS servers in `host:port` format
- **`Codec`**: Implementation of `domain.DNSCodec` for DNS message handling

### Optional Parameters

- **`Timeout`**: Query timeout duration (default: 5 seconds)
- **`Parallel`**: Enable parallel resolution strategy (default: false)
- **`Dial`**: Custom network dial function (default: standard UDP dialer)

## Resolution Strategies

### Serial Resolution (`Parallel: false`)

Queries servers sequentially until one succeeds:

```
Server1 → fail → Server2 → fail → Server3 → success
```

**Benefits**: Lower network usage, predictable server precedence  
**Use Case**: When server order matters or network resources are limited

### Parallel Resolution (`Parallel: true`)

Queries all servers simultaneously:

```
Server1 ┐
Server2 ├─> First success wins
Server3 ┘
```

**Benefits**: Faster response times, better fault tolerance  
**Use Case**: When speed is critical and network resources are available

## Error Handling

### Standardized Error Messages

All errors use consistent, predefined messages:

```go
const (
    errNoServersProvided = "no upstream DNS servers provided"
    errCodecRequired     = "DNS codec is required"
    errConnDeadline      = "failed to set connection deadline: %w"
    errServerFailed      = "server %s: %w"
    errAllServersFailed  = "all %d upstream servers failed"
    errQueryTimeout      = "query timeout after %v"
    errFailedToConnect   = "failed to connect: %w"
    errEncodeFailed      = "encode failed: %w"
    errWriteFailed       = "write failed: %w"
    errReadFailed        = "read failed: %w"
)
```

### Error Scenarios

- **Configuration Errors**: Invalid options during construction
- **Network Failures**: Connection timeouts, unreachable servers
- **Protocol Errors**: DNS encoding/decoding failures
- **Context Cancellation**: Timeout or manual cancellation
- **All Servers Failed**: No upstream server could resolve the query

## Dependency Injection

### Testing Interface

The resolver is designed for comprehensive testing through dependency injection:

```go
// Mock codec for testing
type mockCodec struct {
    encodeFunc func(domain.Question) ([]byte, error)
    decodeFunc func([]byte, uint16, time.Time) (domain.DNSResponse, error)
}

func (m *mockCodec) EncodeQuery(q domain.Question) ([]byte, error) {
    return m.encodeFunc(q)
}

func (m *mockCodec) DecodeResponse(data []byte, id uint16, now time.Time) (domain.DNSResponse, error) {
    return m.decodeFunc(data, id, now)
}

// Mock dial function for network testing
func mockDial(ctx context.Context, network, address string) (net.Conn, error) {
    return &mockConn{}, nil // Return controllable connection
}

// Test with complete control
resolver, _ := upstream.NewResolver(upstream.Options{
    Servers:  []string{"test:53"},
    Codec:    &mockCodec{...}, // wire.DNSCodec
    Dial:     mockDial,
    Parallel: true,
})
```

### Testable Scenarios

- ✅ **Constructor validation**: Empty servers, missing codec
- ✅ **Network failures**: Connection errors, timeouts
- ✅ **Protocol failures**: Encoding/decoding errors
- ✅ **Context handling**: Cancellation, deadlines
- ✅ **Strategy testing**: Serial vs parallel behavior
- ✅ **Error aggregation**: Multiple server failures

## Performance Characteristics

### Context Management

- **Automatic Timeout**: Applies default timeout if context has no deadline
- **Deadline Preservation**: Respects existing context deadlines
- **Proper Cleanup**: Cancels operations when context is done

### Memory Efficiency

- **Minimal Allocations**: Reuses buffers where possible
- **Bounded Goroutines**: Goroutine count limited by server count
- **Channel Cleanup**: Proper cleanup prevents goroutine leaks

### Network Efficiency

- **UDP Transport**: Lightweight DNS communication protocol
- **Connection Reuse**: Efficient connection management per query
- **Concurrent Safety**: Thread-safe for multiple simultaneous queries

## Integration

### Service Layer Integration

```go
// Interface defined in service layer (e.g., internal/dns/services/resolver)
// Upstream resolver implements this interface:

// UpstreamClient interface (defined in service layer)
// - Resolve(ctx context.Context, query domain.Question, now time.Time) ([]domain.ResourceRecord, error)

// Resolver implements the interface (Dependency Inversion Principle)
var _ resolver.UpstreamClient = (*Resolver)(nil)
```

### Repository Pattern

The resolver implements the upstream resolution interface defined in the service layer, maintaining clean architectural boundaries and following the Dependency Inversion Principle.

## Limitations

### Current Scope

- **UDP Only**: No TCP fallback for truncated responses
- **Single Protocol**: IPv4 UDP transport only
- **Basic Features**: Core DNS resolution without extensions

### Future Enhancements

These limitations are by design for the current implementation scope. Future versions may include:

- TCP fallback for large responses
- IPv6 transport support
- EDNS0 extension support
- Connection pooling and reuse

The architecture supports these enhancements through the existing injection points without breaking changes.

## Testing

### Test Coverage

Run tests with:

```bash
# Unit tests only
go test ./internal/dns/gateways/upstream/ -short

# Include integration tests (requires network)
go test ./internal/dns/gateways/upstream/ -v
```

### Test Categories

- **Unit Tests**: Constructor validation, error handling, strategy logic
- **Integration Tests**: Real network connectivity (skipped with `-short`)
- **Mock Tests**: Complete isolation using dependency injection
- **Error Path Tests**: Comprehensive failure scenario coverage

The dependency injection design enables 100% test coverage of all code paths and error conditions.
