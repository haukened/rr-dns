# Common Infrastructure

This directory contains shared infrastructure components that are used across multiple layers of the DNS server application. These components provide fundamental services that don't fit into specific architectural layers but are essential for the application's operation.

## Overview

The `common` package provides:

- **Cross-cutting concerns** that span multiple architectural layers
- **Shared utilities** used throughout the application
- **Foundation services** that other components depend on
- **Infrastructure abstractions** that abstract external dependencies

## Architecture Role

In the CLEAN architecture, the `common` directory serves as:

- **Shared Infrastructure**: Components used by multiple layers
- **Cross-Layer Services**: Services that don't belong to a specific layer
- **Foundation Layer**: Base services that enable other components
- **Utility Layer**: Reusable utilities and helpers

## Directory Structure

```
common/
└── log/          # Structured logging infrastructure
```

## Components

### [Logging (`log/`)](log/)

The logging package provides structured, leveled logging for the entire DNS server application.

**Key Features:**
- Structured logging with key-value pairs
- Configurable output formats (JSON for production, console for development)
- Multiple log levels (debug, info, warn, error, panic, fatal)
- Global logger instance with injection support for testing
- High-performance Zap-based implementation

**Usage Pattern:**
```go
log.Info(map[string]any{
    "query":    "example.com",
    "type":     "A", 
    "duration": "2ms",
}, "DNS query resolved")
```

## Design Principles

### Cross-Cutting Concerns
Components in `common` address concerns that span multiple architectural layers:
- Logging: Used by all layers for observability
- Configuration: Shared settings across components
- Monitoring: Metrics collection from all layers

### Dependency Direction
Common components follow strict dependency rules:
- ✅ **Common can depend on**: Standard library, external libraries
- ❌ **Common cannot depend on**: Domain, service, or other application layers
- ✅ **Other layers can depend on**: Common components

### Interface-Based Design
All common components provide interfaces to enable:
- **Testing**: Easy mocking and isolation
- **Flexibility**: Multiple implementations
- **Decoupling**: Loose coupling with dependent components

## Integration

### Service Layer Integration
```go
// Services use common components for cross-cutting concerns
type DNSService struct {
    logger log.Logger  // Common logging
    config Config      // Common configuration  
    metrics Metrics    // Common metrics
}
```

### Infrastructure Layer Integration
```go
// Infrastructure components use common services
type UDPTransport struct {
    logger log.Logger  // Common logging for transport events
}
```

## Best Practices

### When to Add Components to Common

**✅ Add to Common when:**
- Component is used by multiple architectural layers
- Service provides cross-cutting functionality
- Component has no business logic dependencies
- Utility is reusable across different contexts

**❌ Don't add to Common when:**
- Component contains business logic
- Service is specific to one domain area
- Component depends on application-specific types
- Utility is only used in one location

### Design Guidelines

1. **Keep It Simple**: Common components should be simple and focused
2. **Avoid Business Logic**: No domain knowledge or business rules
3. **Interface First**: Define interfaces before implementations
4. **Dependency Free**: Minimize dependencies on other application layers
5. **Well Tested**: High test coverage for shared components

## Future Expansion

Planned additions to the common directory:

### Metrics and Monitoring
```go
// Metrics collection for observability
type Metrics interface {
    Counter(name string, tags map[string]string) Counter
    Histogram(name string, tags map[string]string) Histogram
    Gauge(name string, tags map[string]string) Gauge
}
```

### Error Handling
```go
// Standardized error handling and wrapping
type ErrorHandler interface {
    Wrap(err error, message string) error
    WithContext(err error, context map[string]any) error
}
```

### Validation
```go
// Common validation utilities
type Validator interface {
    ValidateStruct(s interface{}) error
    ValidateField(value interface{}, tag string) error
}
```

## Testing

Each common component includes comprehensive tests:

```bash
# Test all common components
go test ./internal/dns/common/...

# Test specific component
go test ./internal/dns/common/log/
```

## Dependencies

Common components use minimal, well-established dependencies:

- **[Zap](https://github.com/uber-go/zap)**: High-performance structured logging
- **Standard Library**: Go's built-in packages for core functionality

## Related Directories

- **[Config](../config/)**: Application configuration management
- **[Domain](../domain/)**: Core business logic and domain types  
- **[Gateways](../gateways/)**: External system integrations
- **[Repos](../repos/)**: Data persistence and retrieval

The `common` directory provides the foundation that enables these other components to focus on their specific responsibilities while sharing essential infrastructure services.
