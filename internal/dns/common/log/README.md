# Structured Logging

This package provides structured, leveled logging for the RR-DNS server using [Uber's Zap](https://github.com/uber-go/zap) logging library.

## Overview

The `log` package handles:

- **Structured logging** with key-value pairs for better observability
- **Configurable output formats** (JSON for production, console for development)
- **Log level control** (debug, info, warn, error, panic, fatal)
- **Global logger instance** for consistent logging across the application
- **Test-friendly design** with logger injection and no-op loggers

## Usage

### Basic Logging

```go
package main

import (
    "github.com/haukened/rr-dns/internal/dns/common/log"
)

func main() {
    // Configure logging (typically done once at startup)
    err := log.Configure("prod", "info")
    if err != nil {
        panic(err)
    }
    
    // Structured logging with fields
    log.Info(map[string]any{
        "query":    "example.com",
        "type":     "A",
        "duration": "2ms",
    }, "DNS query resolved")
    
    // Error logging
    log.Error(map[string]any{
        "error": err.Error(),
        "query": "invalid.domain",
    }, "Failed to resolve DNS query")
    
    // Debug logging (only shown in debug level)
    log.Debug(map[string]any{
        "cache_hit": true,
        "ttl":       300,
    }, "Cache lookup completed")
}
```

### Configuration

```go
// Development environment (human-readable console output)
log.Configure("dev", "debug")

// Production environment (structured JSON output)
log.Configure("prod", "info")
```

## Log Levels

| Level | When to Use | Example |
|-------|-------------|---------|
| `debug` | Development debugging, detailed tracing | Cache hits, parser details, internal state |
| `info` | Normal operations, important events | Query received, zone loaded, server started |
| `warn` | Potentially problematic situations | Retries, fallbacks, deprecated usage |
| `error` | Error conditions that don't stop the program | Query failures, validation errors |
| `panic` | Serious errors that may cause program instability | Corrupt data, unrecoverable state |
| `fatal` | Critical errors that cause program termination | Port binding failure, config errors |

## Output Formats

### Development Mode (`env = "dev"`)
```
2025-07-28T10:30:45.123Z	INFO	DNS query resolved	{"query": "example.com", "type": "A", "duration": "2ms"}
2025-07-28T10:30:45.124Z	ERROR	Failed to resolve query	{"error": "NXDOMAIN", "query": "invalid.domain"}
```

### Production Mode (`env = "prod"`)
```json
{"level":"info","ts":1690534245.123,"msg":"DNS query resolved","query":"example.com","type":"A","duration":"2ms"}
{"level":"error","ts":1690534245.124,"msg":"Failed to resolve query","error":"NXDOMAIN","query":"invalid.domain"}
```

## Architecture Integration

This logging system follows CLEAN architecture principles:

- **Infrastructure Layer**: Handles logging concerns and external dependencies
- **Singleton Pattern**: Global logger accessible throughout the application
- **Interface-Based**: `Logger` interface allows for testing and mocking
- **No Business Logic**: Pure logging functionality without domain knowledge

## Testing Support

### Custom Test Logger

```go
// Create a test logger that captures log output
type testLogger struct {
    entries []string
}

func (l *testLogger) Info(_ map[string]any, msg string) { 
    l.entries = append(l.entries, "INFO:"+msg) 
}
func (l *testLogger) Error(_ map[string]any, msg string) { 
    l.entries = append(l.entries, "ERROR:"+msg) 
}
// ... implement other Logger methods

func TestSomeFunction(t *testing.T) {
    // Use custom test logger to capture log output
    testLogger := &testLogger{}
    log.SetLogger(testLogger)
    defer log.SetLogger(log.GetLogger()) // restore
    
    // Run code that logs
    someFunction()
    
    // Verify logging
    found := false
    for _, entry := range testLogger.entries {
        if strings.Contains(entry, "Expected log message") {
            found = true
            break
        }
    }
    if !found {
        t.Error("Expected log message not found")
    }
}
```

### Silent Testing

```go
func TestSilentOperation(t *testing.T) {
    // Disable all logging for test using built-in noop logger
    original := log.GetLogger()
    log.SetLogger(log.NewNoopLogger())
    defer log.SetLogger(original) // restore

    // Run code without log output
    someFunction()
}
```

## Common Logging Patterns

### DNS Query Logging

```go
log.Info(map[string]any{
    "client_ip": "192.168.1.100",
    "query":     "www.example.com",
    "type":      "A",
    "response":  "NOERROR",
    "duration":  "1.2ms",
}, "DNS query processed")
```

### Error Logging with Context

```go
log.Error(map[string]any{
    "error":     err.Error(),
    "operation": "zone_load",
    "file":      "/etc/rr-dns/zones/example.com.yaml",
    "line":      42,
}, "Failed to parse zone file")
```

### Performance Logging

```go
log.Debug(map[string]any{
    "cache_size":    1500,
    "cache_hits":    892,
    "cache_misses":  108,
    "hit_ratio":     "89.2%",
}, "Cache performance metrics")
```

### Security Logging

```go
log.Warn(map[string]any{
    "client_ip":     "203.0.113.45",
    "query_rate":    "150/min",
    "query_type":    "ANY",
    "blocked":       true,
}, "High query rate detected, blocking client")
```

## Configuration Guidelines

### Development
- Use `"dev"` environment for readable console output
- Set `"debug"` level to see all internal operations
- Include detailed context in log fields

### Production
- Use `"prod"` environment for structured JSON output
- Set `"info"` level to reduce log volume
- Include essential context for troubleshooting
- Consider log aggregation systems (ELK, Fluentd, etc.)

### Monitoring
- Log key performance metrics at `info` level
- Include client IPs and query patterns for security analysis
- Log cache hit ratios and response times for optimization

## Dependencies

- **[Zap](https://github.com/uber-go/zap)**: High-performance structured logging library
- **[Zap Core](https://pkg.go.dev/go.uber.org/zap/zapcore)**: Core logging primitives and interfaces

## Performance Characteristics

- **High Performance**: Zap is optimized for minimal allocation and high throughput
- **Structured Fields**: Key-value pairs are efficiently serialized
- **Level Filtering**: Debug logs are skipped entirely in production
- **Async Logging**: Optional async logging for high-volume scenarios

## Implementation Notes

- **Global State**: Uses global logger for convenience but allows injection
- **Thread Safe**: Concurrent logging is safe across goroutines
- **Memory Efficient**: Structured fields avoid string concatenation
- **JSON Output**: Production logs are JSON-formatted for machine parsing
