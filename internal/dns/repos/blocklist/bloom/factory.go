package bloom

import (
	bitsbloom "github.com/bits-and-blooms/bloom/v3"
	"github.com/haukened/rr-dns/internal/dns/repos/blocklist"
)

// factory implements blocklist.BloomFactory using internal sizing formulas.
type factory struct{}

// NewFactory returns a BloomFactory that sizes filters from capacity and FP rate.
func NewFactory() blocklist.BloomFactory { return factory{} }

// New constructs a new BloomFilter instance sized for the given dataset capacity
// and target false-positive rate.
func (factory) New(capacity uint64, fpRate float64) blocklist.BloomFilter {
	m, k := size(capacity, fpRate)
	return &filter{bf: bitsbloom.New(uint(m), uint(k))}
}
