package bloom

import (
	"sync"

	bitsbloom "github.com/bits-and-blooms/bloom/v3"
	"github.com/haukened/rr-dns/internal/dns/repos/blocklist"
)

// filter wraps bits-and-blooms BloomFilter with a mutex for writes.
// Reads (MightContain) are safe concurrently; Add and Clear are serialized.
// This adapter stays inside repos layer to keep CLEAN boundaries.
type filter struct {
	mu sync.RWMutex
	bf *bitsbloom.BloomFilter
}

// NewFilter constructs a thread-safe BloomFilter given m and k.
func NewFilter(m uint64, k uint8) blocklist.BloomFilter {
	// bits-and-blooms exposes New(m, k)
	bf := bitsbloom.New(uint(m), uint(k))
	return &filter{bf: bf}
}

func (f *filter) Add(key []byte) {
	f.mu.Lock()
	f.bf.Add(key)
	f.mu.Unlock()
}

func (f *filter) MightContain(key []byte) bool {
	return f.bf.Test(key)
}

func (f *filter) Clear() {
	f.mu.Lock()
	f.bf.ClearAll()
	f.mu.Unlock()
}
