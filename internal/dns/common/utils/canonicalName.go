package utils

import "strings"

// CanonicalDNSName returns a DNS name in canonical form:
// - Lowercased
// - Trimmed of surrounding whitespace
// - No trailing dot because it doesn't add any runtime benefit, only legacy baggage.
func CanonicalDNSName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	// remove all trailing dots
	for strings.HasSuffix(name, ".") {
		name = strings.TrimSuffix(name, ".")
	}
	return name
}
