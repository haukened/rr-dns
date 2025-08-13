# DNS Blocklist Repository

This package currently provides a no-op blocklist implementation for the resolver service. The `NoopBlocklist` satisfies the blocklist interface but always allows queries. The features described below are planned for future development.

## Overview

Planned capabilities include:

- **Domain blocking** based on configurable blocklist sources
- **Fast lookup performance** using efficient data structures
- **Multiple blocklist formats** (hosts files, domain lists, regex patterns)
- **Real-time updates** with hot-reloading capability
- **Pattern matching** for wildcard and subdomain blocking

## Current Implementation

- `NoopBlocklist`: a placeholder that implements the interface and always returns false from `IsBlocked`

## Architecture

The blocklist repository implements the repository pattern:

```
Service Layer → BlocklistRepository Interface → Blocklist Implementation
                        ↓
                File Sources, URL Sources, etc.
```

## Planned Features

### Domain Blocking Capabilities
- **Exact domain matching**: Block specific domains (e.g., `malware.example.com`)
- **Wildcard blocking**: Block domain patterns (e.g., `*.ads.example.com`)
- **Subdomain blocking**: Block all subdomains of a domain
- **Whitelist support**: Allow specific domains despite blocklist matches
- **Category-based blocking**: Block by malware, ads, tracking, etc.

### Blocklist Sources
- **Static files**: Local hosts files and domain lists
- **Remote sources**: Download blocklists from URLs
- **Custom patterns**: User-defined regex and wildcard patterns
- **Popular lists**: Integration with common blocklist providers

### Performance Optimizations
- **Bloom filters**: Fast negative lookups for non-blocked domains
- **Trie structures**: Efficient prefix matching for domain hierarchies
- **Memory caching**: In-memory storage for fast access
- **Lazy loading**: Load blocklist data on demand

## Interface Design

```go
// BlocklistRepository defines the interface for domain blocking
type BlocklistRepository interface {
    // IsBlocked checks if a domain should be blocked
    IsBlocked(domain string) (bool, string, error)
    
    // IsBlockedWithCategory returns blocking status and category
    IsBlockedWithCategory(domain string) (blocked bool, category string, reason string, error)
    
    // Reload refreshes blocklist data from sources
    Reload() error
    
    // Stats returns blocklist statistics
    Stats() BlocklistStats
}

// BlocklistStats provides metrics about the blocklist
type BlocklistStats struct {
    TotalDomains    int
    BlockedQueries  uint64
    AllowedQueries  uint64
    LastUpdate      time.Time
    Sources         []string
}
```

## Usage Examples

### Basic Domain Blocking

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/haukened/rr-dns/internal/dns/repos/blocklist"
)

func main() {
    // Create blocklist repository
    repo, err := blocklist.New(blocklist.Config{
        Sources: []string{
            "/etc/rr-dns/blocklists/malware.txt",
            "/etc/rr-dns/blocklists/ads.txt",
        },
        UpdateInterval: 1 * time.Hour,
    })
    if err != nil {
        log.Fatalf("Failed to create blocklist: %v", err)
    }
    
    // Check if domain is blocked
    blocked, reason, err := repo.IsBlocked("malware.example.com")
    if err != nil {
        log.Fatalf("Blocklist check failed: %v", err)
    }
    
    if blocked {
        fmt.Printf("Domain blocked: %s\n", reason)
        // Return NXDOMAIN or redirect response
    } else {
        fmt.Println("Domain allowed")
        // Continue with normal resolution
    }
}
```

### Integration with DNS Resolution

```go
func resolveDNSQuery(query domain.Question, blocklist blocklist.Repository) domain.DNSResponse {
    // Check blocklist first
    blocked, category, err := blocklist.IsBlockedWithCategory(query.Name)
    if err != nil {
        log.Error(map[string]any{"error": err.Error()}, "Blocklist check failed")
    }
    
    if blocked {
        log.Info(map[string]any{
            "domain":   query.Name,
            "category": category,
            "client":   clientAddr,
        }, "Domain blocked")
        
        // Return NXDOMAIN response
        return domain.NewDNSResponse(query.ID, 3, nil, nil, nil) // NXDOMAIN
    }
    
    // Continue with normal resolution
    return resolveUpstream(query)
}
```

## Configuration

### Blocklist Sources Configuration

```yaml
# DNS server configuration
blocklist:
  enabled: true
  update_interval: "1h"
  sources:
    - type: "file"
      path: "/etc/rr-dns/blocklists/malware.txt"
      category: "malware"
    - type: "url"
      url: "https://someonewhocares.org/hosts/zero/hosts"
      category: "ads"
      update_interval: "24h"
    - type: "pattern"
      patterns: ["*.doubleclick.net", "*.googleadservices.com"]
      category: "tracking"
  
  # Response strategy for blocked domains
  response_type: "nxdomain"  # Options: "nxdomain", "refused", "redirect"
  redirect_ip: "0.0.0.0"     # For redirect response type
  
  # Performance settings
  bloom_filter_size: 1000000
  cache_size: 100000
```

### Environment Variables

```bash
# Blocklist configuration
UDNS_BLOCKLIST_ENABLED=true
UDNS_BLOCKLIST_UPDATE_INTERVAL=1h
UDNS_BLOCKLIST_SOURCES=/etc/rr-dns/blocklists/
UDNS_BLOCKLIST_RESPONSE_TYPE=nxdomain
```

## File Formats

### Hosts File Format
```
# Standard hosts file format
0.0.0.0 malware.example.com
0.0.0.0 ads.badsite.com
127.0.0.1 localhost  # Comments supported
```

### Domain List Format
```
# Simple domain list
malware.example.com
ads.badsite.com
tracking.example.net
# Comments and empty lines ignored
```

### Pattern File Format
```
# Wildcard patterns
*.doubleclick.net
*.googleadservices.com
ads.*
*tracking*
```

## Performance Characteristics

- **Lookup Time**: O(1) for exact matches, O(log n) for pattern matches
- **Memory Usage**: ~50-100 bytes per blocked domain
- **Update Performance**: Incremental updates for large blocklists
- **Cache Hit Ratio**: 95%+ for repeated domain checks

## Architecture Integration

This blocklist repository follows CLEAN architecture principles:

- **Repository Pattern**: Abstracts data source concerns from business logic
- **Interface-Based**: Service layer depends on interfaces, not implementations
- **Dependency Injection**: All external dependencies are injectable
- **Testable Design**: Comprehensive mocking and testing support

## Future Enhancements

### Advanced Features
- **Regex pattern support**: Complex pattern matching beyond wildcards
- **Time-based blocking**: Block domains during specific time periods
- **Geographic blocking**: Block based on client location
- **Dynamic updates**: Real-time blocklist updates via API

### Integration Possibilities
- **Threat intelligence feeds**: Integration with security vendors
- **Machine learning**: Automatic malware domain detection
- **User reporting**: Community-based domain reporting
- **Analytics**: Detailed blocking statistics and reporting

## Dependencies

- **[Domain Package](../../../domain/)**: Core DNS domain types
- **[Bloom Filter](https://github.com/bits-and-blooms/bloom)**: Fast negative lookups
- **[Trie](https://github.com/derekparker/trie)**: Efficient prefix matching

## Testing

The package will include comprehensive tests covering:

- **Basic blocking functionality** with various domain formats
- **Pattern matching** for wildcards and regex patterns
- **Performance testing** with large blocklists
- **Update mechanisms** and hot-reloading
- **Error handling** for malformed blocklist files

Run tests with:
```bash
go test ./internal/dns/repos/blocklist/
```

## Implementation Status

🚧 **This package is currently planned but not yet implemented.**

The blocklist repository is part of the planned infrastructure for the RR-DNS server. The interface design and architecture have been defined to support future implementation without breaking changes to the service layer.

## Related Packages

- **[DNSCache](../dnscache/)**: Caching layer for DNS responses
- **[Zone](../zone/)**: Authoritative zone data repository
- **[Domain](../../domain/)**: Core DNS domain types and interfaces
- **[Config](../../config/)**: Application configuration management

This blocklist repository will integrate seamlessly with existing components to provide comprehensive DNS filtering capabilities while maintaining the clean architecture principles of the RR-DNS server.
