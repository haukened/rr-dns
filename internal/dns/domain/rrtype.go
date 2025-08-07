package domain

import "fmt"

// RRType represents a DNS resource record type (e.g. A, AAAA, MX).
// See IANA DNS Parameters for assigned codes.
type RRType uint16

// DNS Resource Record Type constants
const (
	RRTypeA      RRType = 1   // A - IPv4 address
	RRTypeNS     RRType = 2   // NS - Name server
	RRTypeCNAME  RRType = 5   // CNAME - Canonical name
	RRTypeSOA    RRType = 6   // SOA - Start of authority
	RRTypePTR    RRType = 12  // PTR - Pointer
	RRTypeMX     RRType = 15  // MX - Mail exchange
	RRTypeTXT    RRType = 16  // TXT - Text
	RRTypeAAAA   RRType = 28  // AAAA - IPv6 address
	RRTypeSRV    RRType = 33  // SRV - Service
	RRTypeNAPTR  RRType = 35  // NAPTR - Naming authority pointer
	RRTypeOPT    RRType = 41  // OPT - EDNS option
	RRTypeDS     RRType = 43  // DS - Delegation signer
	RRTypeRRSIG  RRType = 46  // RRSIG - Resource record signature
	RRTypeNSEC   RRType = 47  // NSEC - Next secure
	RRTypeDNSKEY RRType = 48  // DNSKEY - DNS key
	RRTypeTLSA   RRType = 52  // TLSA - TLS association
	RRTypeSVCB   RRType = 64  // SVCB - Service binding
	RRTypeHTTPS  RRType = 65  // HTTPS - HTTPS binding
	RRTypeANY    RRType = 255 // ANY - Any type (query only)
	RRTypeCAA    RRType = 257 // CAA - Certificate authority authorization
)

// IsValid returns true if the RRType is one of the supported types.
func (t RRType) IsValid() bool {
	switch t {
	case RRTypeA, RRTypeNS, RRTypeCNAME, RRTypeSOA, RRTypePTR, RRTypeMX, RRTypeTXT,
		RRTypeAAAA, RRTypeSRV, RRTypeNAPTR, RRTypeOPT, RRTypeDS, RRTypeRRSIG,
		RRTypeNSEC, RRTypeDNSKEY, RRTypeTLSA, RRTypeSVCB, RRTypeHTTPS, RRTypeANY, RRTypeCAA:
		return true
	default:
		return false
	}
}

// String returns the textual representation of the RRType.
// For unknown types, it returns "UNKNOWN(<value>)".
func (t RRType) String() string {
	switch t {
	case RRTypeA:
		return "A"
	case RRTypeNS:
		return "NS"
	case RRTypeCNAME:
		return "CNAME"
	case RRTypeSOA:
		return "SOA"
	case RRTypePTR:
		return "PTR"
	case RRTypeMX:
		return "MX"
	case RRTypeTXT:
		return "TXT"
	case RRTypeAAAA:
		return "AAAA"
	case RRTypeSRV:
		return "SRV"
	case RRTypeNAPTR:
		return "NAPTR"
	case RRTypeOPT:
		return "OPT"
	case RRTypeDS:
		return "DS"
	case RRTypeRRSIG:
		return "RRSIG"
	case RRTypeNSEC:
		return "NSEC"
	case RRTypeDNSKEY:
		return "DNSKEY"
	case RRTypeTLSA:
		return "TLSA"
	case RRTypeSVCB:
		return "SVCB"
	case RRTypeHTTPS:
		return "HTTPS"
	case RRTypeANY:
		return "ANY"
	case RRTypeCAA:
		return "CAA"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", t)
	}
}

// RRTypeFromString converts a record type string to its corresponding RRType value.
func RRTypeFromString(s string) RRType {
	switch s {
	case "A":
		return RRTypeA
	case "NS":
		return RRTypeNS
	case "CNAME":
		return RRTypeCNAME
	case "SOA":
		return RRTypeSOA
	case "PTR":
		return RRTypePTR
	case "MX":
		return RRTypeMX
	case "TXT":
		return RRTypeTXT
	case "AAAA":
		return RRTypeAAAA
	case "SRV":
		return RRTypeSRV
	case "NAPTR":
		return RRTypeNAPTR
	case "OPT":
		return RRTypeOPT
	case "DS":
		return RRTypeDS
	case "RRSIG":
		return RRTypeRRSIG
	case "NSEC":
		return RRTypeNSEC
	case "DNSKEY":
		return RRTypeDNSKEY
	case "TLSA":
		return RRTypeTLSA
	case "SVCB":
		return RRTypeSVCB
	case "HTTPS":
		return RRTypeHTTPS
	case "ANY":
		return RRTypeANY
	case "CAA":
		return RRTypeCAA
	default:
		return 0 // invalid/unknown
	}
}
