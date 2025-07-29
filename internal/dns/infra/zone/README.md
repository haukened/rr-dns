# Zone File Loader

This package provides comprehensive DNS zone file loading and parsing capabilities, supporting multiple file formats and converting zone data into domain objects.

## Overview

The `zone` package handles:

- **Multi-format support** for YAML, JSON, and TOML zone files
- **Directory scanning** to load all zone files from a configured directory
- **Domain name expansion** with proper FQDN handling and root zone support
- **Authoritative record creation** from zone file data
- **Error handling** with detailed validation and parsing feedback

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
    
    "github.com/haukened/rr-dns/internal/dns/infra/zone"
)

func main() {
    // Load all zone files from directory
    records, err := zone.LoadZoneDirectory("/etc/rr-dns/zones/", 300*time.Second)
    if err != nil {
        log.Fatalf("Failed to load zones: %v", err)
    }
    
    // Process loaded records
    for _, record := range records {
        fmt.Printf("Loaded: %s %s %s\n", 
            record.Name, 
            record.Type.String(), 
            string(record.Data))
    }
}
```

### Single File Loading

```go
// Load specific zone file
records, err := zone.LoadZoneFile("/etc/rr-dns/zones/example.com.yaml", 300*time.Second)
if err != nil {
    log.Fatalf("Failed to load zone file: %v", err)
}
```

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

This zone loader follows CLEAN architecture principles:

- **Infrastructure Layer**: Handles file I/O and external format parsing
- **Domain Integration**: Converts file data to `domain.AuthoritativeRecord` objects
- **Error Isolation**: File parsing errors don't affect other zones
- **Format Abstraction**: Service layer doesn't need to know about file formats

## Performance Characteristics

- **Startup Loading**: All zones loaded once at application startup
- **Memory Efficient**: Streaming parsers used where possible
- **Fail Fast**: Invalid zone files prevent server startup
- **Concurrent Safe**: Loaded records are immutable after parsing

## Testing

The package includes comprehensive tests covering:

- **Multi-format parsing** (YAML, JSON, TOML)
- **Label expansion** rules and edge cases
- **Error conditions** and validation
- **Multiple record values** and complex zones
- **File system integration** with temporary files

Run tests with:
```bash
go test ./internal/dns/infra/zone/
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
- Keep zone files reasonably sized (< 10MB each)
- Use consistent file formats within deployment
- Minimize complex label expansions
- Group related records in single files
