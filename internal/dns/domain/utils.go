package domain

import (
	"golang.org/x/net/publicsuffix"
)

// generateCacheKey returns a consistent cache key derived from a DNS name, type, and class.
// The zone-aware format enables O(1) lookups by automatically extracting the zone root from the FQDN.
// Format: "zoneRoot|name|type|class" (e.g., "example.com.|www.example.com.|1|1")
// Uses pipe (|) separator to avoid conflicts with colons in IPv6 addresses and URIs.
func generateCacheKey(name string, t RRType, c RRClass) string {
	name = removeTrailingDot(name) // Ensure no trailing dot for consistent cache keys
	apexDomain, err := publicsuffix.EffectiveTLDPlusOne(name)
	if err != nil {
		apexDomain = name // Fallback to the original name if parsing fails
	}
	apexDomain = addTrailingDot(apexDomain) // Ensure apex domain ends with a dot
	return apexDomain + "|" + name + "|" + t.String() + "|" + c.String()
}

// removeTrailingDot removes a trailing dot from the given domain name string, if present.
// If the input string does not end with a dot, it is returned unchanged.
func removeTrailingDot(name string) string {
	if len(name) > 0 && name[len(name)-1] == '.' {
		return name[:len(name)-1]
	}
	return name
}

// addTrailingDot ensures that the provided domain name ends with a trailing dot.
// If the name does not already end with a dot, it appends one.
// This is useful for fully-qualified domain names (FQDNs) in DNS operations.
func addTrailingDot(name string) string {
	if len(name) > 0 && name[len(name)-1] != '.' {
		return name + "."
	}
	return name
}
