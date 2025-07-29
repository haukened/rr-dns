# DNS Cache

This package provides a high-performance, TTL-aware DNS cache using an LRU (Least Recently Used) eviction strategy for optimal memory management and query performance.

## Overview

The `dnscache` package handles:

- **LRU-based caching** with automatic memory management
- **TTL-aware expiration** that respects DNS record time-to-live values
- **High-performance lookups** with O(1) average case complexity
- **Automatic cleanup** of expired records during access
- **Thread-safe operations** for concurrent DNS query handling

## Cache Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    DNS Cache                            │
├─────────────────────────────────────────────────────────┤
│  Key: domain.name:type:class                           │
│  Value: *domain.ResourceRecord (with ExpiresAt)        │
├─────────────────────────────────────────────────────────┤
│  LRU Backing Store                                      │
│  ├─ Most Recently Used ─────────────────────────────┐   │
│  │                                                  │   │
│  │  [record1] ← [record2] ← [record3] ← [record4]   │   │
│  │                                                  │   │
│  └─ Least Recently Used ────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

## Usage

### Basic Cache Operations

```go
package main

import (
    "time"
    
    "github.com/haukened/rr-dns/internal/dns/domain"
    "github.com/haukened/rr-dns/internal/dns/infra/dnscache"
)

func main() {
    // Create cache with 1000 entry capacity
    cache, err := dnscache.New(1000)
    if err != nil {
        log.Fatalf("Failed to create cache: %v", err)
    }
    
    // Create a DNS record
    record, err := domain.NewResourceRecord(
        "example.com.", 
        1,    // A record
        1,    // IN class
        300,  // TTL: 5 minutes
        []byte{192, 0, 2, 1}, // IP address data
    )
    if err != nil {
        log.Fatalf("Failed to create record: %v", err)
    }
    
    // Store in cache
    cache.Set(record)
    
    // Retrieve from cache
    if cachedRecord, found := cache.Get(record.CacheKey()); found {
        fmt.Printf("Cache hit: %s\n", cachedRecord.Name)
    } else {
        fmt.Println("Cache miss")
    }
    
    // Check cache size
    fmt.Printf("Cache contains %d records\n", cache.Len())
}
```

### Integration with DNS Resolution

```go
func resolveDNSQuery(query domain.DNSQuery, cache *dnscache.Cache) domain.DNSResponse {
    // Check cache first
    cacheKey := query.CacheKey()
    if cachedRecord, found := cache.Get(cacheKey); found {
        // Cache hit - return cached response
        return domain.NewDNSResponse(query.ID, 0, []domain.ResourceRecord{*cachedRecord}, nil, nil)
    }
    
    // Cache miss - resolve upstream
    response := resolveUpstream(query)
    
    // Cache the response records
    for _, record := range response.Answers {
        cache.Set(&record)
    }
    
    return response
}
```

## Cache Key Format

The cache uses structured keys based on DNS query parameters:

```
Format: "name:type:class"
Examples:
├─ "example.com.:1:1"        (A record for example.com)
├─ "www.example.com.:28:1"   (AAAA record for www.example.com)
├─ "mail.example.com.:15:1"  (MX record for mail.example.com)
└─ "_sip._tcp.example.com.:33:1" (SRV record)
```

## TTL and Expiration Handling

### Automatic Expiration
- Records are checked for expiration on every `Get()` operation
- Expired records are automatically removed from cache
- No background cleanup threads needed (lazy expiration)

### TTL Behavior
```go
// Record with 300-second TTL
record := &domain.ResourceRecord{
    Name:      "example.com.",
    Type:      1,
    Class:     1,
    ExpiresAt: time.Now().Add(300 * time.Second),
    Data:      []byte{192, 0, 2, 1},
}

cache.Set(record)

// Immediate access - cache hit
if _, found := cache.Get(record.CacheKey()); found {
    fmt.Println("Found in cache")
}

// Wait 5 minutes + 1 second
time.Sleep(301 * time.Second)

// Access after expiration - cache miss, record removed
if _, found := cache.Get(record.CacheKey()); !found {
    fmt.Println("Record expired and removed")
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

- **Lookup Time**: O(1) average case
- **Insertion Time**: O(1) average case  
- **Memory Overhead**: Minimal - single LRU structure
- **Thread Safety**: Built-in concurrent access support
- **Cache Hit Ratio**: Typically 80-95% for normal DNS workloads

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

func (r *DNSResolver) Resolve(query domain.DNSQuery) domain.DNSResponse {
    // 1. Check local zones first
    if record := r.zones.Find(query); record != nil {
        return createResponse(query, record)
    }
    
    // 2. Check cache
    if record, found := r.cache.Get(query.CacheKey()); found {
        return createResponse(query, record)
    }
    
    // 3. Query upstream and cache result
    response := r.upstream.Resolve(query)
    for _, record := range response.Answers {
        r.cache.Set(&record)
    }
    
    return response
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
go test ./internal/dns/infra/dnscache/
```

## Dependencies

- **[HashiCorp LRU](https://github.com/hashicorp/golang-lru)**: High-performance LRU cache implementation
- **Domain Package**: Uses `domain.ResourceRecord` for type safety

## Implementation Notes

- **Lazy Expiration**: Records are only checked for expiration on access
- **No Background Threads**: Avoids goroutine overhead and complexity
- **Memory Efficient**: Single LRU structure minimizes memory overhead
- **Thread Safe**: Built-in synchronization for concurrent access
- **Cache Key Strategy**: Uses domain-provided cache key format for consistency

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
