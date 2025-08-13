# Transport Package

The `transport` package provides high-performance network transport abstractions for DNS server implementations. It follows a **transport-driven architecture** where the network layer owns the application lifecycle and drives all DNS processing, reflecting the fundamental nature of DNS as a network-driven application.

## Architecture Philosophy

### Transport-Driven Design

DNS is inherently a **network-driven application** where everything starts with network packets arriving on specific protocols and ports. The transport layer naturally becomes the entrypoint because:

1. **Network Events Drive Everything**: DNS queries arrive as network packets, not internal events
2. **Protocol Diversity**: Different transports (UDP, DoT, DoH, DoQ) have vastly different network characteristics
3. **Performance Constraints**: Network latency and throughput requirements drive architectural decisions
4. **Resource Management**: Network connections, timeouts, and cancellation are transport concerns

### Why Transport Owns the Lifecycle

```go
// Transport-driven: Network layer owns the application
transport := udp.NewTransport(":53")
resolver := resolver.NewResolver(...)

// Transport starts and drives everything
transport.Start(ctx, resolver) // Transport calls resolver for each query
```

**Benefits:**
- **Natural Flow**: Matches how DNS actually works (packets arrive → processing begins)
- **Protocol Flexibility**: Easy to add DoT, DoH, DoQ without changing DNS logic
- **Resource Efficiency**: Transport handles connection pooling, timeouts, cancellation
- **Clear Boundaries**: Network concerns stay in transport, DNS logic stays in resolver
- **Performance Optimization**: Transport can optimize for protocol-specific characteristics

## Architecture

The transport layer implements an efficient pipeline between network and service layers:

```
Network Packets → Transport Layer → Service Layer (Domain Objects)
     ↓               ↓                    ↓
UDP/DoT/DoH → Wire Format Codec → DNS Business Logic
```

## Design Principles

- **Transport-Driven Lifecycle**: Network layer owns application startup, shutdown, and request processing
- **Protocol Abstraction**: Service layer is unaware of transport protocol details  
- **Network-First Design**: Architecture reflects DNS as a fundamentally network-driven application
- **Wire Format Handling**: Transport layer handles encoding/decoding using domain codecs
- **High Performance**: Optimized for low-latency, high-throughput DNS processing
- **Extensibility**: New transport protocols can be added without changing service logic
- **Interface Compliance**: Implements interfaces defined in the service layer (Dependency Inversion Principle)
- **Memory Efficiency**: Minimal allocations and efficient resource utilization
- **Request Ownership**: Transport controls request lifecycle, timeouts, and cancellation

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
// - HandleQuery(ctx context.Context, query domain.Question, clientAddr net.Addr) (domain.DNSResponse, error)
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

// Create a single resolver instance (DNS business logic)
resolver := resolver.NewResolver(resolver.ResolverOptions{
    ZoneCache:     zoneCache,
    UpstreamCache: upstreamCache,
    Upstream:      upstreamClient,
    Blocklist:     blocklist,
    Clock:         clock.NewRealClock(),
    Logger:        logger,
})

// Create UDP transport and start it
udpTransport := transport.NewUDPTransport(":53", codec, logger)
go udpTransport.Start(ctx, resolver)
defer udpTransport.Stop()

// Additional transports (DoT, DoH, DoQ) can be added in the future
```

### Why This Architecture Works

1. **Single Source of Truth**: One resolver instance ensures consistent DNS behavior across all protocols
2. **Protocol-Specific Optimization**: Each transport optimizes for its protocol (UDP packets, TLS sessions, HTTP/2 streams)
3. **Natural Lifecycle**: Transports start/stop independently, matching network service patterns
4. **Resource Sharing**: All protocols share the same caches, upstream connections, and configuration
5. **Clear Responsibilities**: Transport handles networking, resolver handles DNS logic

## Request Flow

The transport-driven request flow reflects the network-first nature of DNS:

1. **Network Packet Arrives** → UDP socket receives packet into 512-byte buffer
2. **Transport Takes Ownership** → Transport allocates right-sized packet buffer (exactly packet size)
3. **Concurrent Processing** → Each packet processed in dedicated goroutine
4. **Wire Decoding** → `codec.DecodeQuery(data)` → `domain.Question`
5. **Resolver Invocation** → `resolver.HandleQuery(ctx, query, clientAddr)` → `domain.DNSResponse`
6. **Wire Encoding** → `codec.EncodeResponse(response)` → `[]byte`
7. **Network Transmission** → Response sent back to client

### Transport Responsibilities
- **Network I/O**: Socket management, packet reading/writing
- **Concurrency**: Goroutine spawning and lifecycle management
- **Wire Format**: Encoding/decoding between bytes and domain objects
- **Timeouts**: Request timeouts and context cancellation
- **Error Handling**: Network error recovery and client notifications
- **Resource Management**: Connection pooling, buffer management

### Resolver Responsibilities  
- **DNS Logic**: Authoritative lookup, recursive resolution, caching
- **Business Rules**: Query validation, blocklist checking, response construction
- **Data Access**: Zone cache, upstream cache, upstream client coordination
- **Logging**: Structured logging of DNS operations and errors

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
