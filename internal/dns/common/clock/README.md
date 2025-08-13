# Clock Abstraction

This package provides a time abstraction layer that enables deterministic testing of time-dependent code throughout the RR-DNS system. It implements the Dependency Inversion Principle by abstracting `time.Now()` calls through an injectable interface.

## Overview

The `clock` package handles:

- **Time abstraction** for production code using real system time
- **Deterministic testing** with controllable mock time
- **TTL simulation** for DNS record expiration testing
- **Time-dependent logic testing** with precise time control

## Design Purpose

DNS systems are inherently time-dependent with features like:
- **TTL (Time To Live)** record expiration
- **Cache lifetime** management  
- **Rate limiting** time windows
- **Performance timing** measurements
- **Log timestamps** for debugging

Traditional testing of time-dependent code is problematic because:
- Tests become flaky due to timing variations
- Fast tests require artificial delays
- TTL expiration testing requires waiting for actual time passage
- Performance benchmarks are affected by system load

## Architecture Role

In the CLEAN architecture, the clock abstraction serves as:

- **Infrastructure Layer**: Abstracts external time concerns
- **Dependency Injection**: Allows test doubles for time operations
- **Interface-Based Design**: Consistent time access throughout the application
- **Testability**: Enables precise control over time in tests

## Core Interface

```go
// Clock provides time access abstraction
type Clock interface {
    Now() time.Time
}
```

## Implementations

### RealClock

Production implementation using actual system time:

```go
type RealClock struct{}

func (c *RealClock) Now() time.Time {
    return time.Now()
}
```

**Characteristics:**
- Zero overhead wrapper around `time.Now()`
- Thread-safe (delegates to standard library)
- Returns actual system time
- Used in production deployments

### MockClock

Test implementation with controllable time:

```go
type MockClock struct {
    CurrentTime time.Time
}

func (c *MockClock) Now() time.Time {
    return c.CurrentTime
}

func (c *MockClock) Advance(d time.Duration) {
    c.CurrentTime = c.CurrentTime.Add(d)
}
```

**Characteristics:**
- Deterministic time for testing
- Controllable time advancement
- Consistent time across multiple calls
- Supports negative duration (time travel backwards)

## Usage Patterns

### Production Code

```go
package dnscache

import (
    "github.com/haukened/rr-dns/internal/dns/common/clock"
    "github.com/haukened/rr-dns/internal/dns/domain"
)

type Cache struct {
    clock clock.Clock
    // ... other fields
}

func NewCache(clk clock.Clock) *Cache {
    return &Cache{
        clock: clk,
        // ... initialize other fields
    }
}

func (c *Cache) Set(record domain.ResourceRecord) {
    now := c.clock.Now()
    expiresAt := now.Add(time.Duration(record.TTL) * time.Second)
    
    // Store record with computed expiration time
    c.store(record, expiresAt)
}

func (c *Cache) Get(key string) ([]domain.ResourceRecord, bool) {
    now := c.clock.Now()
    
    records, found := c.lookup(key)
    if !found {
        return nil, false
    }
    
    // Filter out expired records
    var valid []domain.ResourceRecord
    for _, record := range records {
        if c.isValid(record, now) {
            valid = append(valid, record)
        }
    }
    
    return valid, len(valid) > 0
}
```

### Application Initialization

```go
package main

import (
    "github.com/haukened/rr-dns/internal/dns/common/clock"
    "github.com/haukened/rr-dns/internal/dns/repos/dnscache"
)

func main() {
    // Use real clock in production
    realClock := clock.RealClock{}
    
    // Inject clock into components
    cache := dnscache.NewCache(realClock, 10000)
    resolver := resolver.NewResolver(resolver.Options{
        Cache: cache,
        Clock: realClock,
        // ... other dependencies
    })
    
    // Start server
    server.Start(resolver)
}
```

### Testing with MockClock

```go
package dnscache

import (
    "testing"
    "time"
    
    "github.com/haukened/rr-dns/internal/dns/common/clock"
    "github.com/haukened/rr-dns/internal/dns/domain"
)

func TestCache_TTL_Expiration(t *testing.T) {
    // Create mock clock with fixed time
    startTime := time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)
    mockClock := &clock.MockClock{CurrentTime: startTime}
    
    cache := NewCache(mockClock, 100)
    
    // Create record with 300 second TTL
    record := domain.ResourceRecord{
        Name: "test.example.com.",
        Type: domain.A,
        TTL:  300,
        Data: []byte("192.168.1.1"),
    }
    
    // Store record
    cache.Set(record)
    
    // Verify record is immediately available
    results, found := cache.Get("test.example.com.")
    if !found || len(results) != 1 {
        t.Error("Expected record to be immediately available")
    }
    
    // Advance time by 299 seconds (just before expiry)
    mockClock.Advance(299 * time.Second)
    results, found = cache.Get("test.example.com.")
    if !found || len(results) != 1 {
        t.Error("Expected record to still be valid just before expiry")
    }
    
    // Advance time by 1 more second (at expiry)
    mockClock.Advance(1 * time.Second)
    results, found = cache.Get("test.example.com.")
    if found || len(results) != 0 {
        t.Error("Expected record to be expired")
    }
}

func TestCache_Multiple_Records_Different_TTLs(t *testing.T) {
    startTime := time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)
    mockClock := &clock.MockClock{CurrentTime: startTime}
    
    cache := NewCache(mockClock, 100)
    
    // Records with different TTLs
    shortTTL := domain.ResourceRecord{
        Name: "short.example.com.",
        Type: domain.A,
        TTL:  60,  // 1 minute
        Data: []byte("192.168.1.1"),
    }
    
    longTTL := domain.ResourceRecord{
        Name: "long.example.com.",
        Type: domain.A,
        TTL:  3600, // 1 hour
        Data: []byte("192.168.1.2"),
    }
    
    // Store both records
    cache.Set(shortTTL)
    cache.Set(longTTL)
    
    // Advance time by 90 seconds
    mockClock.Advance(90 * time.Second)
    
    // Short TTL should be expired, long TTL should still be valid
    _, shortFound := cache.Get("short.example.com.")
    _, longFound := cache.Get("long.example.com.")
    
    if shortFound {
        t.Error("Expected short TTL record to be expired")
    }
    if !longFound {
        t.Error("Expected long TTL record to still be valid")
    }
}
```

### Performance Testing

```go
func TestCache_Performance_With_Time_Simulation(t *testing.T) {
    startTime := time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)
    mockClock := &clock.MockClock{CurrentTime: startTime}
    
    cache := NewCache(mockClock, 10000)
    
    // Simulate high-load scenario over time
    for hour := 0; hour < 24; hour++ {
        t.Run(fmt.Sprintf("hour_%d", hour), func(t *testing.T) {
            // Advance to next hour
            if hour > 0 {
                mockClock.Advance(1 * time.Hour)
            }
            
            // Simulate queries for this hour
            for i := 0; i < 1000; i++ {
                key := fmt.Sprintf("query-%d-%d.example.com.", hour, i)
                record := domain.ResourceRecord{
                    Name: key,
                    Type: domain.A,
                    TTL:  300, // 5 minute TTL
                    Data: []byte("192.168.1.1"),
                }
                
                cache.Set(record)
                
                // Immediate lookup should succeed
                if _, found := cache.Get(key); !found {
                    t.Errorf("Failed to retrieve just-stored record: %s", key)
                }
            }
            
            // Verify cache behavior is consistent
            if cache.Len() == 0 {
                t.Error("Cache should contain records")
            }
        })
    }
}
```

## Testing Advantages

### Deterministic Tests
```go
// No waiting for real time
func TestTTLExpiration(t *testing.T) {
    mockClock := &clock.MockClock{CurrentTime: fixedTime}
    
    // Test immediate expiration
    mockClock.Advance(ttlDuration)
    // Record is immediately expired, no waiting
}
```

### Fast Test Execution
```go
// Simulate hours/days of operation in microseconds
func TestLongTermBehavior(t *testing.T) {
    mockClock := &clock.MockClock{CurrentTime: startTime}
    
    // Simulate 30 days of operation
    for day := 0; day < 30; day++ {
        mockClock.Advance(24 * time.Hour) // Instant advancement
        // Test daily operations
    }
}
```

### Precise Edge Case Testing
```go
// Test exact TTL boundaries
func TestTTLBoundary(t *testing.T) {
    mockClock := &clock.MockClock{CurrentTime: startTime}
    
    // Test 1 nanosecond before expiry
    mockClock.Advance(ttl - 1*time.Nanosecond)
    assertRecordValid(t, cache, key)
    
    // Test exactly at expiry
    mockClock.Advance(1 * time.Nanosecond)
    assertRecordExpired(t, cache, key)
}
```

## Integration Examples

### Service Layer Integration

```go
type ResolverService struct {
    clock clock.Clock
    cache Cache
    zones ZoneCache
}

func (r *ResolverService) Resolve(query domain.Question) domain.DNSResponse {
    now := r.clock.Now()
    
    // Check cache with current time
    if cached, found := r.cache.GetValid(query.CacheKey(), now); found {
        return createCachedResponse(cached, now)
    }
    
    // Resolve and cache with timestamp
    response := r.resolveUpstream(query, now)
    r.cache.SetWithTime(response.Answers, now)
    
    return response
}
```

### Configuration and Dependency Injection

```go
// Production configuration
type ProductionConfig struct {
    Clock clock.Clock
}

func NewProductionConfig() *ProductionConfig {
    return &ProductionConfig{
        Clock: clock.RealClock{},
    }
}

// Test configuration
type TestConfig struct {
    Clock *clock.MockClock
}

func NewTestConfig(startTime time.Time) *TestConfig {
    return &TestConfig{
        Clock: &clock.MockClock{CurrentTime: startTime},
    }
}
```

## Performance Characteristics

### RealClock Performance
- **Overhead**: Negligible (~1-2ns per call)
- **Thread Safety**: Inherits from `time.Now()`
- **System Calls**: Delegates to OS time facilities
- **Accuracy**: System clock precision

### MockClock Performance  
- **Overhead**: Near zero (simple field access)
- **Thread Safety**: Safe for concurrent reads, requires synchronization for writes
- **Memory Usage**: Single `time.Time` field (~24 bytes)
- **Determinism**: 100% reproducible results

## Dependencies

- **Standard Library**: `time` package only
- **Zero External Dependencies**: No third-party libraries required
- **Minimal Interface**: Single method contract

## Related Patterns

### Dependency Injection
All time-dependent components accept a `Clock` interface:

```go
func NewDNSCache(clock clock.Clock, size int) *DNSCache
func NewResolver(opts ResolverOptions) *Resolver // opts.Clock
func NewRateLimiter(clock clock.Clock, limit int) *RateLimiter
```

### Test Doubles
The mock clock serves as a test double for time operations:

```go
// Test setup
mockClock := &clock.MockClock{CurrentTime: testStartTime}
component := NewComponent(mockClock)

// Test execution with controlled time
mockClock.Advance(testDuration)
result := component.DoTimeDependentOperation()

// Assertions with predictable timing
assert.Equal(t, expectedResult, result)
```

## Future Enhancements

### Additional Clock Types

```go
// Stepped clock for consistent intervals
type SteppedClock struct {
    start time.Time
    step  time.Duration
    count int
}

// Frozen clock that never advances
type FrozenClock struct {
    frozenTime time.Time
}
```

### Clock Utilities

```go
// Helper functions for common testing patterns
func NewMockClockAt(year, month, day, hour, min, sec int) *MockClock
func AdvanceToNextDay(clock *MockClock)
func AdvanceToNextHour(clock *MockClock)
```

The clock abstraction is fundamental to reliable testing of time-dependent DNS operations, enabling fast, deterministic, and comprehensive test coverage of TTL expiration, cache behavior, and time-sensitive logic throughout the RR-DNS system.
