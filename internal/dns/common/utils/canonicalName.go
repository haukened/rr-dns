package utils

import "strings"

// CanonicalDNSName returns a DNS name in canonical form:
// - Lowercased
// - Trimmed of surrounding whitespace
// - Ensured to end with a trailing dot
func CanonicalDNSName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)

	if name != "" && !strings.HasSuffix(name, ".") {
		name += "."
	}
	return name
}

// PresentationDNSName formats a DNS name for presentation purposes.
// It trims leading and trailing whitespace, converts the name to lowercase,
// and removes a trailing dot if present. Returns the processed DNS name.
// DNS Wire format doesn't have a trailing dot. RFC 4343 calls this "Presentation Format"
func PresentationDNSName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)

	if name != "" && strings.HasSuffix(name, ".") {
		name = name[:len(name)-1]
	}
	return name
}
