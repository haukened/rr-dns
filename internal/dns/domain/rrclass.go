package domain

// RRClass represents a DNS class (usually IN for Internet).
type RRClass uint16

// IsValid returns true if the RRClass is one of the supported classes.
func (c RRClass) IsValid() bool {
	switch c {
	case 1, 3, 4, 254, 255:
		return true
	default:
		return false
	}
}

// String returns the textual representation of the RRClass.
func (c RRClass) String() string {
	switch c {
	case 1:
		return "IN"
	case 3:
		return "CH"
	case 4:
		return "HS"
	case 254:
		return "NONE"
	case 255:
		return "ANY"
	default:
		return "UNKNOWN"
	}
}

// ParseRRClass converts a string name to an RRClass value.
func ParseRRClass(s string) RRClass {
	switch s {
	case "IN":
		return 1
	case "CH":
		return 3
	case "HS":
		return 4
	case "NONE":
		return 254
	case "ANY":
		return 255
	default:
		return 0
	}
}
