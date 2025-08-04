# Zone File Loader

This package provides comprehensive DNS zone file loading and parsing capabilities with value-based record storage, supporting multiple file formats and converting zone data into domain objects for optimal performance.

## Overview

The `zone` package handles:

- **Multi-format support** for YAML, JSON, and TOML zone files
- **Directory scanning** to load all zone files from a configured directory
- **Domain name expansion** with proper FQDN handling and root zone support
- **Value-based record creation** from zone file data for improved performance
- **Error handling** with detailed validation and parsing feedback
- **High-performance parsing** optimized for startup-time zone loading

## Performance Characteristics

Based on benchmarks with value-based record storage:

- **YAML Loading**: ~57.5μs per file, 46.5KB/op, 818 allocs/op
- **JSON Loading**: ~37.8μs per file, 28.3KB/op, 566 allocs/op (fastest)
- **TOML Loading**: ~51.8μs per file, 51.5KB/op, 911 allocs/op
- **Directory Loading**: ~159μs for 3 files, 134KB/op, 2319 allocs/op
- **Record Building**: ~62.7ns per single record, 80B/op, 2 allocs/op

### Benchmark Results
```
BenchmarkLoadZoneFile_YAML-16        20794    57520 ns/op   46530 B/op    818 allocs/op
BenchmarkLoadZoneFile_JSON-16        31630    37832 ns/op   28341 B/op    566 allocs/op
BenchmarkLoadZoneFile_TOML-16        23068    51757 ns/op   51506 B/op    911 allocs/op
BenchmarkLoadZoneDirectory-16         6910   159302 ns/op  134441 B/op   2319 allocs/op
BenchmarkBuildResourceRecord_Single  19497298   62.73 ns/op    80 B/op      2 allocs/op
```

**JSON format provides the best performance** for zone file loading.

## Supported File Formats

### YAML Format (`.yaml`, `.yml`)
```yaml
zone_root: example.com.

"@":  # Root/apex record
  A: 192.168.1.1
  AAAA: 2606:4700:4700::1111

www:
  CNAME: example.com.

mail:
  MX:
    - "10 mail.example.com."

api:
  A:
    - "10.0.0.1"
    - "10.0.0.2"

_txt:
  TXT: "This is a test record"
```

### JSON Format (`.json`)
```json
{
  "zone_root": "example.com.",
  "@": {
    "A": "192.168.1.1",
    "AAAA": "2606:4700:4700::1111"
  },
  "www": {
    "CNAME": "example.com."
  },
  "mail": {
    "MX": ["10 mail.example.com."]
  },
  "api": {
    "A": ["10.0.0.1", "10.0.0.2"]
  },
  "_txt": {
    "TXT": "This is a test record"
  }
}
```

### TOML Format (`.toml`)
```toml
zone_root = "example.com."

["@"]
A = "192.168.1.1"
AAAA = "2606:4700:4700::1111"

[www]
CNAME = "example.com."

[mail]
MX = ["10 mail.example.com."]

[api]
A = ["10.0.0.1", "10.0.0.2"]

[_txt]
TXT = "This is a test record"
```

## Usage

### Loading Zone Directory

```go
package main

import (
    "time"
    
    "github.com/haukened/rr-dns/internal/dns/repos/zone"
)

func main() {
    // Load all zone files from directory (returns values, not pointers)
    records, err := zone.LoadZoneDirectory("/etc/rr-dns/zones/", 300*time.Second)
    if err != nil {
        log.Fatalf("Failed to load zones: %v", err)
    }
    
    // Process loaded records - using value semantics
    for _, record := range records {
        fmt.Printf("Loaded: %s %s %s (TTL: %d)\n", 
            record.Name, 
            record.Type.String(), 
            string(record.Data),
            record.TTL())
    }
    
    fmt.Printf("Loaded %d records from zone directory\n", len(records))
}
```

### Single File Loading

```go
// Load specific zone file - returns []domain.ResourceRecord (values)
records, err := zone.LoadZoneFile("/etc/rr-dns/zones/example.com.yaml", 300*time.Second)
if err != nil {
    log.Fatalf("Failed to load zone file: %v", err)
}

// Records are value-based, providing better performance
for _, record := range records {
    fmt.Printf("Name: %s, Type: %s\n", record.Name, record.Type.String())
}
```

### Integration with DNS Cache

```go
// Value-based records integrate seamlessly with cache
cache, _ := dnscache.New(1000)

// Load zone and cache records
records, _ := zone.LoadZoneDirectory("/etc/zones/", 300*time.Second)

// Group records by cache key for efficient caching
recordsByKey := make(map[string][]domain.ResourceRecord)
for _, record := range records {
    key := record.CacheKey()
    recordsByKey[key] = append(recordsByKey[key], record)
}

// Cache grouped records
for _, groupedRecords := range recordsByKey {
    cache.Set(groupedRecords)
}

## Zone File Structure

### Required Fields

Every zone file must contain:
- **`zone_root`**: The root domain for the zone (e.g., `"example.com."`)

### Label Expansion Rules

| Label | Expansion | Example |
|-------|-----------|---------|
| `"@"` | Zone root | `"@"` → `"example.com."` |
| `"www"` | Subdomain | `"www"` → `"www.example.com."` |
| `"mail.sub"` | Multi-level | `"mail.sub"` → `"mail.sub.example.com."` |
| `"absolute."` | Absolute FQDN | `"absolute."` → `"absolute."` (no change) |

### Record Types and Values

| Record Type | Value Format | Example |
|-------------|--------------|---------|
| `A` | IPv4 address | `"192.168.1.1"` |
| `AAAA` | IPv6 address | `"2606:4700:4700::1111"` |
| `CNAME` | Domain name | `"www.example.com."` |
| `MX` | Priority + domain | `"10 mail.example.com."` |
| `NS` | Domain name | `"ns1.example.com."` |
| `TXT` | Text string | `"v=spf1 include:_spf.example.com ~all"` |
| `SRV` | Priority + weight + port + target | `"10 5 443 server.example.com."` |

### Multiple Values

Records can have multiple values using array syntax:

```yaml
www:
  A:
    - "10.0.0.1"
    - "10.0.0.2"
    - "10.0.0.3"

mail:
  MX:
    - "10 mx1.example.com."
    - "20 mx2.example.com."
```

## Error Handling

The zone loader provides detailed error reporting:

### File Format Errors
```
error parsing zone file /etc/zones/example.yaml: yaml: line 5: mapping values are not allowed in this context
```

### Missing Zone Root
```
error parsing zone file /etc/zones/example.yaml: zone_root field is required
```

### Invalid Record Types
```
error parsing zone file /etc/zones/example.yaml: invalid record in example.yaml: unsupported RRType: INVALID
```

### File System Errors
```
error loading zone directory: open /etc/zones/: permission denied
```

## Architecture Integration

This zone loader follows CLEAN architecture principles with value-based optimizations:

- **Infrastructure Layer**: Handles file I/O and external format parsing
- **Domain Integration**: Converts file data to `domain.ResourceRecord` values (not pointers)
- **Error Isolation**: File parsing errors don't affect other zones
- **Format Abstraction**: Service layer doesn't need to know about file formats
- **Value Semantics**: Returns records as values for better CPU cache locality
- **Memory Efficiency**: Reduced heap allocations compared to pointer-based approaches

## Performance Benefits

The value-based approach provides:
- **Better CPU Cache Locality**: Records stored as values improve access patterns
- **Reduced GC Pressure**: Fewer heap allocations during zone loading
- **Efficient Integration**: Direct compatibility with value-based cache systems
- **Fast Startup**: Optimized parsing for quick server initialization

## Testing

The package includes comprehensive tests covering:

- **Multi-format parsing** (YAML, JSON, TOML)
- **Label expansion** rules and edge cases
- **Error conditions** and validation
- **Multiple record values** and complex zones
- **File system integration** with temporary files

Run tests with:
```bash
# Run all tests
go test ./internal/dns/repos/zone/

# Run with coverage
go test -cover ./internal/dns/repos/zone/

# Run benchmarks to measure performance
go test -bench=. -benchmem ./internal/dns/repos/zone/
```

## Dependencies

- **[Koanf](https://github.com/knadh/koanf)**: Configuration parsing library
- **[YAML Parser](https://github.com/knadh/koanf/parsers/yaml)**: YAML format support
- **[JSON Parser](https://github.com/knadh/koanf/parsers/json)**: JSON format support  
- **[TOML Parser](https://github.com/knadh/koanf/parsers/toml)**: TOML format support

## Advanced Features

### Custom TTL Support

```yaml
zone_root: example.com.
default_ttl: 3600  # 1 hour default

www:
  A: 
    value: "192.168.1.1"
    ttl: 300  # 5 minutes override
```

### Special Labels

- **`@`**: Always expands to zone root
- **`_service`**: Underscore labels for SRV/TXT records
- **Absolute domains**: Names ending with `.` are not expanded

### Validation Rules

- Domain names must be valid according to DNS standards
- Record types must be supported by the domain layer
- IP addresses are validated for A/AAAA records
- MX and SRV records validate priority/weight values

## Best Practices

### File Organization
```
/etc/rr-dns/zones/
├── example.com.yaml      # Primary domain
├── internal.yaml         # Internal services
├── reverse.json          # PTR records
└── test.toml            # Test domains
```

### Security Considerations
- Zone files should have restricted permissions (600 or 644)
- Directory should be owned by DNS service user
- Validate zone content before deployment
- Use absolute paths for zone directory configuration

### Performance Tips
- **Use JSON format** for fastest zone loading (~37μs vs ~57μs for YAML)
- Keep zone files reasonably sized (< 10MB each)
- Use consistent file formats within deployment
- Minimize complex label expansions
- Group related records in single files
- **Value-based integration**: Leverage value semantics for cache compatibility
- **Batch loading**: Load zones at startup rather than on-demand for better performance
