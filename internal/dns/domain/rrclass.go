package domain

// RRClass represents a DNS class (usually IN for Internet).
type RRClass uint16

// DNS Resource Record Class constants
const (
	RRClassIN   RRClass = 1   // IN - Internet
	RRClassCH   RRClass = 3   // CH - Chaos
	RRClassHS   RRClass = 4   // HS - Hesiod
	RRClassNONE RRClass = 254 // NONE - No class
	RRClassANY  RRClass = 255 // ANY - Any class (query only)
)

// IsValid returns true if the RRClass is one of the supported classes.
func (c RRClass) IsValid() bool {
	switch c {
	case RRClassIN, RRClassCH, RRClassHS, RRClassNONE, RRClassANY:
		return true
	default:
		return false
	}
}

// String returns the textual representation of the RRClass.
func (c RRClass) String() string {
	switch c {
	case RRClassIN:
		return "IN"
	case RRClassCH:
		return "CH"
	case RRClassHS:
		return "HS"
	case RRClassNONE:
		return "NONE"
	case RRClassANY:
		return "ANY"
	default:
		return "UNKNOWN"
	}
}

// ParseRRClass converts a string name to an RRClass value.
func ParseRRClass(s string) RRClass {
	switch s {
	case "IN":
		return RRClassIN
	case "CH":
		return RRClassCH
	case "HS":
		return RRClassHS
	case "NONE":
		return RRClassNONE
	case "ANY":
		return RRClassANY
	default:
		return 0
	}
}
