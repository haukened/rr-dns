package domain

import "fmt"

// RCode represents a DNS response code indicating the result of a query.
type RCode uint8

// IsValid returns true if the RCode is within the supported response code range.
func (r RCode) IsValid() bool {
	return r <= 10
}

// String returns the textual representation of the RCode.
func (r RCode) String() string {
	switch r {
	case 0:
		return "NOERROR"
	case 1:
		return "FORMERR"
	case 2:
		return "SERVFAIL"
	case 3:
		return "NXDOMAIN"
	case 4:
		return "NOTIMP"
	case 5:
		return "REFUSED"
	case 6:
		return "YXDOMAIN"
	case 7:
		return "YXRRSET"
	case 8:
		return "NXRRSET"
	case 9:
		return "NOTAUTH"
	case 10:
		return "NOTZONE"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", r)
	}
}

// ParseRCode converts a string name to an RCode value.
func ParseRCode(s string) RCode {
	switch s {
	case "NOERROR":
		return 0
	case "FORMERR":
		return 1
	case "SERVFAIL":
		return 2
	case "NXDOMAIN":
		return 3
	case "NOTIMP":
		return 4
	case "REFUSED":
		return 5
	case "YXDOMAIN":
		return 6
	case "YXRRSET":
		return 7
	case "NXRRSET":
		return 8
	case "NOTAUTH":
		return 9
	case "NOTZONE":
		return 10
	default:
		return 0
	}
}
