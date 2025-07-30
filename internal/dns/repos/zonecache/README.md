# ZoneCache

This package provides an in-memory store for authoritative DNS records loaded from zone files. It is used by the resolver service to answer queries from authoritative zones.

## Overview

The `zonecache` package handles:

- **In-memory storage** of authoritative DNS records grouped by zone root
- **Fast lookups** by FQDN and RRType for query resolution
- **Zone management** with support for replacing and removing entire zones
- **Thread-safe operations** for concurrent read/write access
- **Efficient data structure** optimized for DNS query patterns

## Architecture

The ZoneCache sits between zone file loading and query resolution:

```
Zone Files → Zone Loader → ZoneCache → Resolver Service → DNS Response
```

## Design Principles

- **Performance First**: Optimized data structures for fast DNS lookups
- **Concurrent Safety**: Thread-safe operations using read/write mutex
- **Zone-Based Organization**: Records grouped by zone root for efficient management
- **Memory Efficient**: Minimal overhead while maintaining fast access patterns

## Core Interface

```go
type ZoneCache interface {
    // Find returns authoritative records matching the FQDN and RRType
    Find(fqdn string, rrType domain.RRType) ([]*domain.AuthoritativeRecord, bool)
    
    // ReplaceZone replaces all records for a zone with new records
    ReplaceZone(zoneRoot string, records []*domain.AuthoritativeRecord) error
    
    // RemoveZone removes all records for a zone
    RemoveZone(zoneRoot string) error
    
    // All returns a snapshot of all zone data (for admin/diagnostic purposes)
    All() map[string][]*domain.AuthoritativeRecord
    
    // Zones returns a list of all zone roots currently cached
    Zones() []string
    
    // Count returns the total number of records across all zones
    Count() int
}
```

## Data Structure

The ZoneCache uses a nested map structure for efficient lookups:

```go
// Internal structure (simplified)
type zoneCache struct {
    mu    sync.RWMutex
    zones map[string]map[string][]*domain.AuthoritativeRecord
    //    zoneRoot → fqdn → records
}
```

### Lookup Strategy

1. **Zone Root Lookup**: O(1) access to zone data
2. **FQDN Lookup**: O(1) access to records for a domain
3. **Type Filtering**: Linear scan through records (typically 1-5 records per FQDN)

## Usage Examples

### Basic Operations

```go
package main

import (
    "github.com/haukened/rr-dns/internal/dns/domain"
    "github.com/haukened/rr-dns/internal/dns/repos/zonecache"
)

func main() {
    // Create zone cache
    cache := zonecache.New()
    
    // Create some authoritative records
    record1, _ := domain.NewAuthoritativeRecord(
        "www.example.com.",
        domain.A,
        domain.IN,
        300,
        []byte{192, 0, 2, 1},
    )
    
    record2, _ := domain.NewAuthoritativeRecord(
        "mail.example.com.",
        domain.MX,
        domain.IN,
        300,
        []byte("10 mail.example.com."),
    )
    
    // Replace zone with new records
    records := []*domain.AuthoritativeRecord{record1, record2}
    err := cache.ReplaceZone("example.com.", records)
    if err != nil {
        log.Fatalf("Failed to replace zone: %v", err)
    }
    
    // Find records for query
    results, found := cache.Find("www.example.com.", domain.A)
    if found {
        fmt.Printf("Found %d A records for www.example.com.\n", len(results))
    }
}
```

### Integration with Resolver

```go
type ResolverService struct {
    zoneCache zonecache.ZoneCache
    // ... other dependencies
}

func (r *ResolverService) resolveAuthoritative(query domain.DNSQuery) (domain.DNSResponse, bool) {
    // Look up authoritative records
    records, found := r.zoneCache.Find(query.Name, query.Type)
    if !found {
        return domain.DNSResponse{}, false
    }
    
    // Convert to response records
    var answers []domain.ResourceRecord
    for _, authRecord := range records {
        answers = append(answers, authRecord.ToResourceRecord())
    }
    
    // Create authoritative response
    response := domain.NewDNSResponse(query.ID, domain.NOERROR, answers, nil, nil)
    return response, true
}
```

### Zone Management

```go
// Load zone from file and update cache
func reloadZone(cache zonecache.ZoneCache, zoneRoot, filePath string) error {
    // Load records from zone file
    records, err := zoneloader.LoadZoneFile(filePath, 300*time.Second)
    if err != nil {
        return fmt.Errorf("failed to load zone file: %w", err)
    }
    
    // Replace zone in cache
    err = cache.ReplaceZone(zoneRoot, records)
    if err != nil {
        return fmt.Errorf("failed to update zone cache: %w", err)
    }
    
    log.Info(map[string]any{
        "zone": zoneRoot,
        "records": len(records),
    }, "Zone reloaded successfully")
    
    return nil
}

// Remove a zone entirely
func deleteZone(cache zonecache.ZoneCache, zoneRoot string) error {
    err := cache.RemoveZone(zoneRoot)
    if err != nil {
        return fmt.Errorf("failed to remove zone: %w", err)
    }
    
    log.Info(map[string]any{
        "zone": zoneRoot,
    }, "Zone removed successfully")
    
    return nil
}
```

## Performance Characteristics

### Lookup Performance
- **Zone Lookup**: O(1) - Direct map access
- **FQDN Lookup**: O(1) - Direct map access within zone
- **Type Filtering**: O(n) where n is records per FQDN (typically 1-5)
- **Overall**: O(1) average case for typical DNS queries

### Memory Usage
- **Per Zone**: ~50-100 bytes overhead per zone
- **Per FQDN**: ~100-200 bytes overhead per unique FQDN
- **Per Record**: Minimal overhead, mostly stores pointers to AuthoritativeRecord
- **Total**: Approximately records × 100 bytes for overhead estimation

### Concurrency
- **Read Operations**: Multiple concurrent readers supported
- **Write Operations**: Exclusive write access with reader blocking
- **Typical Pattern**: Read-heavy workload (queries) with occasional writes (zone updates)

## Thread Safety

The ZoneCache is designed for high-concurrency DNS query workloads:

### Read Operations (Concurrent)
- `Find()` - Query record lookups
- `All()` - Administrative snapshots
- `Zones()` - Zone enumeration
- `Count()` - Statistics gathering

### Write Operations (Exclusive)
- `ReplaceZone()` - Zone file reloads
- `RemoveZone()` - Zone deletion

### Locking Strategy
```go
// Read operations acquire read lock
func (zc *zoneCache) Find(fqdn string, rrType domain.RRType) ([]*domain.AuthoritativeRecord, bool) {
    zc.mu.RLock()
    defer zc.mu.RUnlock()
    // ... lookup logic
}

// Write operations acquire write lock
func (zc *zoneCache) ReplaceZone(zoneRoot string, records []*domain.AuthoritativeRecord) error {
    zc.mu.Lock()
    defer zc.mu.Unlock()
    // ... update logic
}
```

## Error Handling

### Common Error Scenarios
- **Invalid Zone Root**: Malformed zone root domain names
- **Duplicate Records**: Multiple records with same name/type/class
- **Memory Constraints**: Large zone files exceeding memory limits
- **Concurrent Modifications**: Race conditions during zone updates

### Error Types
```go
var (
    ErrInvalidZoneRoot = errors.New("invalid zone root format")
    ErrDuplicateRecord = errors.New("duplicate record in zone")
    ErrZoneNotFound    = errors.New("zone not found")
    ErrEmptyZone       = errors.New("zone contains no records")
)
```

## Administrative Features

### Zone Statistics
```go
// Get zone information for monitoring
stats := map[string]any{
    "total_zones": cache.Zones(),
    "total_records": cache.Count(),
    "zones": cache.All(), // Full snapshot for admin UI
}
```

### Health Monitoring
```go
// Check zone cache health
func healthCheck(cache zonecache.ZoneCache) bool {
    zones := cache.Zones()
    if len(zones) == 0 {
        log.Warn(nil, "No zones loaded in cache")
        return false
    }
    
    totalRecords := cache.Count()
    log.Info(map[string]any{
        "zones": len(zones),
        "records": totalRecords,
    }, "Zone cache health check")
    
    return true
}
```

## Testing Strategy

### Unit Tests
- **Concurrent Access**: Multiple goroutines reading/writing simultaneously
- **Zone Operations**: Add, replace, remove zone scenarios
- **Lookup Accuracy**: Verify correct records returned for queries
- **Edge Cases**: Empty zones, non-existent records, invalid inputs

### Performance Tests
- **Lookup Benchmarks**: Measure query response times
- **Memory Usage**: Profile memory consumption with large zones
- **Concurrency**: Test performance under high concurrent load

### Integration Tests
- **Zone Loader Integration**: Test with real zone file data
- **Resolver Integration**: Test with actual DNS query processing

```bash
# Run tests
go test ./internal/dns/repos/zonecache/

# Run benchmarks
go test -bench=. ./internal/dns/repos/zonecache/

# Test with race detection
go test -race ./internal/dns/repos/zonecache/
```

## Configuration

The ZoneCache itself requires no configuration, but integrates with zone loading:

```yaml
# Application configuration (example)
zones:
  directory: "/etc/rr-dns/zones/"
  default_ttl: 300
  cache_size: unlimited  # ZoneCache has no size limit by design
  reload_interval: "5m"  # How often to check for zone file changes
```

## Future Enhancements

### Advanced Features
- **Zone Versioning**: Track zone file versions and rollback capability
- **Incremental Updates**: Support for zone file diffs rather than full replacement
- **Metrics Integration**: Detailed lookup statistics and performance monitoring
- **Persistence**: Optional disk backing for zone data recovery

### Performance Optimizations
- **Bloom Filters**: Fast negative lookups for non-existent records
- **Index Structures**: Additional indexes for complex query patterns
- **Memory Pooling**: Reduce garbage collection pressure
- **Compression**: Compress rarely accessed zone data

## Dependencies

- **[Domain Package](../../domain/)**: Core DNS types (AuthoritativeRecord, RRType)
- **Standard Library**: `sync` for concurrency, `fmt` for errors
- **No External Dependencies**: Pure Go implementation for reliability

## Related Packages

- **[Zone Loader](../zone/)**: Loads records from zone files
- **[DNS Cache](../dnscache/)**: Caches resolved responses (different from authoritative storage)
- **[Resolver Service](../../services/resolver/)**: Main consumer of zone cache
- **[Domain](../../domain/)**: Core DNS types and interfaces

The ZoneCache provides the foundation for authoritative DNS responses, enabling fast and reliable query resolution for domains managed by the RR-DNS server.
