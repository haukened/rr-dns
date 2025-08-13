# Configuration Management

This package provides environment-based configuration management for the RR-DNS server, following the [12-Factor App](https://12factor.net/) methodology for configuration.

## Overview

The `config` package handles:

- **Environment variable parsing** with validation and type conversion
- **Default configuration values** for production deployment
- **Configuration validation** using struct tags and custom validators
- **Type-safe configuration** with structured Go types
- **Error reporting** with clear validation messages

## Configuration Structure

```go
type AppConfig struct {
    CacheSize    uint     `koanf:"cache_size"`    // DNS cache size (default: 1000)
    DisableCache bool     `koanf:"disable_cache"` // Disable DNS response caching (default: false)
    Env          string   `koanf:"env"`           // Runtime environment: "dev" or "prod"
    LogLevel     string   `koanf:"log_level"`     // Log level: "debug", "info", "warn", "error"
    Port         int      `koanf:"port"`          // DNS server port (default: 53)
    ZoneDir      string   `koanf:"zone_dir"`      // Zone files directory
    Servers      []string `koanf:"servers"`       // Upstream DNS servers (ip:port format)
    MaxRecursion int      `koanf:"max_recursion"` // Maximum in-zone CNAME recursion depth
}
```

## Usage

```go
package main

import (
    "log"
    
    "github.com/haukened/rr-dns/internal/dns/config"
)

func main() {
    // Load configuration from environment variables
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }
    
    // Use configuration
    fmt.Printf("DNS server will listen on port %d\n", cfg.Port)
    fmt.Printf("Cache size: %d entries\n", cfg.CacheSize)
    fmt.Printf("Zone directory: %s\n", cfg.ZoneDir)
    fmt.Printf("Upstream servers: %v\n", cfg.Servers)
}
```

## Environment Variables

All configuration is controlled via environment variables with the `DNS_` prefix:

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `DNS_CACHE_SIZE` | uint | 1000 | Maximum number of DNS records to cache |
| `DNS_DISABLE_CACHE` | bool | false | Disable DNS response caching for testing |
| `DNS_ENV` | string | "prod" | Runtime environment (`dev` or `prod`) |
| `DNS_LOG_LEVEL` | string | "info" | Log verbosity level |
| `DNS_PORT` | int | 53 | UDP port for DNS server to bind to |
| `DNS_ZONE_DIR` | string | "/etc/rr-dns/zones/" | Directory containing zone files |
| `DNS_SERVERS` | string | "1.1.1.1:53,1.0.0.1:53" | Comma-separated upstream DNS servers |
| `DNS_MAX_RECURSION` | int | 8 | Maximum in-zone CNAME recursion depth |

## Example Configuration

### Development Environment
```bash
export DNS_ENV=dev
export DNS_LOG_LEVEL=debug
export DNS_PORT=5053
export DNS_CACHE_SIZE=500
export DNS_ZONE_DIR=./zones/
export DNS_SERVERS=8.8.8.8:53,8.8.4.4:53
export DNS_MAX_RECURSION=8
```

### Production Environment
```bash
export DNS_ENV=prod
export DNS_LOG_LEVEL=info
export DNS_PORT=53
export DNS_CACHE_SIZE=10000
export DNS_ZONE_DIR=/etc/rr-dns/zones/
export DNS_SERVERS=1.1.1.1:53,1.0.0.1:53,8.8.8.8:53
export DNS_MAX_RECURSION=8
```

## Validation

The configuration system includes comprehensive validation:

### Built-in Validations
- **Required fields**: All configuration values must be provided or have defaults
- **Enum validation**: `Env` must be "dev" or "prod"
- **Range validation**: `Port` must be 1-65534, `CacheSize` must be ≥1
- **Custom validation**: `Servers` must be valid IP:port combinations

### Custom Validators
- **IP:Port format**: Validates upstream server addresses are properly formatted
- **File system paths**: Ensures zone directory paths are valid
- **Network ports**: Validates port numbers are in valid range

## Error Handling

The configuration loader provides detailed error messages for common issues:

```go
cfg, err := config.Load()
if err != nil {
    // Error examples:
    // "validation failed: Field 'Port' must be between 1 and 65534"
    // "validation failed: Field 'Servers[0]' must be a valid IP:port"
    // "validation failed: Field 'ZoneDir' is required"
    log.Fatalf("Configuration error: %v", err)
}
```

## Architecture Integration

This configuration system follows CLEAN architecture principles:

- **Infrastructure Layer**: Handles environment variable concerns
- **No Business Logic**: Pure configuration parsing and validation
- **Interface Compliance**: Provides structured data for service layer
- **Testability**: Fully unit tested with various scenarios

## Testing

The package includes comprehensive tests covering:

- **Default configuration loading**
- **Environment variable override**
- **Validation error cases**
- **Invalid format handling**
- **Custom validator testing**

Run tests with:
```bash
go test ./internal/dns/config/
```

## Dependencies

- **[koanf](https://github.com/knadh/koanf)**: Configuration management library
- **[validator](https://github.com/go-playground/validator)**: Struct validation framework

## Implementation Notes

- **12-Factor Compliance**: All configuration via environment variables
- **Type Safety**: Strong typing with validation at startup
- **Fail Fast**: Configuration errors prevent server startup
- **No Runtime Changes**: Configuration is immutable after loading
- **Production Ready**: Includes sensible defaults for production deployment
