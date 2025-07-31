package utils

import "golang.org/x/net/publicsuffix"

func GetApexDomain(name string) string {
	name = removeTrailingDot(name) // Ensure no trailing dot for consistent cache keys
	apexDomain, err := publicsuffix.EffectiveTLDPlusOne(name)
	if err != nil {
		apexDomain = name // Fallback to the original name if parsing fails
	}
	return addTrailingDot(apexDomain) // Ensure apex domain ends with a dot
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
