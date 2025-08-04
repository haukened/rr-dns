# DNS Utilities Package

This package provides common utility functions for DNS name processing and normalization used throughout the rr-dns codebase.

## Overview

The utilities handle the complexities of DNS name formatting, canonicalization, and apex domain extraction following RFC 1035 specifications and modern DNS standards.

## Functions

### `CanonicalDNSName(name string) string`

Converts a DNS name to its canonical form for consistent processing and comparison.

**Transformations applied:**
- Converts to lowercase (DNS names are case-insensitive)
- Trims surrounding whitespace
- Ensures trailing dot for fully-qualified domain names (FQDNs)

**Usage:**
```go
canonical := utils.CanonicalDNSName("  WwW.ExAmPlE.CoM  ")
// Returns: "www.example.com."

canonical = utils.CanonicalDNSName("localhost")
// Returns: "localhost."

canonical = utils.CanonicalDNSName("")
// Returns: ""
```

**Properties:**
- **Idempotent**: `CanonicalDNSName(CanonicalDNSName(x)) == CanonicalDNSName(x)`
- **Deterministic**: Always produces the same output for the same input
- **RFC Compliant**: Follows DNS name formatting standards

### `GetApexDomain(name string) string`

Extracts the apex domain (effective TLD + 1) from a fully-qualified domain name using the Public Suffix List.

**Features:**
- Uses `golang.org/x/net/publicsuffix` for accurate TLD recognition
- Handles complex TLDs like `.co.uk`, `.github.io`, `.amazonaws.com`
- Graceful fallback for invalid domains
- Consistent trailing dot formatting

**Usage:**
```go
apex := utils.GetApexDomain("api.service.example.com.")
// Returns: "example.com."

apex = utils.GetApexDomain("www.example.co.uk")
// Returns: "example.co.uk."

apex = utils.GetApexDomain("user.github.io")
// Returns: "user.github.io."

apex = utils.GetApexDomain("localhost")
// Returns: "localhost." (fallback for single labels)
```

**Error Handling:**
- Invalid domains fallback to the original input with proper dot formatting
- IP addresses are handled gracefully (though not recommended for DNS names)
- Empty strings return empty strings

### Utility Functions

#### `removeTrailingDot(name string) string`
Removes a trailing dot from the given domain name string, if present.

#### `addTrailingDot(name string) string`  
Ensures that the provided domain name ends with a trailing dot (useful for FQDNs).

## Use Cases

### Zone Cache Keys
```go
// Normalize domain names for consistent cache lookups
key := utils.CanonicalDNSName(query.Name)
```

### Zone Organization
```go
// Group records by their apex domain
apex := utils.GetApexDomain(record.Name)
zoneRecords[apex] = append(zoneRecords[apex], record)
```

### DNS Query Processing
```go
// Ensure consistent name formatting in DNS queries
fqdn := utils.CanonicalDNSName(query.Name)
zone := utils.GetApexDomain(fqdn)
```

## Implementation Details

### Public Suffix List Integration

The `GetApexDomain` function leverages the Public Suffix List (PSL) to accurately determine where domain registration boundaries exist. This is crucial for:

- **Security**: Preventing cookie/storage leaks across domain boundaries
- **DNS Zone Management**: Organizing records by actual domain ownership
- **Cache Efficiency**: Grouping related subdomains under their apex domain

### Performance Considerations

- Functions are designed for frequent use in DNS query paths
- Minimal memory allocations
- String operations optimized for common DNS name patterns
- No external network calls (PSL data is embedded)

### Edge Cases Handled

1. **Malformed Input**: Graceful fallback without panics
2. **Empty Strings**: Consistent empty string handling
3. **IP Addresses**: Fallback behavior for numeric inputs
4. **International Domains**: Supports ASCII-compatible encoding (ACE)
5. **Single Labels**: Proper handling of non-FQDN inputs like "localhost"

## Testing

The package includes comprehensive test suites covering:

- **Property-based tests**: Idempotency, determinism, consistency
- **Edge cases**: Empty strings, malformed domains, IP addresses
- **RFC compliance**: DNS name formatting standards
- **Public Suffix List**: Complex TLD scenarios

Run tests:
```bash
go test ./internal/dns/common/utils/...
```

## Dependencies

- `strings` (standard library)
- `golang.org/x/net/publicsuffix` - Public Suffix List implementation

## Examples

### Complete DNS Name Processing Pipeline
```go
func processQuery(rawName string) (canonical, apex string) {
    // Step 1: Normalize the input
    canonical = utils.CanonicalDNSName(rawName)
    
    // Step 2: Determine the apex domain for zone lookup
    apex = utils.GetApexDomain(canonical)
    
    return canonical, apex
}

// Example usage:
canonical, apex := processQuery("  API.Service.EXAMPLE.com  ")
// canonical: "api.service.example.com."
// apex: "example.com."
```

### Zone Cache Implementation
```go
func (zc *ZoneCache) FindRecords(query domain.DNSQuery) ([]domain.ResourceRecord, bool) {
    fqdn := utils.CanonicalDNSName(query.Name)
    zone := utils.GetApexDomain(fqdn)
    
    zoneRecords, found := zc.zones[zone]
    if !found {
        return nil, false
    }
    
    // Continue with record lookup...
}
```

This utility package is essential for consistent DNS name handling throughout the rr-dns system, ensuring reliable caching, zone management, and query processing.