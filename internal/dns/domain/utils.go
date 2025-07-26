package domain

import "fmt"

// generateCacheKey returns a consistent cache key derived from a DNS name, type, and class.
func generateCacheKey(name string, t RRType, c RRClass) string {
	return fmt.Sprintf("%s:%d:%d", name, t, c)
}
