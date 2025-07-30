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
└── zone/          # Zone file loading and management
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

// Example repository interfaces (defined in service layer)
// ZoneRepository interface:
// - FindRecord(name string, recordType uint16) (*domain.ResourceRecord, error)
// - LoadZones(directory string) error
// - GetAuthoritative(domain string) ([]domain.ResourceRecord, error)

// CacheRepository interface:
// - Get(key string) (*domain.ResourceRecord, bool)
// - Set(record *domain.ResourceRecord)
// - Len() int
```

## Design Principles

### Data Source Abstraction
Repositories hide data source implementation details:

```go
// Service layer doesn't know about file formats or cache implementation
type DNSService struct {
    zones     ZoneRepository     // Could be files, database, API
    cache     CacheRepository   // Could be memory, Redis, etc.
    blocklist BlocklistRepository // Could be files, remote lists
}
```

### Interface-Based Design
All repositories are interface-based for flexibility and testing:

```go
// Easy to mock for testing
type mockZoneRepo struct {
    records map[string]*domain.ResourceRecord
}

func (m *mockZoneRepo) FindRecord(name string, recordType uint16) (*domain.ResourceRecord, error) {
    return m.records[name], nil
}
```

### Performance Optimization
Repositories optimize for common DNS access patterns:

- **Cache Layer**: Fast access to recently used records
- **Zone Data**: Pre-loaded authoritative records
- **Blocklist**: Bloom filters for fast negative lookups

## Integration Patterns

### Service Layer Integration
```go
// Import service layer interfaces
import "github.com/haukened/rr-dns/internal/dns/services/resolver"

type DNSResolver struct {
    zones     resolver.ZoneRepository      // Interface defined in service layer
    cache     resolver.CacheRepository     // Interface defined in service layer
    upstream  resolver.UpstreamClient      // Interface defined in service layer
}

func (r *DNSResolver) Resolve(query domain.DNSQuery) domain.DNSResponse {
    // 1. Check authoritative zones
    if record := r.zones.FindRecord(query.Name, query.Type); record != nil {
        return createAuthoritativeResponse(query, record)
    }
    
    // 2. Check cache
    if record, found := r.cache.Get(query.CacheKey()); found {
        return createCachedResponse(query, record)
    }
    
    // 3. Query upstream and cache result
    response := r.upstream.Resolve(query)
    for _, record := range response.Answers {
        r.cache.Set(&record)
    }
    
    return response
}
```

### Configuration Integration
```go
// Repositories are configured through application config
type RepositoryConfig struct {
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
- **Cache Size Limits**: Configurable LRU cache sizes
- **Zone Data**: Loaded once at startup, minimal memory overhead
- **Efficient Data Structures**: Optimized for DNS access patterns

### Access Patterns
- **Cache-First**: Check cache before expensive operations
- **Zone Priority**: Authoritative zones take precedence
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
go test ./internal/dns/repos/dnscache/
go test ./internal/dns/repos/zone/
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
