# rrdata (Resource Record Data Encoding/Decoding)

This package provides focused, type‑aware DNS Resource Record (RR) RDATA encoding and decoding utilities. It converts between human‑readable textual forms (used in zone files, logs, and tests) and canonical wire format byte slices (used in DNS message assembly / parsing).

The package is intentionally small and dependency‑light to keep binary size and allocations minimal in hot paths.

---

## Goals
- Fast, allocation‑lean translation between text and wire forms
- Centralized RRType‑specific logic (avoid scattering per‑type parsing across layers)
- Deterministic, testable behavior with clear error messages
- Only domain model dependency (`domain.RRType`) + internal utils for canonicalization
- Extensible: adding new RR types follows a consistent pattern

---

## Public API

```go
// Encode converts a single textual RDATA string into wire bytes for the given type.
func Encode(rrType domain.RRType, text string) ([]byte, error)

// Decode converts wire bytes into a canonical textual representation for the given type.
func Decode(rrType domain.RRType, wire []byte) (string, error)
```

If a record type is not yet implemented, Encode / Decode return a typed error message:
```
<RTYPE> record encoding not implemented yet
<RTYPE> record decoding not implemented yet
```

For unknown / fallback types (not matched in switch), Encode returns raw bytes of the input string (`[]byte(text)`), and Decode returns `string(wire)` (pass‑through behavior).

---

## Supported Record Types
Implemented end‑to‑end (encode + decode):
- A (IPv4)
- AAAA (IPv6)
- NS
- CNAME
- SOA
- PTR
- MX
- TXT
- SRV
- CAA

Planned / placeholders (return not implemented errors): NAPTR, OPT, DS, RRSIG, NSEC, DNSKEY, TLSA, SVCB, HTTPS.

---

## Textual Formats (Inputs to Encode)
| Type | Text Form | Notes |
|------|-----------|-------|
| A | `192.0.2.1` | Valid IPv4 required |
| AAAA | `2001:db8::1` | Valid IPv6 required |
| NS / CNAME / PTR | `target.example.com.` | Canonicalized (lowercase + trailing dot ensured) |
| MX | `10 mail.example.com.` | Preference (uint16) + FQDN |
| SOA | `mname rname serial refresh retry expire minimum` | 7 space‑separated fields; rname uses dot in place of `@` |
| TXT | Arbitrary UTF‑8 string | Encoded as length + bytes (single segment) |
| SRV | `priority weight port target.` | 4 space‑separated fields |
| CAA | `flags tag value` | Tag preserved, value stored directly |

Decoding produces formats matching the table above (canonical domain normalization applied where appropriate).

---

## Error Handling
Common validation failures:
- Invalid IP address for A / AAAA
- Label length > 63 or malformed domain name
- Wrong field count (SOA must have 7, SRV must have 4, MX must have 2, etc.)
- Integer parsing failures for numeric fields
- Truncated wire data (bounds checks on decode)

Errors are descriptive and prefixed with context (e.g. `invalid SOA mname:`, `invalid MX preference`, `invalid domain name encoding`).

---

## Domain Name Encoding
Domain names are encoded in standard DNS wire format: length‑prefixed labels ending with a zero byte. Name compression is intentionally NOT performed here—that occurs at the wire message layer. Helper functions:
- `encodeDomainName(name string) ([]byte, error)`
- `decodeDomainName(b []byte) (string, error)`

Both enforce canonicalization via `utils.CanonicalDNSName`.

---

## Design Choices
- Keep per‑type logic in small files (e.g. `001a.go`, `006soa.go`) for clarity + focused tests
- Numeric prefixes on filenames roughly map to RR type codes (makes grepping / organization easier)
- Avoid premature generalization—each record type implements only what it needs
- No pointer returns; callers own resulting byte slices
- No streaming: all operations are on fully materialized byte slices (packets are small)

---

## Adding a New RR Type
1. Create a file named `<typecode><mnemonic>.go` (e.g. `029loc.go`).
2. Implement `encode<Type>Data(text string) ([]byte, error)` and `decode<Type>Data(b []byte) (string, error)`.
3. Add switch cases in `Encode` and `Decode`.
4. Write focused tests: `029loc_test.go` covering encode, decode, round trip, and edge cases.
5. If format includes domains, reuse `encodeDomainName` / `decodeDomainName`.
6. Ensure error messages mirror existing style for consistency.

---

## Usage Examples
```go
// Encode textual A record into wire bytes
wire, err := rrdata.Encode(domain.RRTypeA, "192.0.2.1")
if err != nil { /* handle */ }

// Decode wire bytes back to text
text, err := rrdata.Decode(domain.RRTypeA, wire)

// Create a ResourceRecord using both representations
rr, _ := domain.NewAuthoritativeResourceRecord(
    "www.example.com.",
    domain.RRTypeA,
    domain.RRClassIN,
    300,
    wire,
    text,
)
```

```go
// MX example
wire, _ := rrdata.Encode(domain.RRTypeMX, "10 mail.example.com.")
text, _ := rrdata.Decode(domain.RRTypeMX, wire)
rr, _ := domain.NewCachedResourceRecord(
    "example.com.",
    domain.RRTypeMX,
    domain.RRClassIN,
    600,
    wire,
    text,
    time.Now(),
)
```

---

## Testing
Each record type has a dedicated `*_test.go` file validating:
- Encode happy path
- Decode happy path
- Round‑trip (text → wire → text)
- Edge cases (invalid lengths, malformed domains, bad integers)

`decoding_test.go` and `encoding_test.go` contain broader multi‑type coverage.

Run tests:
```bash
go test ./internal/dns/common/rrdata -count=1
```

---

## Limitations / Future Work
- Several RR types return not implemented errors (see list above)
- No support yet for multi‑segment TXT >255 bytes (single segment assumption)
- No DNSSEC (RRSIG, NSEC, DS, DNSKEY) semantics beyond stubs
- Name compression intentionally delegated to higher layer (wire codec)
- Validation is syntactic; semantic policy (e.g., CNAME set constraints) enforced elsewhere

---

## Relationship to Other Packages
- Used by wire codec to decode RDATA when parsing responses
- Used by zone loader (indirectly) to produce canonical wire bytes during authoritative load
- Feeds into `domain.ResourceRecord` dual representation (Data + Text) simplifying resolver logic

---

## License
This component is part of the rr-dns project and inherits the root project license.
