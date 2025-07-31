package domain

import (
	"github.com/haukened/rr-dns/internal/dns/common/utils"
)

// generateCacheKey returns a consistent cache key derived from a DNS name, type, and class.
// The zone-aware format enables O(1) lookups by automatically extracting the zone root from the FQDN.
// Format: "zoneRoot|name|type|class" (e.g., "example.com.|www.example.com.|1|1")
// Uses pipe (|) separator to avoid conflicts with colons in IPv6 addresses and URIs.
func generateCacheKey(name string, t RRType, c RRClass) string {
	// ensure the name is canonicalized and ends with a dot
	name = utils.CanonicalDNSName(name)
	// get the apex domain
	apexDomain := utils.GetApexDomain(name)
	// construct the cache key
	return apexDomain + "|" + name + "|" + t.String() + "|" + c.String()
}
