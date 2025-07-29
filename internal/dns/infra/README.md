# Infrastructure Layer

This directory contains the infrastructure implementations for the RR-DNS server, following CLEAN architecture principles. Each package handles specific external concerns while maintaining clean separation from business logic.

## Overview

The infrastructure layer provides concrete implementations for:

- **Configuration Management** - Environment-based configuration loading
- **Structured Logging** - High-performance structured logging with Zap
- **Zone File Loading** - Multi-format DNS zone file parsing (YAML/JSON/TOML)
- **DNS Caching** - High-performance LRU cache with TTL awareness
- **Upstream Resolution** - DNS query forwarding to external resolvers

## Architecture Compliance

All infrastructure packages follow CLEAN architecture principles:

```
┌─────────────────────────────────────────────────────────┐
│                 Service Layer                           │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐       │
│  │   Query     │ │   Zone      │ │   Block     │       │
│  │  Resolver   │ │ Management  │ │   List      │       │
│  └─────────────┘ └─────────────┘ └─────────────┘       │
└─────────────────────────────────────────────────────────┘
                        │ │ │
                        ▼ ▼ ▼
┌─────────────────────────────────────────────────────────┐
│              Infrastructure Layer                       │
│ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐   │
│ │ Config   │ │   Log    │ │   Zone   │ │   Cache  │   │
│ │          │ │          │ │ Loader   │ │          │   │
│ └──────────┘ └──────────┘ └──────────┘ └──────────┘   │
│ ┌──────────┐                                          │
│ │Upstream  │    External Dependencies:                │
│ │Resolver  │    • Environment Variables               │
│ └──────────┘    • File System                        │
│                 • Network (UDP)                       │
│                 • Logging Libraries                   │
└─────────────────────────────────────────────────────────┘
```

## Package Responsibilities

### [`config/`](./config/) - Configuration Management
- **Purpose**: Load and validate configuration from environment variables
- **Key Features**: 12-Factor App compliance, validation, type safety
- **Dependencies**: Koanf configuration library, Validator framework
- **Interface**: Provides `AppConfig` struct with all server settings

### [`log/`](./log/) - Structured Logging  
- **Purpose**: Provide structured, leveled logging throughout the application
- **Key Features**: JSON/console output, configurable levels, global logger
- **Dependencies**: Uber Zap logging library
- **Interface**: Global logging functions with structured field support

### [`zone/`](./zone/) - Zone File Loading
- **Purpose**: Load and parse DNS zone files in multiple formats
- **Key Features**: YAML/JSON/TOML support, domain expansion, validation
- **Dependencies**: Koanf parsers for multiple formats
- **Interface**: Returns slice of `domain.AuthoritativeRecord` objects

### [`dnscache/`](./dnscache/) - DNS Caching
- **Purpose**: High-performance caching of DNS records with TTL awareness
- **Key Features**: LRU eviction, automatic expiration, thread safety
- **Dependencies**: HashiCorp LRU cache implementation
- **Interface**: Set/Get operations for `domain.ResourceRecord` objects

### [`upstream/`](./upstream/) - Upstream DNS Resolution
- **Purpose**: Forward DNS queries to external resolvers
- **Key Features**: Multiple servers, failover, DNS wire format, health checks
- **Dependencies**: Standard library networking, custom DNS encoding
- **Interface**: Implements `repo.UpstreamResolver` for service layer

## Dependency Direction

Infrastructure packages follow strict dependency rules:

```
✅ Allowed Dependencies:
├─ Infrastructure → Domain (uses domain types)
├─ Infrastructure → External Libraries (networking, parsing, etc.)
└─ Infrastructure → Standard Library

❌ Prohibited Dependencies:
├─ Infrastructure → Service Layer
├─ Infrastructure → Repository Interfaces
└─ Infrastructure → Other Infrastructure (except domain)
```

## Common Patterns

### Error Handling
All infrastructure packages use consistent error handling:

```go
// Wrap errors with context
return fmt.Errorf("failed to load config: %w", err)

// Validate inputs early
if input == "" {
    return fmt.Errorf("input cannot be empty")
}

// Use domain validation where appropriate
if err := record.Validate(); err != nil {
    return fmt.Errorf("invalid record: %w", err)
}
```

### Configuration
Infrastructure components accept configuration via:

```go
// Constructor injection
resolver := upstream.NewResolver(servers, timeout)
cache, _ := dnscache.New(size)

// Global configuration (logging)
log.Configure(env, level)

// Directory/file paths
records, _ := zone.LoadZoneDirectory(dir, defaultTTL)
```

### Interface Implementation
Infrastructure components implement repository interfaces:

```go
// Upstream resolver implements interface
var _ repo.UpstreamResolver = (*upstream.Resolver)(nil)

// Cache could implement interface (future)
type CacheRepository interface {
    Get(key string) (*domain.ResourceRecord, bool)
    Set(record *domain.ResourceRecord)
}
```

## Testing Strategy

Each infrastructure package includes comprehensive tests:

### Unit Tests
- **Pure logic testing** without external dependencies
- **Error condition coverage** for all failure modes
- **Input validation** and edge case handling
- **Interface compliance** verification

### Integration Tests
- **File system operations** (zone loading)
- **Network operations** (upstream resolution)
- **External library integration** (configuration parsing)
- **Real-world scenarios** with actual data

### Test Organization
```
package_test.go          # Main functionality tests
package_integration_test.go  # External dependency tests
package_benchmark_test.go    # Performance tests
```

## Performance Considerations

### Memory Management
- **DNS Cache**: LRU with configurable size limits
- **Zone Loading**: Streaming parsers where possible
- **Configuration**: Immutable after loading
- **Logging**: Structured fields avoid string concatenation

### Network Efficiency
- **Upstream Resolution**: Connection reuse and timeouts
- **UDP Protocol**: Minimal overhead for DNS queries
- **Concurrent Safety**: Thread-safe operations for parallel queries

### Startup Performance
- **Configuration**: Fast environment variable parsing
- **Zone Loading**: Concurrent file processing where possible
- **Cache Initialization**: Lazy allocation of internal structures

## Monitoring and Observability

Infrastructure packages support observability:

### Logging
```go
log.Info(map[string]any{
    "component": "upstream",
    "servers":   len(servers),
    "timeout":   timeout.String(),
}, "Upstream resolver initialized")
```

### Metrics (Future)
- Cache hit ratios and memory usage
- Upstream resolution latencies and error rates
- Zone loading times and record counts
- Configuration validation results

### Health Checks
```go
// Upstream resolver health
if err := resolver.Health(); err != nil {
    log.Error(map[string]any{"error": err.Error()}, "Upstream resolver unhealthy")
}
```

## Development Guidelines

### Adding New Infrastructure
1. **Create package** in `internal/dns/infra/your-component/`
2. **Define interface** in `internal/dns/repo/` if needed
3. **Implement functionality** using only allowed dependencies
4. **Add comprehensive tests** covering unit and integration scenarios
5. **Create README.md** following the established format
6. **Update this index** with component description

### Code Quality Standards
- **Go formatting**: Use `gofmt` and `golint`
- **Error handling**: Always wrap errors with context
- **Documentation**: Public functions have Go doc comments
- **Testing**: Minimum 80% code coverage
- **Dependencies**: Justify all external dependencies

## Future Enhancements

Planned infrastructure improvements:

- **Metrics Collection**: Prometheus metrics for all components
- **Health Endpoints**: HTTP endpoints for infrastructure health
- **Configuration Reload**: Hot reload of zone files and settings
- **Advanced Caching**: Multi-tier cache with persistence options
- **Connection Pooling**: Enhanced upstream resolver with connection reuse
