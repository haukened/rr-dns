package blocklist

// CacheStats reports lightweight cache metrics.
// All fields are best-effort snapshots and may be updated concurrently.
type CacheStats struct {
	Capacity  int    // configured capacity (0 for disabled cache)
	Size      int    // current number of entries
	Hits      uint64 // total cache hits since construction
	Misses    uint64 // total cache misses since construction
	Evictions uint64 // total evictions since construction
}

// StoreStats reports lightweight store metrics and metadata.
// Values are read from the store in a cheap, read-only transaction.
type StoreStats struct {
	Version     uint64 // snapshot version (0 if unknown)
	UpdatedUnix int64  // last updated unix time (0 if unknown)
	ExactKeys   uint64 // number of exact keys
	SuffixKeys  uint64 // number of suffix keys
}
