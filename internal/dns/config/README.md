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
    Env      string        `koanf:"env"` // "dev" | "prod"
    Log      LoggingConfig `koanf:"log"`
    Resolver ResolverConfig `koanf:"resolver"`
    Blocklist BlocklistConfig `koanf:"blocklist"`
}

type LoggingConfig struct {
    Level string `koanf:"level"` // debug|info|warn|error
}

type CacheConfig struct {
    Size uint `koanf:"size"` // entries, 0 disables
}

type ResolverConfig struct {
    ZoneDirectory string   `koanf:"zones"`
    Upstream      []string `koanf:"upstream"` // ip:port
    MaxRecursion  int      `koanf:"depth"`
    Port          int      `koanf:"port"`
    Cache         CacheConfig `koanf:"cache"`
}

type BlocklistConfig struct {
    BlocklistDirectory string   `koanf:"dir"`
    URLs               []string `koanf:"urls"`
    Cache              CacheConfig `koanf:"cache"`
    DB                 string   `koanf:"db"`
    Strategy           string   `koanf:"strategy"` // refused|nxdomain|sinkhole
    Sinkhole           *SinkholeOptions `koanf:"sinkhole"` // required if strategy=sinkhole
}

type SinkholeOptions struct {
    Target []string `koanf:"target"` // IPs (A/AAAA)
    TTL    int      `koanf:"ttl"`
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

All configuration is controlled via environment variables with the `DNS_` prefix. Keys use underscores which are transformed to dots internally (e.g., `DNS_RESOLVER_CACHE_SIZE` → `resolver.cache.size`). Values with spaces or commas are split into lists.

Core:

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `DNS_ENV` | string | `prod` | Runtime environment (`dev` or `prod`) |
| `DNS_LOG_LEVEL` | string | `info` | Log verbosity level |

Resolver:

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `DNS_RESOLVER_PORT` | int | 53 (8053 in Docker) | UDP port to bind |
| `DNS_RESOLVER_ZONES` | string | `/etc/rr-dns/zone.d/` (`/zones` in Docker) | Zone directory |
| `DNS_RESOLVER_UPSTREAM` | list | `1.1.1.1:53,1.0.0.1:53` | Upstream DNS servers (ip:port) |
| `DNS_RESOLVER_DEPTH` | int | 8 | Max in-zone alias chase depth |
| `DNS_RESOLVER_CACHE_SIZE` | uint | 1000 | Resolver cache entries (0 disables) |

Blocklist:

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `DNS_BLOCKLIST_DIR` | string | `/etc/rr-dns/blocklist.d/` | Directory for blocklist files |
| `DNS_BLOCKLIST_URLS` | list | — | Remote blocklist URLs (space/comma-separated) |
| `DNS_BLOCKLIST_DB` | string | `/var/lib/rr-dns/blocklist.db` | Blocklist database path |
| `DNS_BLOCKLIST_CACHE_SIZE` | uint | 1000 | Blocklist cache entries |
| `DNS_BLOCKLIST_STRATEGY` | string | `refused` | `refused` | `nxdomain` | `sinkhole` |
| `DNS_BLOCKLIST_SINKHOLE_TARGET` | list | — | Required if strategy=`sinkhole`; IPs (A/AAAA) |
| `DNS_BLOCKLIST_SINKHOLE_TTL` | int | — | Required if strategy=`sinkhole`; TTL seconds |

## Example Configuration

### Development Environment
```bash
export DNS_ENV=dev
export DNS_LOG_LEVEL=debug
export DNS_RESOLVER_PORT=5053
export DNS_RESOLVER_CACHE_SIZE=500
export DNS_RESOLVER_ZONES=./zones/
export DNS_RESOLVER_UPSTREAM="8.8.8.8:53,8.8.4.4:53"
export DNS_RESOLVER_DEPTH=8
# Blocklist (optional)
export DNS_BLOCKLIST_DIR=/etc/rr-dns/blocklist.d/
export DNS_BLOCKLIST_DB=/var/lib/rr-dns/blocklist.db
export DNS_BLOCKLIST_STRATEGY=refused
```

### Production Environment
```bash
export DNS_ENV=prod
export DNS_LOG_LEVEL=info
export DNS_RESOLVER_PORT=53
export DNS_RESOLVER_CACHE_SIZE=10000
export DNS_RESOLVER_ZONES=/etc/rr-dns/zone.d/
export DNS_RESOLVER_UPSTREAM="1.1.1.1:53,1.0.0.1:53,8.8.8.8:53"
export DNS_RESOLVER_DEPTH=8
# Blocklist (optional)
export DNS_BLOCKLIST_DIR=/etc/rr-dns/blocklist.d/
export DNS_BLOCKLIST_DB=/var/lib/rr-dns/blocklist.db
export DNS_BLOCKLIST_STRATEGY=sinkhole
export DNS_BLOCKLIST_SINKHOLE_TARGET="127.0.0.1 ::1"
export DNS_BLOCKLIST_SINKHOLE_TTL=30
```

## Validation

The configuration system includes comprehensive validation:

### Built-in Validations
- **Required fields**: Core and required nested fields must be present, others have defaults
- **Enum validation**: `Env` must be "dev" or "prod"; `Strategy` must be one of `refused|nxdomain|sinkhole`
- **Range validation**: `Port` 1–65535; `Cache.Size` ≥ 0; `Sinkhole.TTL` ≥ 0
- **Conditional validation**: `Sinkhole` is required when `Strategy=sinkhole`
- **Custom validation**: `Resolver.Upstream` entries must be valid `ip:port`

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
