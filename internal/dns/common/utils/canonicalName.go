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
