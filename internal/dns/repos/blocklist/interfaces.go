package blocklist

import "github.com/haukened/rr-dns/internal/dns/domain"

// BloomSizer computes Bloom filter parameters from capacity (n) and target FP rate (p).
// It returns m (number of bits) and k (number of hash functions).
// Implemented in v0.3 task #30.
type BloomSizer interface {
	Size(n uint64, p float64) (m uint64, k uint8)
}

// BloomFilter is the minimal interface the repository needs from Bloom filters.
// Implementations may wrap exact/suffix filters separately.
type BloomFilter interface {
	Add(key []byte)
	MightContain(key []byte) bool
	Clear()
}

// DecisionCache caches block decisions by canonical name with basic metrics.
// Implemented in v0.3 task #33.
type DecisionCache interface {
	Get(name string) (domain.BlockDecision, bool)
	Put(name string, d domain.BlockDecision)
	Len() int
	Purge()
	Stats() (hits, misses, evictions uint64)
}

// StoreStats captures high-level counts and metadata for the persistent store.
type StoreStats struct {
	ExactCount  uint64
	SuffixCount uint64
	Version     uint64
	UpdatedUnix int64 // seconds since epoch
}

// Store abstracts the persistent index (planned Bolt backend in #31/#32).
// - ExistsExact: presence of an exact-domain key in the exact bucket
// - VisitSuffixes: iterate reversed-domain keys by prefix for suffix matching
// - Stats: counts and metadata; Close: release resources
type Store interface {
	ExistsExact(name string) (bool, error)
	VisitSuffixes(revPrefix []byte, visit func(key []byte) bool) error
	Stats() StoreStats
	Close() error
}

// RepoStats exposes repository-level counters and underlying store stats.
type RepoStats struct {
	Hits       uint64
	Misses     uint64
	Evictions  uint64
	Store      StoreStats
	LastUpdate int64 // seconds since epoch
}

// Repository is the composition layer that wires cache → bloom → store.
// Implemented in v0.3 task #34.
// Decide returns a value-type BlockDecision for the canonical name.
// Update rebuilds/writes/swaps the store, refreshes Bloom, and clears cache.
type Repository interface {
	Decide(name string) domain.BlockDecision
	Update() error
	RepoStats() RepoStats
}
