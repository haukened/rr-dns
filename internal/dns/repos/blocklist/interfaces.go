package blocklist

import "github.com/haukened/rr-dns/internal/dns/domain"

// BloomFactory constructs Bloom filters sized for a dataset capacity and FP rate.
// Implementations may compute m/k internally; callers do not care about details.
type BloomFactory interface {
	New(capacity uint64, fpRate float64) BloomFilter
}

// BloomFilter is the minimal interface needed during lookups.
type BloomFilter interface {
	Add(key []byte)
	MightContain(key []byte) bool
}

// DecisionCache caches block decisions by canonical name.
type DecisionCache interface {
	Get(name string) (domain.BlockDecision, bool)
	Put(name string, d domain.BlockDecision)
	Len() int
	Purge()
	Stats() CacheStats
}

// Store persists and serves raw BlockRules.
// Reads use first-match semantics for performance:
// - prefer exact match; if found, short-circuit
// - else walk suffix anchors from most- to least-specific and stop at first match
// Writes are atomic, full-snapshot replacements (all sources aggregated).
type Store interface {
	// GetFirstMatch returns the highest-specificity rule matching name.
	// ok=false means no rule matched.
	GetFirstMatch(name string) (rule domain.BlockRule, ok bool, err error)

	// Atomic write path: replace all data in a single snapshot operation.
	// rules is the complete set across all sources.
	RebuildAll(rules []domain.BlockRule, version uint64, updatedUnix int64) error

	// Maintenance
	Purge() error
	Close() error
	Stats() StoreStats
}

// Repository wires cache → bloom → store.
// Decide returns a value-type BlockDecision for the canonical name.
// UpdateAll performs an atomic, all-sources snapshot update:
// - rebuilds the persistent store
// - recreates/resizes Bloom via a BloomFactory
// - clears the decision cache
type Repository interface {
	Decide(name string) domain.BlockDecision
	UpdateAll(rules []domain.BlockRule, version uint64, updatedUnix int64) error
	CacheStats() CacheStats
	StoreStats() StoreStats
}
