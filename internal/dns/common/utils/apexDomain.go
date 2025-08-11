package utils

import "golang.org/x/net/publicsuffix"

func GetApexDomain(name string) string {
	name = CanonicalDNSName(name) // Ensure no trailing dot for consistent cache keys
	apexDomain, err := publicsuffix.EffectiveTLDPlusOne(name)
	if err != nil {
		apexDomain = name // Fallback to the original name if parsing fails
	}
	return apexDomain
}
