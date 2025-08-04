# Transport Package

The `transport` package provides high-performance network transport abstractions for DNS server implementations. It handles the conversion between DNS wire format and domain objects, allowing the service layer to work purely with domain types while supporting multiple transport protocols with optimized performance characteristics.

## Architecture

The transport layer implements an efficient pipeline between network and service layers:

```
Network Packets → Transport Layer → Service Layer (Domain Objects)
                      ↓
               Wire Format Codec
```

## Design Principles

- **Protocol Abstraction**: Service layer is unaware of transport protocol details
- **Wire Format Handling**: Transport layer handles encoding/decoding using domain codecs
- **High Performance**: Optimized for low-latency, high-throughput DNS processing
- **Extensibility**: New transport protocols can be added without changing service logic
- **Interface Compliance**: Implements interfaces defined in the service layer (Dependency Inversion Principle)
- **Memory Efficiency**: Minimal allocations and efficient resource utilization

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
- **Architecture**: Goroutine-per-request model for optimal concurrency
- **Packet Processing**: 512-byte buffer with right-sized packet allocation
- **Performance**: Sub-5μs response latency (~4.5μs typical)
- **Memory Efficiency**: ~930 bytes/operation with 20 allocations/operation
- **Concurrency**: Natural backpressure via Go scheduler
- **Graceful Shutdown**: Context cancellation and stop channel coordination
- **Error Handling**: Comprehensive structured logging for operations and failures

#### Performance Characteristics
```
BenchmarkUDPTransport_QueryProcessing-16     258008    4521 ns/op    930 B/op    20 allocs/op
BenchmarkUDPTransport_ConcurrentConnections-16 262330  4590 ns/op    931 B/op    20 allocs/op
```

#### Optimization Features
- **Minimal Allocations**: Single packet buffer allocation per request
- **Right-sized Buffers**: Allocates exactly the packet size received
- **Clean Memory Model**: No shared state between goroutines
- **Fast Error Paths**: Early returns and efficient error handling

## Future Transport Implementations

The architecture is designed to support additional protocols with similar performance optimizations:

- **DNS over HTTPS (DoH)**: RFC 8484 - HTTP/2 based transport with connection pooling
- **DNS over TLS (DoT)**: RFC 7858 - TLS encrypted DNS with session reuse
- **DNS over QUIC (DoQ)**: RFC 9250 - QUIC based transport with multiplexing

## Usage Example

```go
// Import the service layer interface definitions
import "github.com/haukened/rr-dns/internal/dns/services/resolver"

// Create transport (implements resolver.ServerTransport)
transport := transport.NewUDPTransport(":53", codec, logger)

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

1. **Network Packet Arrives** → UDP socket receives packet into 512-byte buffer
2. **Transport Allocates** → Right-sized packet buffer (exactly packet size)
3. **Goroutine Spawned** → Each packet processed in dedicated goroutine
4. **Wire Decoding** → `codec.DecodeQuery(data)` → `domain.DNSQuery`
5. **Service Processing** → `handler.HandleRequest(query, clientAddr)` → `domain.DNSResponse`
6. **Wire Encoding** → `codec.EncodeResponse(response)` → `[]byte`
7. **Network Transmission** → Response sent back to client

## Performance Characteristics

- **Latency**: Sub-5μs response time (4.5μs typical)
- **Throughput**: ~250,000+ queries/second on modern hardware
- **Memory**: Minimal allocations with efficient garbage collection
- **Scalability**: Natural concurrency via Go scheduler
- **Resource Usage**: Low CPU overhead with optimal memory patterns

## Monitoring & Observability

The transport layer provides structured logging for:
- **Lifecycle Events**: Transport startup/shutdown with configuration details
- **Request Processing**: Query/response details with client information and timing
- **Error Conditions**: Decode failures, network errors, and operational issues
- **Performance Metrics**: Packet sizes, processing paths, and response codes

## Benefits

- **High Performance**: Sub-5μs latency with minimal memory allocations
- **Protocol Independence**: Same service logic works with any transport
- **Production Ready**: Optimized for high-throughput DNS workloads
- **Testing**: Easy to mock transport and service layers independently
- **Future-Proof**: New protocols can be added without breaking existing code
- **Resource Efficient**: Optimal memory usage and CPU utilization
- **Observability**: Comprehensive structured logging for monitoring and debugging
