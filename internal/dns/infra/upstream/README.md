# Upstream DNS Resolver

This package provides the infrastructure implementation for forwarding DNS queries to upstream DNS servers when local resolution fails.

## Overview

The `upstream.Resolver` implements the DNS forwarding functionality required by the RR-DNS architecture. It handles:

- **UDP-based DNS communication** with external DNS servers
- **Query encoding/decoding** following RFC 1035 DNS wire format
- **Multiple upstream servers** with failover capability
- **Configurable timeouts** and context-based cancellation
- **Health checking** for monitoring upstream connectivity

## Usage

```go
package main

import (
    "context"
    "time"
    
    "github.com/haukened/rr-dns/internal/dns/domain"
    "github.com/haukened/rr-dns/internal/dns/infra/upstream"
)

func main() {
    // Create resolver with custom upstream servers
    resolver := upstream.NewResolver(
        []string{"1.1.1.1:53", "1.0.0.1:53"}, // Cloudflare DNS
        5 * time.Second, // 5-second timeout
    )
    
    // Create a DNS query
    query, _ := domain.NewDNSQuery(1234, "example.com.", 1, 1) // A record query
    
    // Resolve with context
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    response, err := resolver.Resolve(ctx, query)
    if err != nil {
        // Handle error
        return
    }
    
    // Process response
    if response.HasAnswers() {
        // Handle successful resolution
    }
}
```

## Architecture Integration

This implementation follows CLEAN architecture principles:

- **Infrastructure Layer**: Handles external DNS communication concerns
- **Interface Segregation**: Implements the `repo.UpstreamResolver` interface
- **Dependency Inversion**: Service layer depends on interface, not implementation
- **Testability**: Fully unit tested with mocked network calls

## Configuration

The resolver accepts:

- **Servers**: List of upstream DNS servers in `host:port` format
- **Timeout**: Default timeout for DNS queries
- **Context**: For cancellation and deadline management

## Error Handling

The resolver handles various error conditions:

- **Network failures**: Connection timeouts, unreachable servers
- **DNS protocol errors**: Malformed responses, ID mismatches
- **Multiple server failures**: Tries all configured servers before failing
- **Context cancellation**: Respects context deadlines and cancellation

## DNS Wire Format

The implementation includes basic DNS wire format encoding/decoding:

- **Query encoding**: Converts `domain.DNSQuery` to RFC 1035 binary format
- **Response decoding**: Parses binary DNS responses to `domain.DNSResponse`
- **Label compression**: Basic support for DNS name compression
- **Standard compliance**: Follows RFC 1035 DNS message format

## Testing

The package includes comprehensive tests:

- **Unit tests**: All encoding/decoding logic
- **Error handling**: Various failure scenarios
- **Integration tests**: Real network connectivity (skipped by default)
- **Interface compliance**: Verifies implementation matches interface

Run tests with:
```bash
go test ./internal/dns/infra/upstream/
```

## Limitations

Current implementation limitations:

- **Basic wire format**: Simplified DNS encoding/decoding
- **UDP only**: No TCP fallback for large responses
- **IPv4 only**: No IPv6 transport support
- **No EDNS0**: No extended DNS features

These limitations are acceptable for the MVP phase and can be extended as needed.
