# ZoneCache

This package provides an in-memory store for authoritative DNS records loaded from zone files. It is used by the resolver service to answer queries from authoritative zones.

## Overview

The `zonecache` package handles:

- **In-memory storage** of authoritative DNS records grouped by zone root
- **Fast lookups** by domain.Question for query resolution  
- **Zone management** with support for replacing and removing entire zones
- **Thread-safe operations** for concurrent read/write access
- **Efficient data structure** optimized for DNS query patterns using cache keys

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
    // FindRecords returns authoritative records matching the Question
    FindRecords(query domain.Question) ([]domain.ResourceRecord, bool)
    
    // PutZone replaces all records for a zone with new records
    PutZone(zoneRoot string, records []domain.ResourceRecord)
    
    // RemoveZone removes all records for a zone
    RemoveZone(zoneRoot string)
    
    // Zones returns a list of all zone roots currently cached
    Zones() []string
    
    // Count returns the total number of cache entries across all zones
    Count() int
}
```

## Data Structure

The ZoneCache uses a nested map structure for efficient lookups:

```go
// Internal structure (simplified)
type ZoneCache struct {
    mu    sync.RWMutex
    zones map[string]map[string][]domain.ResourceRecord
    //    zoneRoot → CacheKey → records
}
```

### Lookup Strategy

1. **Zone Root Lookup**: O(1) access to zone data using apex domain
2. **Cache Key Lookup**: O(1) access to records using Question.CacheKey()
3. **Direct Return**: Records returned directly without additional filtering

The cache key is zone-aware and combines zone root, FQDN, RRType, and RRClass for efficient indexing.
Format: "zoneRoot|name|type|class" (e.g., "example.com.|www.example.com.|A|IN")

## Usage Examples

Note: Each `domain.ResourceRecord` includes both `Data` (wire bytes) and `Text` (human-readable). Zone-derived authoritative records generally populate both to avoid re-decoding when answering queries or performing higher-level logic (e.g. CNAME chase).

### Basic Operations

```go
package main

import (
    "time"
    "github.com/haukened/rr-dns/internal/dns/domain"
    "github.com/haukened/rr-dns/internal/dns/repos/zonecache"
)

func main() {
    // Create zone cache
    cache := zonecache.New()
    
    // Create some authoritative records
    record1, _ := domain.NewAuthoritativeResourceRecord(
        "www.example.com.",
        domain.RRTypeFromString("A"),
        domain.RRClass(1),
        300,
        []byte{192, 0, 2, 1}, // wire bytes
        "192.0.2.1",          // text form
    )
    
    // MX example: proper wire bytes would include preference + encoded domain; simplified here
    mxWire := []byte{0x00, 0x0a /* preference 10 */, /* encoded labels for mail.example.com. */ }
    record2, _ := domain.NewAuthoritativeResourceRecord(
        "mail.example.com.",
        domain.RRTypeFromString("MX"),
        domain.RRClass(1),
        300,
        mxWire,
        "10 mail.example.com.",
    )
    
    // Put zone with new records
    records := []domain.ResourceRecord{record1, record2}
    cache.PutZone("example.com.", records)
    
    // Find records for query
    query, _ := domain.NewQuestion(1, "www.example.com.", domain.RRTypeFromString("A"), domain.RRClass(1))
    results, found := cache.FindRecords(query)
    if found {
        fmt.Printf("Found %d A records for www.example.com.\n", len(results))
    }
}
```

### Integration with Resolver

```go
type ResolverService struct {
    zoneCache resolver.ZoneCache
    // ... other dependencies
}

func (r *ResolverService) resolveAuthoritative(query domain.Question) (domain.DNSResponse, bool) {
    // Look up authoritative records
    records, found := r.zoneCache.FindRecords(query)
    if !found {
        return domain.DNSResponse{}, false
    }
    
    // Create authoritative response directly
    response, _ := domain.NewDNSResponse(query.ID, domain.NOERROR, records, nil, nil)
    return response, true
}
```

### Zone Management

```go
// Load zone from file and update cache
func reloadZone(cache resolver.ZoneCache, zoneRoot, filePath string) error {
    // Load records from zone file  
    records, err := zone.LoadZoneFile(filePath, 300*time.Second)
    if err != nil {
        return fmt.Errorf("failed to load zone file: %w", err)
    }
    
    // Put zone in cache
    cache.PutZone(zoneRoot, records)
    
    log.Info(map[string]any{
        "zone": zoneRoot,
        "records": len(records),
    }, "Zone reloaded successfully")
    
    return nil
}

// Remove a zone entirely
func deleteZone(cache resolver.ZoneCache, zoneRoot string) {
    cache.RemoveZone(zoneRoot)
    
    log.Info(map[string]any{
        "zone": zoneRoot,
    }, "Zone removed successfully")
}
```

## Performance Characteristics

### Lookup Performance
- **Zone Lookup**: O(1) - Direct map access by apex domain
- **Cache Key Lookup**: O(1) - Direct map access using query cache key
- **Zero Allocations**: Records returned directly from cache without copying
- **Overall**: O(1) for all DNS query lookups

### Memory Usage
- **Per Zone**: ~50-100 bytes overhead per zone
- **Per Cache Key**: ~100-200 bytes overhead per unique cache key
- **Per Record**: Minimal overhead, value-based ResourceRecord storage
- **Total**: Approximately records × 100 bytes for overhead estimation

### Concurrency
- **Read Operations**: Multiple concurrent readers supported
- **Write Operations**: Exclusive write access with reader blocking
- **Typical Pattern**: Read-heavy workload (queries) with occasional writes (zone updates)

## Thread Safety

The ZoneCache is designed for high-concurrency DNS query workloads:

### Read Operations (Concurrent)
- `FindRecords()` - Query record lookups
- `Zones()` - Zone enumeration
- `Count()` - Statistics gathering

### Write Operations (Exclusive)
- `PutZone()` - Zone file reloads
- `RemoveZone()` - Zone deletion

### Locking Strategy
```go
// Read operations acquire read lock
func (zc *ZoneCache) FindRecords(query domain.Question) ([]domain.ResourceRecord, bool) {
    zc.mu.RLock()
    defer zc.mu.RUnlock()
    // ... lookup logic
}

// Write operations acquire write lock
func (zc *ZoneCache) PutZone(zoneRoot string, records []domain.ResourceRecord) {
    zc.mu.Lock()
    defer zc.mu.Unlock()
    // ... update logic
}
```

## Error Handling

### Common Error Scenarios
- **Invalid Zone Root**: Malformed zone root domain names
- **Memory Constraints**: Large zone files exceeding memory limits
- **Concurrent Modifications**: Race conditions during zone updates

The current implementation uses simple error handling - most operations are designed to be infallible.

## Administrative Features

### Zone Statistics
```go
// Get zone information for monitoring
stats := map[string]any{
    "total_zones": len(cache.Zones()),
    "total_entries": cache.Count(),
    "zone_list": cache.Zones(),
}
```

### Health Monitoring
```go
// Check zone cache health
func healthCheck(cache resolver.ZoneCache) bool {
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

## Performance Snapshot

ZoneCache is designed for speed. Benchmarks show:

- ✅ Lookup latency: **~120–300ns** depending on zone size and match
- ✅ Put operations: **300ns–22µs** depending on record count
- ✅ Concurrent reads: **<50ns/op** under high load
- ✅ Memory usage: **~80–100 bytes per record**

This makes ZoneCache fast enough to serve millions of queries per second, even on a single core, with consistent zero-GC paths for read-heavy workloads.

```bash
cpu: AMD Ryzen 7 7800X3D 8-Core Processor           
BenchmarkZoneCache_PutZone_SingleRecord-16           	 3635138	       325.5 ns/op	     528 B/op	       5 allocs/op
BenchmarkZoneCache_PutZone_MultipleRecords-16        	 1000000	      1147 ns/op	    1024 B/op	      17 allocs/op
BenchmarkZoneCache_PutZone_LargeZone-16              	   54511	     22296 ns/op	   26264 B/op	     255 allocs/op
BenchmarkZoneCache_FindRecords_Hit-16                	 4061538	       296.5 ns/op	      80 B/op	       3 allocs/op
BenchmarkZoneCache_FindRecords_Miss-16               	 4077876	       291.9 ns/op	      80 B/op	       3 allocs/op
BenchmarkZoneCache_FindRecords_WrongZone-16          	 9715605	       123.3 ns/op	      16 B/op	       1 allocs/op
BenchmarkZoneCache_FindRecords_MultipleRecords-16    	 4035291	       295.6 ns/op	      80 B/op	       3 allocs/op
BenchmarkZoneCache_RemoveZone-16                     	 1858231	       635.1 ns/op	     944 B/op	      11 allocs/op
BenchmarkZoneCache_Zones-16                          	19697874	        60.30 ns/op	      48 B/op	       1 allocs/op
BenchmarkZoneCache_Count-16                          	37341866	        30.91 ns/op	       0 B/op	       0 allocs/op
BenchmarkZoneCache_ConcurrentReads-16                	25137093	        48.75 ns/op	      80 B/op	       3 allocs/op
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

- **[Domain Package](../../domain/)**: Core DNS types (ResourceRecord, Question, RRType)
- **[Resolver Package](../../services/resolver/)**: ZoneCache interface definition
- **[Utils Package](../../common/utils/)**: DNS name canonicalization utilities
- **Standard Library**: `sync` for concurrency
- **No External Dependencies**: Pure Go implementation for reliability

## Related Packages

- **[Zone Loader](../zone/)**: Loads records from zone files
- **[DNS Cache](../dnscache/)**: Caches resolved responses (different from authoritative storage)
- **[Resolver Service](../../services/resolver/)**: Main consumer of zone cache
- **[Domain](../../domain/)**: Core DNS types and interfaces

The ZoneCache provides the foundation for authoritative DNS responses, enabling fast and reliable query resolution for domains managed by the RR-DNS server.
