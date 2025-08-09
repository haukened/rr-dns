# Wire Package

The `wire` package provides encoding and decoding of DNS messages for UDP transport, implementing the DNS wire format as specified in [RFC 1035](https://tools.ietf.org/html/rfc1035).

## Overview

This package implements the `domain.DNSCodec` interface for standard DNS over UDP messages, handling:

- DNS query encoding/decoding
- DNS response encoding/decoding  
- DNS name compression and decompression
- Binary wire format validation
- Error handling for malformed packets

## Features

- ✅ **RFC 1035 Compliant**: Full implementation of DNS wire format specification
- ✅ **Label Compression**: Handles DNS name compression pointers for efficient packet size
- ✅ **Robust Error Handling**: Comprehensive validation with detailed error messages
- ✅ **100% Test Coverage**: Thoroughly tested including edge cases and error paths
- ✅ **Binary Protocol Support**: Direct byte-level DNS message manipulation
- ✅ **Extensible Architecture**: Designed to support additional codec implementations

## API

### Core Interface

The package exports a single global variable implementing the `DNSCodec` interface:

```go
var UDP domain.DNSCodec = &udpCodec{}
```

### DNSCodec Methods

#### EncodeQuery
```go
EncodeQuery(query domain.Question) ([]byte, error)
```
Serializes a Question into binary format suitable for UDP transmission.

**Parameters:**
- `query`: DNS query with ID, Name, and Type fields

**Returns:**
- `[]byte`: Binary DNS message
- `error`: Error if label too long (>63 chars) or other encoding issues

#### DecodeQuery
```go
DecodeQuery(data []byte) (domain.Question, error)
```
Parses a binary DNS query message into a Question struct.

**Parameters:**
- `data`: Raw DNS query bytes

**Returns:**
- `domain.Question`: Parsed query structure
- `error`: Error if malformed packet, wrong question count, etc.

#### EncodeResponse
```go
EncodeResponse(resp domain.DNSResponse) ([]byte, error)
```
Serializes a DNSResponse into binary format suitable for UDP transmission.

**Parameters:**
- `resp`: DNS response with ID, answers, and other fields

**Returns:**
- `[]byte`: Binary DNS response message
- `error`: Error if domain name encoding fails

#### DecodeResponse
```go
DecodeResponse(data []byte, expectedID uint16, now time.Time) (domain.DNSResponse, error)
```
Parses a binary DNS response message, validating the response ID.

**Parameters:**
- `data`: Raw DNS response bytes
- `expectedID`: Expected query ID for validation

**Returns:**
- `domain.DNSResponse`: Parsed response with resource records
- `error`: Error if malformed, ID mismatch, or invalid resource records

## Usage Examples

### Encoding a DNS Query

```go
import (
    "github.com/haukened/rr-dns/internal/dns/domain"
    "github.com/haukened/rr-dns/internal/dns/gateways/wire"
)

query, _ := domain.NewQuestion(12345, "example.com.", domain.RRTypeA, domain.RRClassIN)

data, err := wire.UDP.EncodeQuery(query)
if err != nil {
    log.Fatal(err)
}
// data now contains binary DNS query ready for UDP transmission
```

### Decoding a DNS Response

```go
// Assume 'responseData' contains binary DNS response from network
expectedID := uint16(12345)

response, err := wire.UDP.DecodeResponse(responseData, expectedID, time.Now())
if err != nil {
    log.Fatal(err)
}

// Access parsed response
fmt.Printf("Response ID: %d\n", response.ID)
for _, answer := range response.Answers {
    fmt.Printf("Answer: %s %s\n", answer.Name, answer.Type)
}
```

### Complete Query/Response Cycle

```go
// Create and encode query
query, _ := domain.NewQuestion(42, "google.com.", domain.RRTypeA, domain.RRClassIN)

queryData, _ := wire.UDP.EncodeQuery(query)

// Send via UDP (pseudocode)
responseData := sendUDPQuery(queryData)

// Decode response
response, err := wire.UDP.DecodeResponse(responseData, 42, time.Now())
if err != nil {
    log.Fatal(err)
}

// Process answers
for _, rr := range response.Answers {
    if rr.Type == 1 { // A record
        ip := net.IP(rr.Data)
        fmt.Printf("%s resolves to %s\n", rr.Name, ip)
    }
}
```

## Internal Functions

### decodeName
```go
func decodeName(data []byte, offset int) (string, int, error)
```
Decodes DNS names with compression pointer support. Handles recursive compression references per RFC 1035.

### encodeDomainName  
```go
func encodeDomainName(name string) ([]byte, error)
```
Encodes domain names into DNS wire format without compression. Validates label lengths (≤63 characters).

## Error Handling

The package provides detailed error messages for various failure scenarios:

### Query/Response Errors
- `"query too short"` - Packet smaller than minimum 12-byte header
- `"response too short"` - Response packet too small
- `"ID mismatch: expected X, got Y"` - Response ID doesn't match query
- `"expected exactly one question"` - Query has wrong question count

### Name Encoding/Decoding Errors
- `"label too long: <label>"` - DNS label exceeds 63 characters
- `"offset out of bounds"` - Read beyond packet boundary
- `"compression pointer out of bounds"` - Invalid compression pointer
- `"label length out of bounds"` - Label extends beyond packet

### Resource Record Errors
- `"truncated answer section"` - Not enough bytes for answer
- `"truncated answer section after name"` - Missing answer fields after name
- `"truncated rdata"` - RDATA shorter than specified length
- `"failed to decode answer name"` - Invalid name in answer section
- `"invalid resource record"` - Resource record construction failed

## DNS Wire Format Details

### Header Format (12 bytes)
```
 0  1  2  3  4  5  6  7  8  9 10 11 12 13 14 15
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                      ID                       |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|QR|   Opcode  |AA|TC|RD|RA|   Z    |   RCODE   |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                    QDCOUNT                    |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                    ANCOUNT                    |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                    NSCOUNT                    |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                    ARCOUNT                    |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
```

### Name Compression
DNS names use label compression where repeated domain suffixes are replaced with 2-byte pointers:
- Pointer format: `11xxxxxx xxxxxxxx` (top 2 bits = 11, remaining 14 bits = offset)
- Compression saves bandwidth for repeated domain names
- Recursive resolution supported with cycle detection

### Question Section
```
 0  1  2  3  4  5  6  7  8  9 10 11 12 13 14 15
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
/                     QNAME                     /
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                     QTYPE                     |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                     QCLASS                    |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
```

## Testing

The package includes comprehensive tests with 100% statement coverage:

```bash
cd internal/dns/gateways/wire
go test -v ./
go test -cover ./
```

Test categories:
- **Encoding Tests**: Valid queries/responses, invalid names, edge cases
- **Decoding Tests**: Valid packets, truncated data, malformed headers
- **Error Path Tests**: All error conditions with precise validation
- **Compression Tests**: Pointer resolution, recursive compression, invalid pointers
- **Edge Cases**: Empty names, maximum label lengths, boundary conditions

## Performance Considerations

- **Zero-copy Decoding**: Uses byte slices without unnecessary allocations
- **Efficient Encoding**: Minimal buffer operations with `bytes.Buffer`
- **Compression Support**: Reduces packet size for repeated domain names
- **Binary Protocol**: Direct byte manipulation for optimal performance

## Extensibility

The wire package is designed around the `domain.DNSCodec` interface, making it extensible for future protocol implementations. While this package currently provides a UDP codec implementation, the architecture supports additional codec types:

### Future Codec Possibilities

- **DNSSEC Codec**: Support for DNS Security Extensions (RFC 4033-4035)
  - RRSIG, DNSKEY, DS, and NSEC record handling
  - Cryptographic signature validation
  - Chain of trust verification

- **DNS over HTTPS (DoH) Codec**: HTTP/2-based DNS transport (RFC 8484)
  - JSON and binary message formats
  - HTTP header handling and status codes
  - TLS-encrypted DNS queries

- **DNS over TLS (DoT) Codec**: TLS-encrypted DNS transport (RFC 7858)
  - TLS handshake and session management
  - Encrypted wire format preservation
  - Certificate validation

- **DNS over QUIC (DoQ) Codec**: QUIC-based DNS transport (RFC 9250)
  - UDP-based encrypted transport
  - Multiplexed stream handling
  - 0-RTT connection establishment

### Implementation Pattern

Future codecs can be implemented by creating new types that satisfy the `domain.DNSCodec` interface:

```go
type secureCodec struct {
    // DNSSEC-specific fields
}

func (c *secureCodec) EncodeQuery(query domain.Question) ([]byte, error) {
    // DNSSEC query encoding logic
}

func (c *secureCodec) DecodeResponse(data []byte, expectedID uint16, now time.Time) (domain.DNSResponse, error) {
    // DNSSEC response decoding with signature validation
}

// Export as global variable
var DNSSEC domain.DNSCodec = &secureCodec{}
```

This modular design allows the DNS resolver to work with different transport mechanisms and security protocols without changing the core domain logic.

## RFC 1035 Compliance

This implementation follows RFC 1035 specifications for:
- DNS message format and header structure
- Label compression algorithm and pointer resolution
- Question and resource record encoding
- Error handling for malformed messages
- Maximum label length (63 characters) and name length (255 characters)

## Dependencies

- `encoding/binary` - Binary data encoding/decoding
- `bytes` - Efficient byte buffer operations  
- `strings` - String manipulation for domain names
- `github.com/haukened/rr-dns/internal/dns/domain` - Domain types and interfaces

## See Also

- [RFC 1035 - Domain Names - Implementation and Specification](https://tools.ietf.org/html/rfc1035)
- [DNS Message Format](https://www.iana.org/assignments/dns-parameters/dns-parameters.xhtml)
- [DNS Compression](https://datatracker.ietf.org/doc/html/rfc1035#section-4.1.4)
