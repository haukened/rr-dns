# DNS Cache

This package provides a high-performance, TTL-aware DNS cache using an LRU (Least Recently Used) eviction strategy with value-based storage for optimal memory management and query performance.

## Overview

The `dnscache` package handles:

- **LRU-based caching** with automatic memory management
- **TTL-aware expiration** that respects DNS record time-to-live values
- **High-performance lookups** with O(1) average case complexity and value-based storage
- **Automatic cleanup** of expired records during access
- **Thread-safe operations** for concurrent DNS query handling
- **Value semantics** for improved CPU cache locality and reduced GC pressure

## Cache Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    DNS Cache                            │
├─────────────────────────────────────────────────────────┤
│  Key: domain.name|domain.name|type|class              │
│  Value: []domain.ResourceRecord (value-based storage)  │
├─────────────────────────────────────────────────────────┤
│  LRU Backing Store                                      │
│  ├─ Most Recently Used ─────────────────────────────┐   │
│  │                                                  │   │
│  │  [records] ← [records] ← [records] ← [records]   │   │
│  │                                                  │   │
│  └─ Least Recently Used ────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

## Performance Benefits

The value-based storage approach provides:
- **Better CPU cache locality**: Records stored as values, not pointers
- **Reduced GC pressure**: Fewer heap allocations and pointer dereferences
- **Sub-microsecond access**: Get operations complete in ~93ns
- **Efficient bulk operations**: Multiple records per cache key supported

## Usage

Records stored in the cache carry both wire-format bytes (`Data`) and a human-readable form (`Text`). At least one must be present; constructors now require a `text string` argument alongside `data []byte`.

### Basic Cache Operations

```go
package main

import (
    "time"
    
    "github.com/haukened/rr-dns/internal/dns/domain"
    "github.com/haukened/rr-dns/internal/dns/repos/dnscache"
)

func main() {
    // Create cache with 1000 entry capacity
    cache, err := dnscache.New(1000)
    if err != nil {
        log.Fatalf("Failed to create cache: %v", err)
    }
    
    // Create a DNS record using the constructor
    record, err := domain.NewCachedResourceRecord(
        "example.com.", 
        domain.RRTypeFromString("A"), // A record
        domain.RRClass(1),            // IN class
        300,                          // TTL: 5 minutes
        []byte{192, 0, 2, 1},         // IP address bytes
        "192.0.2.1",                 // text form
        time.Now(),                   // creation time
    )
    if err != nil {
        log.Fatalf("Failed to create record: %v", err)
    }
    
    // Store in cache (value-based storage)
    err = cache.Set([]domain.ResourceRecord{record})
    if err != nil {
        log.Fatalf("Failed to cache record: %v", err)
    }
    
    // Retrieve from cache
    if cachedRecords, found := cache.Get(record.CacheKey()); found {
        fmt.Printf("Cache hit: %s (%d records)\n", cachedRecords[0].Name, len(cachedRecords))
    } else {
        fmt.Println("Cache miss")
    }
    
    // Check cache size
    fmt.Printf("Cache contains %d unique keys\n", cache.Len())
}
```

### Integration with DNS Resolution

```go
func resolveDNSQuery(query domain.Question, cache resolver.Cache) domain.DNSResponse {
    // Check cache first
    cacheKey := query.CacheKey()
    if cachedRecords, found := cache.Get(cacheKey); found {
        // Cache hit - return cached response with all records
    return domain.DNSResponse{ID: query.ID, RCode: domain.NOERROR, Answers: cachedRecords, Question: query}
    }
    
    // Cache miss - resolve upstream
    // ...resolve via upstream client returning []domain.ResourceRecord...
    responseRecords := []domain.ResourceRecord{/* filled by upstream */}
    
    // Cache the response records (multiple records can share same cache key)
    if len(responseRecords) > 0 {
        err := cache.Set(responseRecords)
        if err != nil {
            // Log error but continue - caching failure shouldn't break resolution
            log.Printf("Failed to cache DNS response: %v", err)
        }
    }
    
    return domain.DNSResponse{ID: query.ID, RCode: domain.NOERROR, Answers: responseRecords, Question: query}
}

### Multiple Records Per Cache Key

```go
// DNS responses often contain multiple records for the same query
var aRecords []domain.ResourceRecord // multiple records for same name/type/class

// All records stored together under same cache key
cache.Set(aRecords)

// Retrieved together as a slice
if records, found := cache.Get("example.com.|www.example.com.|A|IN"); found {
    fmt.Printf("Found %d A records for example.com\n", len(records))
    for _, record := range records {
        fmt.Printf("  IP: %v\n", record.Data)
    }
}
```

## Cache Key Format

The cache uses structured keys based on DNS query parameters:

```
Format: "zoneRoot|name|type|class"
Examples:
├─ "example.com.|example.com.|A|IN"        (A record for apex)
├─ "example.com.|www.example.com.|AAAA|IN"   (AAAA record for www.example.com)
├─ "example.com.|mail.example.com.|MX|IN"  (MX record for mail.example.com)
└─ "example.com.|_sip._tcp.example.com.|SRV|IN" (SRV record)
```

**Note**: Keys are generated automatically using `record.CacheKey()` method for consistency.

## TTL and Expiration Handling

### Automatic Expiration
- Records are checked for expiration on every `Get()` operation
- Expired records are automatically removed from cache
- No background cleanup threads needed (lazy expiration)

### TTL Behavior
```go
// Record with 300-second TTL created using constructor
record, err := domain.NewCachedResourceRecord(
    "example.com.",
    domain.RRTypeFromString("A"),
    domain.RRClass(1),
    300, // TTL in seconds
    []byte{192, 0, 2, 1},
    "192.0.2.1",
    time.Now(),
)

cache.Set([]domain.ResourceRecord{record})

// Immediate access - cache hit
if records, found := cache.Get(record.CacheKey()); found {
    fmt.Printf("Found %d records in cache\n", len(records))
}

// Wait 5 minutes + 1 second
time.Sleep(301 * time.Second)

// Access after expiration - cache miss, records automatically filtered out
if records, found := cache.Get(record.CacheKey()); !found {
    fmt.Println("Records expired and removed")
}
```

## Error Handling

The cache handles various error conditions:

```go
// ErrMultipleKeys: Attempting to cache records with different cache keys together
records := []domain.ResourceRecord{recordA, recordB} // different cache keys
err := cache.Set(records)
if err == dnscache.ErrMultipleKeys {
    // Handle the error - records must have same cache key
}

// Invalid cache size
cache, err := dnscache.New(-1)
if err != nil {
    // Handle invalid cache size
}
```

## Memory Management

### LRU Eviction Strategy
When cache reaches capacity:
1. **New records** are added normally
2. **Least recently used** records are evicted automatically
3. **Access updates** record position (moves to front)
4. **Memory usage** stays within configured bounds

### Cache Sizing Guidelines

| Deployment | Cache Size | Memory Usage | Use Case |
|------------|------------|--------------|----------|
| Development | 100-500 | ~1-5 MB | Local testing |
| Small Office | 1,000-5,000 | ~10-50 MB | <100 clients |
| Corporate | 10,000-50,000 | ~100-500 MB | 100-1000 clients |
| Service Provider | 100,000+ | ~1+ GB | High volume |

### Memory Calculation
```
Approximate memory per record: ~100-200 bytes
Total memory ≈ cache_size × 150 bytes
```

## Performance Characteristics

Based on benchmarks with the value-based implementation:

- **Lookup Time**: ~93ns per Get operation (sub-microsecond)
- **Insertion Time**: ~350ns per Set operation  
- **Memory Overhead**: ~160B per Set, ~64B per Get
- **Thread Safety**: Built-in concurrent access support
- **Cache Hit Ratio**: Typically 80-95% for normal DNS workloads
- **Multiple Records**: ~428ns for retrieving 5 records together

### Benchmark Results
```
BenchmarkDnsCache_Set-16            3394304    350.7 ns/op    160 B/op    5 allocs/op
BenchmarkDnsCache_Get-16           12613561     93.00 ns/op     64 B/op    1 allocs/op
BenchmarkDnsCache_SetMultiple-16    1292930    929.3 ns/op    288 B/op   12 allocs/op
BenchmarkDnsCache_GetMultiple-16    2841843    427.9 ns/op    848 B/op    4 allocs/op
```

## Architecture Integration

This cache follows CLEAN architecture principles:

- **Infrastructure Layer**: Handles caching concerns and memory management
- **Domain Integration**: Stores and retrieves `domain.ResourceRecord` objects
- **Service Independence**: Cache operations don't affect business logic
- **Interface Compliance**: Implements caching patterns expected by service layer

## Configuration Best Practices

### Cache Size Tuning
```go
// Start with conservative size
cache, _ := dnscache.New(1000)

// Monitor hit ratio and memory usage
hitRatio := float64(hits) / float64(total_queries)
memoryUsage := cache.Len() * 150 // approximate bytes

// Increase size if hit ratio < 80% and memory allows
if hitRatio < 0.8 && memoryUsage < maxMemoryBudget {
    // Consider larger cache
}
```

### Integration Patterns
```go
type DNSResolver struct {
    cache    *dnscache.Cache
    upstream UpstreamResolver
    zones    ZoneRepository
}

func (r *DNSResolver) Resolve(query domain.Question) domain.DNSResponse {
    // 1. Check local zones first
    if records := r.zones.Find(query); len(records) > 0 {
        return createResponse(query, records)
    }
    
    // 2. Check cache
    if records, found := r.cache.Get(query.CacheKey()); found {
        return createResponse(query, records)
    }
    
    // 3. Query upstream and cache result
    answers, _ := r.upstream.Resolve(ctx, query, time.Now())
    if len(answers) > 0 {
        if err := r.cache.Set(answers); err != nil {
            // Log but don't fail - caching errors shouldn't break resolution
            log.Printf("Cache error: %v", err)
        }
        return domain.DNSResponse{ID: query.ID, RCode: domain.NOERROR, Answers: answers, Question: query}
    }
    return domain.DNSResponse{ID: query.ID, RCode: domain.SERVFAIL, Answers: nil, Question: query}
}
```

## Testing

The package includes comprehensive tests covering:

- **Basic cache operations** (set, get, delete)
- **TTL expiration behavior** and automatic cleanup
- **LRU eviction** when cache reaches capacity
- **Concurrent access** patterns and thread safety
- **Memory usage** and performance characteristics

Run tests with:
```bash
# Run all tests
go test ./internal/dns/repos/dnscache/

# Run with coverage
go test -cover ./internal/dns/repos/dnscache/

# Run benchmarks
go test -bench=. -benchmem ./internal/dns/repos/dnscache/
```

## Dependencies

- **[HashiCorp LRU](https://github.com/hashicorp/golang-lru)**: High-performance LRU cache implementation
- **Domain Package**: Uses `domain.ResourceRecord` for type safety

## Implementation Notes

- **Lazy Expiration**: Records are only checked for expiration on access
- **Value-Based Storage**: Records stored as values for better performance
- **No Background Threads**: Avoids goroutine overhead and complexity
- **Memory Efficient**: Single LRU structure minimizes memory overhead
- **Thread Safe**: Built-in synchronization for concurrent access
- **Automatic Filtering**: Expired records filtered during Get operations
- **Bulk Operations**: Multiple records with same cache key supported
- **Error Handling**: Validates cache key consistency across record sets

## Monitoring and Metrics

Consider tracking these metrics for cache performance:

```go
type CacheMetrics struct {
    Hits        uint64
    Misses      uint64
    Evictions   uint64
    CurrentSize int
    MaxSize     int
}

func (m *CacheMetrics) HitRatio() float64 {
    total := m.Hits + m.Misses
    if total == 0 {
        return 0
    }
    return float64(m.Hits) / float64(total)
}
```
