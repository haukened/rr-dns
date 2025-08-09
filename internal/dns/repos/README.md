# Repositories

This directory contains repository implementations that handle data persistence, caching, and retrieval operations. Repositories abstract data sources and provide clean interfaces for accessing DNS-related data while maintaining architectural boundaries.

## Overview

The `repos` package provides:

- **Data access abstractions** for various DNS data sources
- **Caching strategies** for improved query performance  
- **Zone data management** from multiple file formats
- **Blocklist functionality** for domain filtering (planned)

## Architecture Role

In the CLEAN architecture, repositories serve as:

- **Data Layer**: Abstract data source concerns from business logic
- **Repository Pattern**: Provide consistent interfaces for data access
- **Persistence Abstraction**: Hide storage implementation details
- **Cache Management**: Optimize data access performance

## Directory Structure

```
repos/
├── blocklist/      # DNS blocklist repository (planned)
├── dnscache/       # High-performance DNS record caching
├── zone/          # Zone file loading and management
└── zonecache/     # In-memory authoritative record storage
```

## Components

### [DNS Cache (`dnscache/`)](dnscache/)

High-performance, TTL-aware DNS cache with LRU eviction.

**Key Features:**
- LRU-based caching with automatic memory management
- TTL-aware expiration respecting DNS record lifetimes
- O(1) lookup performance for cached records
- Thread-safe concurrent access
- Automatic cleanup of expired records

**Performance Characteristics:**
- **Lookup Time**: O(1) average case
- **Memory Overhead**: ~150 bytes per cached record
- **Cache Hit Ratio**: Typically 80-95% for normal workloads
- **Eviction Strategy**: Least Recently Used (LRU)

### [Zone Cache (`zonecache/`)](zonecache/)

In-memory storage for authoritative DNS records with concurrent-safe access.

**Key Features:**
- Fast O(1) lookup performance for authoritative records
- Thread-safe concurrent access using RWMutex
- DNS hierarchy-aware zone matching
- Atomic zone replacement operations
- Memory-efficient nested map structure

**Performance Characteristics:**
- **Lookup Time**: ~2.7μs average case
- **Zone Replace**: ~1.5μs per operation
- **Memory Usage**: Minimal overhead with direct record storage
- **Concurrent Performance**: ~408ns per operation under load
- **Data Structure**: Zone → FQDN → RRType → Records

**Use Cases:**
- Authoritative DNS server record storage
- Zone data caching after file loading
- High-performance DNS query resolution

### [Zone Repository (`zone/`)](zone/)

Multi-format zone file loader supporting YAML, JSON, and TOML formats.

**Key Features:**
- Support for YAML, JSON, and TOML zone files
- Directory scanning for multiple zone files
- Domain name expansion with proper FQDN handling
- Comprehensive error handling and validation
- Authoritative record creation from zone data

**Supported Formats:**
- **YAML**: Human-readable zone configuration
- **JSON**: Machine-readable structured format
- **TOML**: Configuration-style zone definition

### [Blocklist Repository (`blocklist/`) 🚧](blocklist/)

Planned DNS blocklist functionality for domain filtering.

**Planned Features:**
- Domain blocking with multiple pattern types
- Fast lookup performance using bloom filters
- Multiple blocklist source support
- Real-time updates with hot-reloading
- Category-based blocking and whitelisting

**Status**: Interface designed, implementation planned

## Repository Pattern

All repositories implement interfaces defined in the service layer following the Dependency Inversion Principle:

```go
// Interfaces defined in service layer (e.g., internal/dns/services/resolver)
// Repository implementations comply with these contracts:

// ZoneCache interface (defined in services/resolver/interfaces.go):
// - FindRecords(query domain.Question) ([]domain.ResourceRecord, bool)
// - PutZone(zoneRoot string, records []domain.ResourceRecord)
// - RemoveZone(zoneRoot string)
// - Zones() []string
// - Count() int

// Cache interface:
// - Get(key string) ([]domain.ResourceRecord, bool)
// - Set(records []domain.ResourceRecord) error
// - Delete(key string)
// - Len() int
// - Keys() []string
```

## Design Principles

### Data Source Abstraction
Repositories hide data source implementation details:

```go
// Service layer doesn't know about file formats or cache implementation
type DNSService struct {
    zoneCache resolver.ZoneCache     // In-memory authoritative records
    cache     resolver.Cache         // TTL-aware recursive cache
    zones     ZoneRepository         // Zone file management
    blocklist resolver.Blocklist     // Domain filtering
}
```

### Interface-Based Design
All repositories are interface-based for flexibility and testing:

```go
// Easy to mock for testing
type mockZoneCache struct {
    records map[string][]domain.ResourceRecord
}

func (m *mockZoneCache) FindRecords(query domain.Question) ([]domain.ResourceRecord, bool) {
    if records, found := m.records[query.Name]; found {
        var matches []domain.ResourceRecord
        for _, record := range records {
            if record.Type == query.Type {
                matches = append(matches, record)
            }
        }
        return matches, len(matches) > 0
    }
    return nil, false
}
```

### Performance Optimization
Repositories optimize for common DNS access patterns:

- **Zone Cache**: Ultra-fast authoritative record lookups
- **TTL Cache**: Fast access to recently resolved recursive queries
- **Zone Loading**: Pre-loaded authoritative records from files
- **Blocklist**: Bloom filters for fast negative lookups

## Integration Patterns

### Service Layer Integration
```go
// Import service layer interfaces
import "github.com/haukened/rr-dns/internal/dns/services/resolver"

type DNSResolver struct {
    zoneCache resolver.ZoneCache        // Interface defined in service layer
    cache     resolver.Cache            // Interface defined in service layer
    upstream  resolver.UpstreamClient   // Interface defined in service layer
}

func (r *DNSResolver) Resolve(query domain.Question) domain.DNSResponse {
    // 1. Check authoritative zones first
    if records, found := r.zoneCache.FindRecords(query); found {
        return createAuthoritativeResponse(query, records)
    }
    
    // 2. Check recursive cache
    if records, found := r.cache.Get(query.CacheKey()); found {
        return createCachedResponse(query, records)
    }
    
    // 3. Query upstream and cache result
    answers, _ := r.upstream.Resolve(ctx, query, time.Now())
    if len(answers) > 0 {
        _ = r.cache.Set(answers)
        resp, _ := domain.NewDNSResponse(query.ID, domain.NOERROR, answers, nil, nil)
        return resp
    }
    return domain.DNSResponse{}
}
```

### Configuration Integration
```go
// Repositories are configured through application config
type RepositoryConfig struct {
    ZoneCache struct {
        InitialCapacity int `yaml:"initial_capacity"`
    } `yaml:"zone_cache"`
    
    Cache struct {
        Size         int           `yaml:"size"`
        TTLOverride  time.Duration `yaml:"ttl_override"`
    } `yaml:"cache"`
    
    Zones struct {
        Directory    string   `yaml:"directory"`
        DefaultTTL   int      `yaml:"default_ttl"`
        Formats      []string `yaml:"formats"`
    } `yaml:"zones"`
    
    Blocklist struct {
        Enabled      bool     `yaml:"enabled"`
        Sources      []string `yaml:"sources"`
        UpdateInterval time.Duration `yaml:"update_interval"`
    } `yaml:"blocklist"`
}
```

## Error Handling

Repositories provide consistent error handling:

### Zone Loading Errors
- File format parsing errors
- Missing required fields
- Invalid domain names or record types
- File system access errors

### Zone Cache Errors
- DNS hierarchy validation failures
- Invalid zone root format
- Concurrent access conflicts

### Cache Errors
- Memory allocation failures
- Concurrent access issues
- TTL calculation problems

### Blocklist Errors (Planned)
- Blocklist source unavailable
- Pattern compilation errors
- Update synchronization failures

## Performance Considerations

### Memory Management
- **Zone Cache**: Direct in-memory storage with minimal overhead
- **TTL Cache**: Configurable LRU cache sizes with automatic expiration
- **Zone Data**: Loaded once at startup, minimal memory overhead
- **Efficient Data Structures**: Optimized for DNS access patterns

### Access Patterns
- **Authoritative-First**: Check zone cache before recursive resolution
- **Cache-Second**: Check TTL cache before expensive upstream queries
- **Zone Priority**: Authoritative zones take precedence over cached data
- **Lazy Loading**: Load data only when needed

### Concurrent Access
- **Thread-Safe**: All repositories support concurrent access
- **Lock-Free Operations**: Where possible, avoid locking
- **Read Optimization**: Optimize for read-heavy workloads

## Testing Strategy

### Unit Testing
Each repository includes comprehensive unit tests:

```bash
# Test all repositories
go test ./internal/dns/repos/...

# Test specific repository
go test ./internal/dns/repos/zonecache/
go test ./internal/dns/repos/dnscache/
go test ./internal/dns/repos/zone/

# Run benchmarks
go test ./internal/dns/repos/zonecache/ -bench=.
```

### Integration Testing
- File system integration tests
- Large dataset performance tests
- Concurrent access validation

### Mock Testing
- Interface-based mocking for service layer tests
- Predictable test data and behavior
- Error condition simulation

## Configuration Examples

### Zone Cache Configuration
```yaml
zone_cache:
  initial_capacity: 1000  # Initial map capacity for performance
```

### Cache Configuration
```yaml
cache:
  size: 10000           # Maximum cached records
  ttl_override: "300s"  # Override TTL for all records
  cleanup_interval: "1m" # Background cleanup frequency
```

### Zone Configuration
```yaml
zones:
  directory: "/etc/rr-dns/zones/"
  default_ttl: 300
  formats: ["yaml", "json", "toml"]
  watch_changes: true  # Hot-reload on file changes
```

### Blocklist Configuration (Planned)
```yaml
blocklist:
  enabled: true
  sources:
    - "/etc/rr-dns/blocklists/malware.txt"
    - "https://example.com/blocklist.txt"
  update_interval: "1h"
  response_type: "nxdomain"
```

## Dependencies

Repositories use minimal, focused dependencies:

- **[HashiCorp LRU](https://github.com/hashicorp/golang-lru)**: Cache implementation
- **[Koanf](https://github.com/knadh/koanf)**: Configuration parsing for zones
- **Standard Library**: File I/O, time handling, data structures
- **Domain Package**: Core DNS types and interfaces

## Future Enhancements

### Database Integration
```go
// SQL database repository
type SQLZoneRepository struct {
    db *sql.DB
}

// NoSQL cache repository  
type RedisCache struct {
    client redis.Client
}
```

### Remote Data Sources
```go
// API-based zone repository
type APIZoneRepository struct {
    client http.Client
    endpoint string
}

// Remote blocklist repository
type RemoteBlocklistRepository struct {
    sources []BlocklistSource
    client  http.Client
}
```

### Advanced Caching
- **Multi-level caching**: Memory + persistent storage
- **Cache warming**: Pre-populate cache with common queries
- **Intelligent eviction**: Beyond LRU using access patterns

## Related Directories

- **[Domain](../domain/)**: Core business logic and interface definitions
- **[Common](../common/)**: Shared infrastructure services  
- **[Config](../config/)**: Application configuration management
- **[Gateways](../gateways/)**: External system integrations

Repositories provide the data foundation that enables the DNS server to efficiently serve queries while maintaining clean separation between business logic and data access concerns.
