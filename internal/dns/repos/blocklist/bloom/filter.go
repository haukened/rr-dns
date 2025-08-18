package bloom

import (
	"sync"

	bitsbloom "github.com/bits-and-blooms/bloom/v3"
)

// filter wraps bits-and-blooms BloomFilter with a mutex for writes.
// Reads (MightContain) are safe concurrently; Add and Clear are serialized.
// This adapter stays inside repos layer to keep CLEAN boundaries.
type filter struct {
	mu sync.RWMutex
	bf *bitsbloom.BloomFilter
}

func (f *filter) Add(key []byte) {
	f.mu.Lock()
	f.bf.Add(key)
	f.mu.Unlock()
}

func (f *filter) MightContain(key []byte) bool {
	return f.bf.Test(key)
}
