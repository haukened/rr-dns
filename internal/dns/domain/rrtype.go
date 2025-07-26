package domain

import "fmt"

// RRType represents a DNS resource record type (e.g. A, AAAA, MX).
// See IANA DNS Parameters for assigned codes.
type RRType uint16

// IsValid returns true if the RRType is one of the supported types.
func (t RRType) IsValid() bool {
	switch t {
	case 1, 2, 5, 6, 12, 15, 16, 28, 33, 35, 41, 43, 46, 47, 48, 52, 64, 65, 255, 257:
		return true
	default:
		return false
	}
}

// String returns the textual representation of the RRType.
// For unknown types, it returns "UNKNOWN(<value>)".
func (t RRType) String() string {
	switch t {
	case 1:
		return "A"
	case 2:
		return "NS"
	case 5:
		return "CNAME"
	case 6:
		return "SOA"
	case 12:
		return "PTR"
	case 15:
		return "MX"
	case 16:
		return "TXT"
	case 28:
		return "AAAA"
	case 33:
		return "SRV"
	case 35:
		return "NAPTR"
	case 41:
		return "OPT"
	case 43:
		return "DS"
	case 46:
		return "RRSIG"
	case 47:
		return "NSEC"
	case 48:
		return "DNSKEY"
	case 52:
		return "TLSA"
	case 64:
		return "SVCB"
	case 65:
		return "HTTPS"
	case 255:
		return "ANY"
	case 257:
		return "CAA"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", t)
	}
}

// RRTypeFromString converts a record type string to its corresponding RRType value.
func RRTypeFromString(s string) RRType {
	switch s {
	case "A":
		return 1
	case "NS":
		return 2
	case "CNAME":
		return 5
	case "SOA":
		return 6
	case "PTR":
		return 12
	case "MX":
		return 15
	case "TXT":
		return 16
	case "AAAA":
		return 28
	case "SRV":
		return 33
	case "NAPTR":
		return 35
	case "OPT":
		return 41
	case "DS":
		return 43
	case "RRSIG":
		return 46
	case "NSEC":
		return 47
	case "DNSKEY":
		return 48
	case "TLSA":
		return 52
	case "SVCB":
		return 64
	case "HTTPS":
		return 65
	case "ANY":
		return 255
	case "CAA":
		return 257
	default:
		return 0 // invalid/unknown
	}
}
